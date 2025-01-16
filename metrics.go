package main

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// Prometheus metrics
var (
	dnsQueriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dns_queries_total",
			Help: "Total number of DNS queries.",
		},
		[]string{"interval"},
	)
	dnsResponsesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dns_responses_total",
			Help: "Total number of DNS responses.",
		},
		[]string{"interval"},
	)
	dnsTopSourceIPs = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "dns_top_source_ips",
			Help: "Top 10 source IPs by query volume.",
		},
		[]string{"ip"},
	)
	dnsSourceIPsCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "dns_unique_source_ips",
			Help: "Number of unique source IPs in DNS queries per minute.",
		},
	)
	dataWithAnswerCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "data_with_answer_count",
			Help: "Total number of DNS log entries with Answer containing 10.28.8.78",
		},
	)
	queryNameCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "dns_query_name_count",
			Help: "Count of DNS responses with Answer containing 10.28.8.78 by QueryName.",
		},
		[]string{"query_name", "main_domains"},
	)
	remoteAddressCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "dns_remote_address_count",
			Help: "Count of DNS responses with Answer containing 10.28.8.78 by RemoteAddress.",
		},
		[]string{"remote_address"},
	)
)

var (
	minuteUniqueIPs = struct {
		Lock sync.Mutex
		Data map[int64]map[string]bool
	}{Data: make(map[int64]map[string]bool)}

	sourceIPCount = struct {
		sync.RWMutex
		count map[string]int
	}{count: make(map[string]int)}

	queryNameResponseCount = struct {
		Lock  sync.Mutex
		count map[string]int
	}{count: make(map[string]int)}

	queryNameMainDomain = struct {
		Lock sync.Mutex
		Data map[string]map[string]bool
	}{Data: make(map[string]map[string]bool)}

	remoteAddressResponseCount = struct {
		Lock  sync.Mutex
		count map[string]int
	}{count: make(map[string]int)}
)
