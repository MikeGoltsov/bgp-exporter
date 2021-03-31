package main

import (
	"log"
	"net"
	"os"
	"strconv"
)

type config struct {
	Asn       int
	Addr      string
	prom_port int
	rid       net.IP
}

func newConfig(testConfig bool) config {
	const prefix = "BGPEX_"
	c := config{}

	//ASN
	if i := os.Getenv(prefix + "ASN"); len(i) > 0 {
		d, err := strconv.Atoi(i)
		if err != nil {
			log.Panic(err, "ASN is invalid")
		}
		c.Asn = d
	} else {
		c.Asn = 64512
	}

	//BGP Router ID
	if rid := os.Getenv(prefix + "RID"); len(rid) > 0 {
		c.rid = net.ParseIP(rid)
		if c.rid.To4() == nil {
			log.Panic("Router ID is invalid")
		}
	} else {
		c.rid = net.ParseIP("1.1.1.1")
	}

	//LISTEN ADDR
	if addr := os.Getenv(prefix + "ADDR"); len(addr) > 0 {
		if _, err := net.ResolveTCPAddr("tcp", addr); err != nil {
			log.Panic(err, "ADDR is invalid")
		}
		c.Addr = addr
	} else {
		c.Addr = "0.0.0.0"
	}

	//PROMETHEUS PORT
	if i := os.Getenv(prefix + "PROMETHEUS_PORT"); len(i) > 0 {
		d, err := strconv.Atoi(i)
		if err != nil {
			log.Panic(err, "Port is invalid")
		}
		c.prom_port = d
	} else {
		c.prom_port = 9179
	}

	return c
}
