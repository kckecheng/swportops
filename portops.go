package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/soniah/gosnmp"
)

// Port identifier
type Port string

// OID port OID
type OID string

// SWConn switch SNMP connection
type SWConn struct {
	Address   string
	Community string
	SNMPConn  *gosnmp.GoSNMP
	Ports     map[Port]OID
}

// NewConn create a connection to the switch
func NewConn(address, community string) (*SWConn, error) {
	gosnmp.Default.Target = address
	gosnmp.Default.Community = community
	err := gosnmp.Default.Connect()
	if err != nil {
		return nil, err
	}
	return &SWConn{Address: address, Community: community, SNMPConn: gosnmp.Default}, nil
}

// GetSysname get the system name of the switch
func (sw *SWConn) GetSysname() (string, error) {
	oids := []string{"1.3.6.1.2.1.1.5.0"}
	result, err := gosnmp.Default.Get(oids)
	if err != nil {
		return "", err
	}

	bytes := result.Variables[0].Value.([]byte)
	return string(bytes), nil
}

// GetPorts get the interfaces
func (sw *SWConn) GetPorts() error {
	oid := "1.3.6.1.2.1.31.1.1.1.1"
	results, err := sw.SNMPConn.WalkAll(oid)
	if err != nil {
		return err
	}

	ports := make(map[Port]OID)
	for _, result := range results {
		parts := strings.Split(result.Name, ".")
		oid := ".1.3.6.1.2.1.2.2.1.7." + parts[len(parts)-1]
		bytes := result.Value.([]byte)
		port := string(bytes)
		ports[Port(port)] = OID(oid)
	}
	sw.Ports = ports
	return nil
}

// PortCfg online/offline a port. Raw commands are as below:
// Online : snmpset -v 2c -c private 192.168.1.1 1.3.6.1.2.1.2.2.1.7.16793600 i 1
// Offline: snmpset -v 2c -c private 192.168.1.1 1.3.6.1.2.1.2.2.1.7.16793600 i 2
func (sw *SWConn) PortCfg(port Port, ops string) error {
	oid, ok := sw.Ports[port]
	if !ok {
		return fmt.Errorf("Cannot find port %s", port)
	}

	var value int
	switch ops {
	case "on":
		value = 1
	case "off":
		value = 2
	default:
		return fmt.Errorf("Not supported operation %s", ops)
	}
	payload := gosnmp.SnmpPDU{Name: string(oid), Type: gosnmp.Integer, Value: value}
	result, err := sw.SNMPConn.Set([]gosnmp.SnmpPDU{payload})
	if err != nil {
		return err
	}

	if result.Error != gosnmp.NoError {
		return fmt.Errorf("SNMP error %d", result.Error)
	}
	return nil
}

func main() {
	args := os.Args[1:]
	nargs := len(args)
	address := ""
	community := "private"
	if nargs == 0 || nargs > 2 {
		fmt.Printf("Usage: ./portops <switch address> [community string]")
		os.Exit(1)
	} else if nargs == 1 {
		address = args[0]
	} else {
		address = args[0]
		community = args[1]
	}
	log.Printf("Connect to switch %s with community string %s\n", address, community)

	sw, err := NewConn(address, community)
	if err != nil {
		log.Fatal(err)
	}
	defer sw.SNMPConn.Conn.Close()

	log.Println("Get switch sysname ...")
	swname, err := sw.GetSysname()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Switch name: %s\n", swname)

	log.Println("Get switch ports ...")
	err = sw.GetPorts()
	if err != nil {
		log.Fatal(err)
	}
	for k, v := range sw.Ports {
		fmt.Printf("Port Name: %s - Port Index OID: %s\n", k, v)
	}

	log.Printf("Disable port fc1/5 ...\n")
	sw.PortCfg(Port("fc1/5"), "off")
	log.Printf("Enable port fc1/5 ...\n")
	sw.PortCfg(Port("fc1/5"), "on")
}
