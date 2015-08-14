package dynectProvider

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/docker-infra/go-dynect/dynect"
)

var (
	forceF = flag.Bool("force", false, "Try to delete CNAME record in case of error")
)

type DynectProvider struct {
	client *dynect.Client
}

type Rdata struct {
	Address string `json:"address"`
}
type ARecord struct {
	RecordData Rdata `json:"rdata"`
	Ttl        int   `json:"ttl"`
}
type CreateRecordRequest struct {
	Rdata Rdata `json:"rdata"`
	Ttl   int   `json:"ttl"`
}

type UpdateRecordRequest struct {
	ARecords []ARecord `json:"ARecords"`
}

type Publish struct {
	Publish string `json:"publish"`
}

func NewProvider() (provider *DynectProvider, err error) {
	customer := os.Getenv("DYNECT_CUSTOMER")
	username := os.Getenv("DYNECT_USERNAME")
	password := os.Getenv("DYNECT_PASSWORD")

	client, err := dynect.New(customer, username, password)
	if err != nil {
		return
	}
	provider = &DynectProvider{client: client}
	return
}

func (provider *DynectProvider) setARecords(zone, fqdn, ip string) (result bool, err error) {
	result = false
	updateReq := &UpdateRecordRequest{
		[]ARecord{
			{Rdata{ip}, 0},
		},
	}
	zreqBytes, err := json.Marshal(updateReq)
	if err != nil {
		return
	}
	path := fmt.Sprintf("ARecord/%s/%s/", zone, fqdn)
	responseBytes, err := provider.client.Request("PUT", path, bytes.NewReader(zreqBytes))
	if err != nil {
		return
	}
	response := &dynect.Response{}
	if err = json.Unmarshal(responseBytes, response); err != nil {
		return
	}
	if response.Status == "success" {
		result = true
		return
	} else {
		err = errors.New(fmt.Sprintf("The request failed: %s", string(responseBytes)))
		return
	}
}

func (provider *DynectProvider) DeleteCName(zone, fqdn string) (result bool, err error) {
	result = false
	path := fmt.Sprintf("CNAMERecord/%s/%s/", zone, fqdn)
	responseBody, err := provider.client.Request("DELETE", path, nil)
	if err != nil {
		return
	}
	response := &dynect.Response{}
	if err = json.Unmarshal(responseBody, response); err != nil {
		return
	}
	if response.Status == "success" {
		result = true
		return
	} else {
		err = fmt.Errorf("Can't Delete CNAME entry for %s. Response: %s", fqdn, string(responseBody))
		return
	}
}

func (provider *DynectProvider) publishChanges(zone string) (result bool, err error) {
	result = false
	publishRequest := &Publish{Publish: "True"}
	publishRequestBytes, err := json.Marshal(publishRequest)
	if err != nil {
		return
	}
	publishPath := fmt.Sprintf("Zone/%s", zone)
	prResultBytes, err := provider.client.Request("PUT", publishPath, bytes.NewReader(publishRequestBytes))
	if err != nil {
		return
	}
	prResult := &dynect.Response{}
	if err = json.Unmarshal(prResultBytes, prResult); err != nil {
		return
	}
	if prResult.Status == "success" {
		result = true
		return
	} else {
		err = errors.New(fmt.Sprintf("Publishing changes failed: %s", string(prResultBytes)))
		return
	}
}

func (provider *DynectProvider) UpdateARecord(zone, fqdn, ip string, force bool) (result bool, err error) {
	result = false
	result, err = provider.setARecords(zone, fqdn, ip)
	if err != nil {
		if *forceF {
			result, err = provider.DeleteCName(zone, fqdn)
			if err != nil {
				return
			}
		} else {
			return
		}
	}
	result, err = provider.publishChanges(zone)
	return
}
