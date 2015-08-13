package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"

	"github.com/docker-infra/go-dynect/dynect"
)

type Rdata struct {
	Address string `json:"address"`
}
type UpdateZoneRequest struct {
	Rdata Rdata `json:"rdata"`
	Ttl   int   `json:"ttl"`
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
			log.Printf("Hooray, we found a valid first line in the hosts file with IP %s", result[1])
			return result[1], nil
		}
	}
	err = errors.New("Didn't find a valid line in hosts file")
	return
}

func main() {
	customer := os.Getenv("DYNECT_CUSTOMER")
	username := os.Getenv("DYNECT_USERNAME")
	password := os.Getenv("DYNECT_PASSWORD")
	log.Printf("Using username %s, Customer %s, Password %s ", username, customer, password)
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
	zoneReq := &UpdateZoneRequest{
		Rdata: Rdata{
			Address: publicIp,
		},
		Ttl: 0,
	}
	zreqBytes, err := json.Marshal(zoneReq)
	if err != nil {
		log.Panicf("Can't marshal zone request: %v", err)
	}

	path := fmt.Sprintf("ARecord/%s/%s/", zone, fqdn)
	_, err = client.Request("POST", path, bytes.NewReader(zreqBytes))
	if err != nil {
		log.Panicf("Failed to update FQDN %s for this host: %v", fqdn, err)
	} else {
		log.Printf("Updated FQDN %s with A record for IP %s", fqdn, publicIp)
	}
}
