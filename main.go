package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

func BgpThread(cfg config) {
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
		go handlePeer(conn, cfg)
	}
}

func main() {
	cfg := newConfig(false)
	log.SetFormatter(&log.TextFormatter{FullTimestamp: false})
	log.SetLevel(log.DebugLevel)
	log.Printf("App Started")
	fmt.Println(cfg)

	prometheus.MustRegister(routes)
	http.Handle("/metrics", promhttp.Handler())
	//	http.ListenAndServe(":"+strconv.Itoa(cfg.prom_port), nil)

	go func() {
		if err := http.ListenAndServe(":"+strconv.Itoa(cfg.prom_port), nil); err != nil {
			if err != http.ErrServerClosed {
				log.Fatal("Server crashed")
			}
		}
	}()

	go BgpThread(cfg)

	log.Printf("App running")
	//wait for OS signal
	c := make(chan os.Signal, 2)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	<-c

}
