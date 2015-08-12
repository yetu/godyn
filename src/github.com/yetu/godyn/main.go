package main

import (
  "github.com/docker-infra/go-dynect/dynect"
  "log"
  "os"
  "fmt"
  "encoding/json"
  "bytes"
)
type Rdata struct {
  Address string `json:address`
}
type UpdateZoneRequest struct {
  Rdata Rdata `json:"rdata"`
  Ttl int `json:"ttl"`
}

func main() {
  customer := os.Getenv("DYNECT_CUSTOMER")
  username := os.Getenv("DYNECT_USERNAME")
  password := os.Getenv("DYNECT_PASSWORD")
  zone := os.Getenv("DYNECT_ZONE")
  fqdn := os.Getenv("DYNECT_FQDN")

  client, err := dynect.New(customer,username,password)
  if err != nil {
    log.Panicf("Can't create dynect client: %v", err)
  }
  // TODO determine public ip
  zoneReq := &UpdateZoneRequest{
    Rdata: Rdata {
      Address: "10.10.10.10",
    },
    Ttl: 0,
  }
  zreqBytes, err := json.Marshal(zoneReq)
  if err != nil {
    log.Panicf("Can't marshal zone request: %v", err)
  }

  path := fmt.Sprintf("ARecord/%s/%s/",zone,fqdn)
  client.Request("POST",path,bytes.NewReader(zreqBytes))
}
