package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	totalConnections = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bgp_connections_total",
		Help: "The total number of connections",
	})

	aliveConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bgp_connections_alive",
		Help: "The number of live connections",
	})
)
