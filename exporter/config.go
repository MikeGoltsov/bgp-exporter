package exporter

import (
	"log"
	"net"
	"os"
	"strconv"

	"github.com/spf13/viper"
)

type Config struct {
	Asn       int
	Addr      string
	Prom_port int
	Rid       net.IP
}

func NewConfig(testConfig bool) Config {
	const prefix = "BGPEX_"
	c := Config{}

	viper.SetDefault("ASN", "64512")

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
		c.Rid = net.ParseIP(rid)
		if c.Rid.To4() == nil {
			log.Panic("Router ID is invalid")
		}
	} else {
		c.Rid = net.ParseIP("1.1.1.1")
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
		c.Prom_port = d
	} else {
		c.Prom_port = 9179
	}

	c.Asn = viper.GetInt("ASN")
	return c
}
