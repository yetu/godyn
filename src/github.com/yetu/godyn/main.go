package main

import (
	"bufio"
	"errors"
	"flag"
	"log"
	"os"
	"regexp"

	"github.com/yetu/godyn/dns"
	"github.com/yetu/godyn/dynectProvider"
)

var dnsProvider dns.Provider

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
	zone := os.Getenv("GODYN_ZONE")
	fqdn := os.Getenv("GODYN_FQDN")
	publicIp, err := getPublicIpFromHosts()
	if err != nil {
		log.Panicf("Can't determine public IP for this container: %v", err)
	}
	log.Printf("Trying to update A record for %s to current container IP %s", fqdn, publicIp)
	dnsProvider, err = dynectProvider.NewProvider()
	if err != nil {
		log.Panicf("Can't create DNS provider: %v", err)
	}
	_, err = dnsProvider.UpdateARecord(zone, fqdn, publicIp, false)
	if err != nil {
		log.Fatalf("Failed to update A record: %v", err)
	} else {
		log.Printf("Successfully updated FQDN %s with A record for %s", fqdn, publicIp)
	}
}
