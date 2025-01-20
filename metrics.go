package main

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// 如何在redis中存储呢？
// redis中是键值
// key val
// 加前缀_metrics:dns_queries_total

// Prometheus metrics
var (
	// 1 DNS query and response number
	dnsQueriesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "dns_queries_total",
			Help: "Total number of DNS queries.",
		},
	)
	dnsResponsesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "dns_responses_total",
			Help: "Total number of DNS responses.",
		},
	)
)
var (
	// 2 每分钟不同源IP数
	dnsUniqueIPCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "dns_unique_ip_count",
			Help: "Number of unique source IPs in DNS queries per minute.",
		},
	)
	dicMinuteUniqueIPs = struct {
		Lock sync.Mutex
		Data map[int64]map[string]bool
	}{Data: make(map[int64]map[string]bool)}
)
var (
	// 3 引流响应数量
	dnsModifiedResponseCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "dns_modified_response_count",
			Help: "Total number of DNS log entries with Answer containing 10.28.8.78",
		},
	)
)
var (
	// 4 被引流的域名、计数和主站和作用
	dnsModifiedQnameInfo = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dns_modified_qname_info",
			Help: "Info of DNS responses with Answer containing 10.28.8.78 by QueryName.",
		},
		[]string{"qname", "main_domains"},
	)
	dicModifiedQnameMain = struct {
		Lock sync.Mutex
		Data map[string]map[string]bool
	}{Data: make(map[string]map[string]bool)}
	// Gauge版本。改成Counter
	// dnsModifiedQnameInfo = prometheus.NewGaugeVec(
	// 	prometheus.GaugeOpts{
	// 		Name: "dns_modified_qname_info",
	// 		Help: "Info of DNS responses with Answer containing 10.28.8.78 by QueryName.",
	// 	},
	// 	[]string{"qname", "main_domains"},
	// )
	// dicModifiedQnameCount = struct {
	// 	Lock  sync.Mutex
	// 	count map[string]int
	// }{count: make(map[string]int)}
	// dicModifiedQnameMain = struct {
	// 	Lock sync.Mutex
	// 	Data map[string]map[string]bool
	// }{Data: make(map[string]map[string]bool)}
)

var (
	// 5 被引流的Top客户端IP
	dnsModifiedQnameIPCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dns_modified_qname_ip_count",
			Help: "Count of DNS responses with Answer containing 10.28.8.78 by RemoteAddress.",
		},
		[]string{"remote_address"},
	)
	// dicModifiedQnameIPcount = struct {
	// 	Lock  sync.Mutex
	// 	count map[string]int
	// }{count: make(map[string]int)}
)
