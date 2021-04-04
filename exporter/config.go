package exporter

import (
	"net"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	Asn                int
	Rid                net.IP
	ListenAddr         string
	MetricsPort        int
	DeleteOnDisconnect bool
}

func NewConfig(testConfig bool) Config {
	c := Config{}

	viper.SetDefault("asn", "64512")
	viper.SetDefault("RouterID", "1.1.1.1")
	viper.SetDefault("ListenAddr", "0.0.0.0")
	viper.SetDefault("MetricsPort", "9179")
	viper.SetDefault("DeleteOnDisconnect", false)

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
		log.Fatal("Router ID is invalid")
	}

	c.MetricsPort = viper.GetInt("MetricsPort")

	if _, err := net.ResolveTCPAddr("tcp", viper.GetString("ListenAddr")+":"+BGP_TCP_PORT); err != nil {
		log.Fatal("Listen addres is invalid: ", err)
	} else {
		c.ListenAddr = viper.GetString("ListenAddr")
	}

	c.DeleteOnDisconnect = viper.GetBool("DeleteOnDisconnect")

	return c
}
