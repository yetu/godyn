package dynect

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"time"
)

const (
	apiURL      = "https://api.dynect.net/REST/"
	contentType = "application/json"
)

var (
	debugF = flag.Bool("debug", false, "Dump unsanitized(!) requests for debugging purposes")
)

// Response is the common response struct
type Response struct {
	JobID    int    `json:"job_id"`
	Status   string `json:"status"`
	Messages []struct {
		Source    string `json:"SOURCE"`
		Level     string `json:"LVL"`
		ErrorCode string `json:"ERR_CD"`
		Info      string `json:"INFO"`
	} `json:"msgs"`
}

type sessionRequest struct {
	Customer string `json:"customer_name"`
	User     string `json:"user_name"`
	Password string `json:"password"`
}

type sessionResponse struct {
	Response
	Data struct {
		Version string `json:"version"`
		Token   string `json:"token"`
	} `json:"data"`
}

// Client is the dynect client
type Client struct {
	client        *http.Client
	headers       map[string]string
	retryInterval time.Duration
}

// New logs in to dynECT api, sets the auth token and returns the client
func New(customer, user, password string) (*Client, error) {
	c := &Client{
		headers: map[string]string{
			"Content-Type": contentType,
		},
		retryInterval: 1 * time.Second,
	}
	c.client = &http.Client{
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			debug("Got redirect, req: %#v", req)
			c.setHeaders(req)
			return nil
		},
	}
	sreq := &sessionRequest{
		Customer: customer,
		User:     user,
		Password: password,
	}
	sreqBytes, err := json.Marshal(sreq)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Post(apiURL+"Session/", contentType, bytes.NewReader(sreqBytes))
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	sres := &sessionResponse{}
	if err := json.Unmarshal(body, sres); err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		if len(body) == 0 {
			return nil, fmt.Errorf("Error: %s", resp.Status)
		}
		return nil, fmt.Errorf("%s: %s", sres.Messages[0].Level, sres.Messages[0].Info)
	}
	c.headers["Auth-Token"] = sres.Data.Token
	return c, nil
}

// Request sends a request to dynect api and returns the responses body
func (c *Client) Request(method, path string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, apiURL+path, body)
	if err != nil {
		return nil, err
	}

	c.setHeaders(req)
	if *debugF {
		dump, err := httputil.DumpRequest(req, true)
		if err != nil {
			log.Printf("Couldn't dump request: %s", err)
		} else {
			log.Println(string(dump))
		}
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		dump, err := httputil.DumpResponse(resp, true)
		if err != nil {
			return nil, fmt.Errorf("Error %s", resp.Status)
		}
		return nil, fmt.Errorf("Error %s: %s", resp.Status, dump)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return respBody, err
	}
	rs := &Response{}
	if err := json.Unmarshal(respBody, rs); err != nil {
		return respBody, err
	}

	if rs.Status == "incomplete" {
		debug("incomplete request, retrying in %s", c.retryInterval)
		resp.Body.Close()
		time.Sleep(c.retryInterval)
		return c.Request("GET", resp.Request.URL.Path, nil) // Poll URL of latest redirect
	}
	return respBody, err
}

// Execute sends a request to dynect api and returns a Response. Use this is you're not
// interested in the response but just if the request succeeded
func (c *Client) Execute(method, path string, body io.Reader) error {
	respBody, err := c.Request(method, path, body)
	if err != nil {
		return err
	}

	dynResp := &Response{}
	if err := json.Unmarshal(respBody, dynResp); err != nil {
		return fmt.Errorf("Couldn't unmarshal:\n\t%s\nError: %s", respBody, err)
	}

	if dynResp.Status != "success" {
		return fmt.Errorf("%s: %s", dynResp.Messages[0].Level, dynResp.Messages[0].Info)
	}
	return nil
}

// SetRetryInterval sets the time to wait between retries of incomplete jobs
func (c *Client) SetRetryInterval(interval time.Duration) {
	c.retryInterval = interval
}

func (c *Client) setHeaders(req *http.Request) {
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
}

func debug(fs string, args ...interface{}) {
	if *debugF {
		log.Printf(fs, args...)
	}
}
