package exporter

import (
	"log"
	"net"

	"github.com/spf13/viper"
)

type Config struct {
	Asn       int
	Addr      string
	Prom_port int
	Rid       net.IP
}

func NewConfig(testConfig bool) Config {
	c := Config{}

	viper.SetDefault("asn", "64512")
	viper.SetDefault("RouterID", "1.1.1.1")
	viper.SetDefault("ListenAddr", "0.0.0.0")
	viper.SetDefault("PrometheusPort", "9179")

	viper.SetConfigName("bgp-exporter")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/")
	viper.AddConfigPath(".")
	viper.ReadInConfig()

	viper.AutomaticEnv()
	viper.SetEnvPrefix("bgpexp")

	c.Asn = viper.GetInt("asn")

	c.Rid = net.ParseIP(viper.GetString("RouterID"))
	if c.Rid.To4() == nil {
		log.Panic("Router ID is invalid")
	}

	c.Prom_port = viper.GetInt("PrometheusPort")

	if _, err := net.ResolveTCPAddr("tcp", viper.GetString("ListenAddr")+":179"); err != nil {
		log.Panic(err, "Listen addres is invalid")
	} else {
		c.Addr = viper.GetString("ListenAddr")
	}

	return c
}
