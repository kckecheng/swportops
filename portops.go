package main

import (
	"fmt"
	"log"

	g "github.com/soniah/gosnmp"
)

// SWConn switch SNMP connection
type SWConn struct {
	Address   string
	Community string
	SNMPConn  *g.GoSNMP
}

// NewConn create a connection to the switch
func NewConn(address, community string) (*SWConn, error) {
	g.Default.Target = address
	err := g.Default.Connect()
	if err != nil {
		return nil, err
	}
	return &SWConn{Address: address, Community: community, SNMPConn: g.Default}, nil
}

// GetSysname get the system name of the switch
func (sw *SWConn) GetSysname() (string, error) {
	oids := []string{"1.3.6.1.2.1.1.5.0"}
	result, err := g.Default.Get(oids)
	if err != nil {
		return "", err
	}

	bytes := result.Variables[0].Value.([]byte)
	return string(bytes), nil
}

// GetPorts get the interfaces
func (sw *SWConn) GetPorts() (map[string]string, error) {
	oid := "1.3.6.1.2.1.31.1.1.1.1"
	results, err := sw.SNMPConn.WalkAll(oid)
	if err != nil {
		return nil, err
	}

	ports := make(map[string]string)
	for _, result := range results {
		oid := result.Name
		bytes := result.Value.([]byte)
		port := string(bytes)
		ports[oid] = port
	}
	return ports, nil
}

func main() {
	address := "10.228.225.202"
	community := "public"

	sw, err := NewConn(address, community)
	if err != nil {
		log.Fatal(err)
	}
	defer sw.SNMPConn.Conn.Close()

	swname, err := sw.GetSysname()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Switch name: %s\n", swname)

	ports, err := sw.GetPorts()
	if err != nil {
		log.Fatal(err)
	}
	for k, v := range ports {
		fmt.Printf("Port OID: %s - Port Name: %s\n", k, v)
	}
}
