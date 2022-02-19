package exporter

import (
	"net"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const defaultLogLevel = log.InfoLevel

// Config global configuration of exporter.
type Config struct {
	Asn                int
	Rid                net.IP
	ListenAddr         string
	MetricsPort        int
	DeleteOnDisconnect bool
	LogLevel           log.Level
}

// parseFlags parses configuration from config file or env.
func parseFlags() {
	var configPath string

	pflag.StringVarP(&configPath, "config", "c", "", "Config file path")
	pflag.IntP("asn", "a", 64512, "AS number of exporter")
	pflag.StringP("listen-address", "l", "0.0.0.0", "listen adress")
	pflag.Parse()

	viper.SetDefault("asn", "64512")
	viper.SetDefault("router-id", "1.1.1.1")
	viper.SetDefault("listen-address", "0.0.0.0")
	viper.SetDefault("metrics-port", "9179")
	viper.SetDefault("clear-neighbour", false)
	viper.SetDefault("log-level", "info")

	if configPath != "" {
		log.Infof("Parsing config: %s", configPath)
		viper.SetConfigFile(configPath)
		err := viper.ReadInConfig()
		if err != nil {
			log.Error("Unable to read config file: ", err)
		}
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
	viper.SetEnvPrefix("bgpexp")

	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		log.Fatal("Unable to parse command line: ", err)
	}
}

// configure read and checks configuration from config file or env.
func Configure() Config {
	parseFlags()

	ll, err := log.ParseLevel(viper.GetString("log-level"))
	if err != nil {
		log.Error("Invalid log level", err)
		log.Warn("Using default log level", defaultLogLevel)
		ll = defaultLogLevel
	}

	c := Config{
		LogLevel:           ll,
		Asn:                viper.GetInt("asn"),
		MetricsPort:        viper.GetInt("metrics-port"),
		DeleteOnDisconnect: viper.GetBool("clear-neighbour"),
		ListenAddr:         viper.GetString("listen-address"),
	}

	c.Rid = net.ParseIP(viper.GetString("router-id"))
	if c.Rid.To4() == nil {
		log.Fatal("Router ID is invalid")
	}

	if _, err := net.ResolveTCPAddr("tcp", c.ListenAddr+":"+BGP_TCP_PORT); err != nil {
		log.Fatal("Listen address is invalid: ", err)
	}

	return c
}
