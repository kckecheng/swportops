package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/soniah/gosnmp"
)

// Port switch port and OID mapping
type Port struct {
	Name string `json:"port"`
	OID  string `json:"oid"`
}

// HTTPMsg HTTP response message
type HTTPMsg struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// SWConn switch SNMP connection
type SWConn struct {
	Address   string
	Community string
	SNMPConn  *gosnmp.GoSNMP
}

// NewConn create a connection to the switch
func NewConn(address, community string) (*SWConn, error) {
	gosnmp.Default.Target = address
	gosnmp.Default.Community = community
	err := gosnmp.Default.Connect()
	if err != nil {
		msg := fmt.Sprintf("Fail to connect to %s with community string %s", address, community)
		log.Println(msg)
		log.Printf("Internal error: %s", err.Error())
		return nil, errors.New(msg)
	}
	return &SWConn{Address: address, Community: community, SNMPConn: gosnmp.Default}, nil
}

// GetSysname get the system name of the switch
func (sw *SWConn) GetSysname() (string, error) {
	oids := []string{"1.3.6.1.2.1.1.5.0"}
	result, err := gosnmp.Default.Get(oids)
	if err != nil {
		msg := fmt.Sprintf("Fail to get sysname due to %s", err.Error())
		log.Println(msg)
		return "", errors.New(msg)
	}

	bytes := result.Variables[0].Value.([]byte)
	return string(bytes), nil
}

// GetPorts get the interfaces
func (sw *SWConn) GetPorts() ([]Port, error) {
	oid := "1.3.6.1.2.1.31.1.1.1.1"
	results, err := sw.SNMPConn.WalkAll(oid)
	if err != nil {
		msg := fmt.Sprintf("Fail to get ports due to %s", err.Error())
		log.Println(msg)
		return nil, errors.New(msg)
	}

	ports := []Port{}
	for _, result := range results {
		parts := strings.Split(result.Name, ".")
		oid := ".1.3.6.1.2.1.2.2.1.7." + parts[len(parts)-1]
		bytes := result.Value.([]byte)
		name := string(bytes)
		port := Port{Name: name, OID: oid}
		ports = append(ports, port)
	}
	return ports, nil
}

// PortCfg online/offline a port. Raw commands are as below:
// Online : snmpset -v 2c -c private 192.168.1.1 1.3.6.1.2.1.2.2.1.7.16793600 i 1
// Offline: snmpset -v 2c -c private 192.168.1.1 1.3.6.1.2.1.2.2.1.7.16793600 i 2
func (sw *SWConn) PortCfg(oid, ops string) error {
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
		msg := fmt.Sprintf("Fail to %s port with OID %s due to %s", ops, oid, err.Error())
		log.Println(msg)
		return errors.New(msg)
	}

	if result.Error != gosnmp.NoError {
		return fmt.Errorf("SNMP error with code %d", result.Error)
	}
	return nil
}

func homeLink(w http.ResponseWriter, r *http.Request) {
	welcome := `
	<html>
	<title>Switch Port Ops</title>
	</html>
	<body>
	<h3>List all switch ports and their OID</h3>
	GET /ports?switch=&#60;switch address&#62;[&community=&#60;community string, private is the default&#62;]
	<h3>On/Off a switch port based on its port OID</h3>
	GET /port?switch=&#60;switch address&#62;[&community=&#60;community string, private is the default&#62;]&oid=&#60;port OID&#62;&ops=&#60;on|off&#62;
	</body>
	`
	fmt.Fprintf(w, welcome)
}

func initSWConn(r *http.Request) (*SWConn, error) {
	params := r.URL.Query()
	address := params.Get("switch")

	community := params.Get("community")
	if community == "" {
		community = "private"
	}

	sw, err := NewConn(address, community)
	if err != nil {
		return nil, err
	}
	return sw, nil
}

func processBadRequest(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	w.WriteHeader(http.StatusBadRequest)
	msg, _ := json.Marshal(HTTPMsg{400, err.Error()})
	w.Write(msg)
}

func portsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	sw, err := initSWConn(r)
	if err != nil {
		processBadRequest(w, err)
		return
	}
	defer sw.SNMPConn.Conn.Close()

	ports, err := sw.GetPorts()
	if err != nil {
		msg := fmt.Sprintf("Cannot list ports for switch %s with community string %s due to %s", sw.Address, sw.Community, err.Error())
		newErr := errors.New(msg)
		processBadRequest(w, newErr)
		return
	}

	w.WriteHeader(http.StatusOK)
	payload, _ := json.Marshal(ports)
	w.Write(payload)
	return
}

func opsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	sw, err := initSWConn(r)
	if err != nil {
		processBadRequest(w, err)
		return
	}
	defer sw.SNMPConn.Conn.Close()

	params := r.URL.Query()
	oid := params.Get("oid")
	ops := params.Get("ops")
	if oid == "" || !strings.Contains(oid, ".1.3.6.1.2.1.2.2.1.7.") || (ops != "on" && ops != "off") {
		msg := fmt.Sprintf("OID %s or ops %s is not provided or valid", oid, ops)
		processBadRequest(w, errors.New(msg))
		return
	}

	err = sw.PortCfg(oid, ops)
	if err != nil {
		processBadRequest(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	msg, _ := json.Marshal(HTTPMsg{200, fmt.Sprintf("Successfully %s port with OID %s", ops, oid)})
	w.Write(msg)
	return
}

func main() {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", homeLink).Methods("GET")

	router.HandleFunc("/ports", portsHandler).Methods("GET")
	router.HandleFunc("/port", opsHandler).Methods("GET")

	log.Println("Start HTTP server on port 8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
