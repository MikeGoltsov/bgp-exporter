package main

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	"bgp-exporter/exporter"

	log "github.com/sirupsen/logrus"
)

func BgpThread(cfg exporter.Config) {
	l, err := net.Listen("tcp", cfg.Addr+":179")
	if err != nil {
		log.Fatal("Error listening:", err.Error())
	}
	// Close the listener when the application closes.
	defer l.Close()
	log.Info("Listening on " + cfg.Addr + ":179")
	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			log.Error("Error accepting: ", err.Error())
		}
		// Handle connections in a new goroutine.
		go exporter.HandlePeer(conn, cfg)
	}
}

func main() {
	cfg := exporter.NewConfig(false)
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	log.SetLevel(log.DebugLevel)
	log.Info("App Starting")

	go exporter.StartMetricsServer(cfg)

	go BgpThread(cfg)

	log.Info("App running")
	//wait for OS signal
	c := make(chan os.Signal, 2)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	<-c
	log.Info("Exit by Signal")
}
