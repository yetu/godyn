package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"

	"github.com/docker-infra/go-dynect/dynect"
)

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

/*
This tries to parse the file /etc/hosts inside the container and extract the first
defined host. We hope that this is always the public IP address
*/
func getPublicIpFromHosts() (ip string, err error) {
	hostRegex, _ := regexp.Compile(`(\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b)\s*(.*)`)
	hostsFile, err := os.Open("/etc/hosts")
	if err != nil {
		return
	}
	defer hostsFile.Close()
	scanner := bufio.NewScanner(hostsFile)
	for scanner.Scan() {
		line := scanner.Text()
		if hostRegex.MatchString(line) {
			result := hostRegex.FindStringSubmatch(line)
			ip = result[1]
			err = nil
			return result[1], nil
		}
	}
	err = errors.New("Didn't find a valid line in hosts file")
	return
}

func main() {
	flag.Parse()
	customer := os.Getenv("DYNECT_CUSTOMER")
	username := os.Getenv("DYNECT_USERNAME")
	password := os.Getenv("DYNECT_PASSWORD")
	zone := os.Getenv("DYNECT_ZONE")
	fqdn := os.Getenv("DYNECT_FQDN")

	client, err := dynect.New(customer, username, password)
	if err != nil {
		log.Panicf("Can't create dynect client: %v", err)
	}
	publicIp, err := getPublicIpFromHosts()
	if err != nil {
		log.Panicf("Can't determine public IP for this container: %v", err)
	}
	/*zoneReq := &CreateRecordRequest{
		Rdata: Rdata{
			Address: publicIp,
		},
		Ttl: 0,
	}*/

	updateReq := &UpdateRecordRequest{
		[]ARecord{
			{Rdata{publicIp}, 0},
		},
	}
	zreqBytes, err := json.Marshal(updateReq)
	//log.Printf("Request Payload: %s", string(zreqBytes))
	if err != nil {
		log.Panicf("Can't marshal zone request: %v", err)
	}

	path := fmt.Sprintf("ARecord/%s/%s/", zone, fqdn)
	responseBytes, err := client.Request("PUT", path, bytes.NewReader(zreqBytes))
	if err != nil {
		log.Panicf("Failed to update FQDN %s for this host: %v", fqdn, err)
	} else {
		responseString := string(responseBytes)
		response := &dynect.Response{}
		if err := json.Unmarshal(responseBytes, response); err != nil {
			log.Panicf("Can't unmarshall response from dynect: %s Error %v", responseString, err)
		}
		if response.Status == "success" {
			log.Printf("Successfully updated FQDN %s to IP %s", fqdn, publicIp)
			publishRequest := &Publish{Publish: "True"}
			publishRequestBytes, err := json.Marshal(publishRequest)
			if err != nil {
				log.Panicf("Can't marshal publish zone request: %v", err)
			}
			publishPath := fmt.Sprintf("Zone/%s", zone)
			prResultBytes, err := client.Request("PUT", publishPath, bytes.NewReader(publishRequestBytes))
			if err != nil {
				log.Panicf("Can't publish changes: %v", err)
			} else {
				prResult := &dynect.Response{}
				if err := json.Unmarshal(prResultBytes, prResult); err != nil {
					log.Panicf("Can't unmarshall response from dynect. Error %v Response %s", err, string(prResultBytes))
				}
				if prResult.Status == "success" {
					log.Printf("Successfully published changes to Dynect")
				} else {
					log.Panicf("Failed to publish changes to Dynect: %s", string(prResultBytes))
				}
			}
		} else {
			log.Panicf("Failed to modify zone in Dynect. Response: %s", responseString)
		}
	}
}
