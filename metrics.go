package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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
	route_change = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "bgp_route_changes",
		Help: "Number of blob storage operations waiting to be processed, partitioned by user and type.",
	}, []string{"peer", "route", "aspath"},
	)
)
