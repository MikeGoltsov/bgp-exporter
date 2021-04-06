package exporter

import (
	"log"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	myasn = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "bgp_local_asn",
		Help: "Local BGP ASN number",
	}, []string{"asn"})

	totalConnections = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bgp_connections_total",
		Help: "The total number of connections",
	})

	aliveConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bgp_connections_alive",
		Help: "The number of live connections",
	})
	routes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "bgp_route",
		Help: "Number of blob storage operations waiting to be processed, partitioned by user and type.",
	}, []string{"peer", "route", "aspath"},
	)
	routeChange = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "bgp_route_changes",
		Help: "Number of blob storage operations waiting to be processed, partitioned by user and type.",
	}, []string{"peer", "route", "aspath"},
	)
)

//StartMetricsServer init and run web server
func StartMetricsServer(cfg *Config) {
	myasn.WithLabelValues(strconv.Itoa(cfg.Asn)).Inc()

	prometheus.MustRegister(routes)
	prometheus.MustRegister(routeChange)
	http.Handle("/metrics", promhttp.Handler())

	if err := http.ListenAndServe(":"+strconv.Itoa(cfg.MetricsPort), nil); err != nil {
		if err != http.ErrServerClosed {
			log.Fatal("Server crashed")
		}
	}

}
