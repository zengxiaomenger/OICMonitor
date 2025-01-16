package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// DNSMessage represents a DNS log entry
type DNSMessage struct {
	Timestamp     int64    `json:"Timestamp"`
	RemoteAddress string   `json:"RemoteAddress"`
	QueryName     string   `json:"QueryName"`
	ResponseCode  int      `json:"ResponseCode"`
	ResponseTime  float64  `json:"ResponseTime"`
	AnswerCount   int      `json:"AnswerCount"`
	Answer        []string `json:"Answer"`
}

// MySQL connection parameters
const (
	dbUser     = "root"
	dbPassword = "112113114"
	dbHost     = "127.0.0.1"
	dbPort     = "3306"
	dbName     = "coredns"
)

var db *sql.DB

func init() {
	var err error
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dbUser, dbPassword, dbHost, dbPort, dbName)
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
}

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
	// //引流主站
	// queryNameMainDomain = prometheus.NewGaugeVec(
	// 	prometheus.GaugeOpts{
	// 		Name: "dns_query_name_main_domain",
	// 		Help: "Main Domain of DNS responses with Answer containing 10.28.8.78 by QueryName.",
	// 	},
	// 	[]string{"query_name"},
	// )
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

// RecordMetrics updates Prometheus metrics based on a DNS log message
func RecordMetrics(message DNSMessage, topic string) {
	if topic == "DNS_LOG_QUERY" {
		dnsQueriesTotal.WithLabelValues("minute").Inc()
		updateUniqueIPs(message.RemoteAddress)
	} else if topic == "DNS_LOG" {
		dnsResponsesTotal.WithLabelValues("minute").Inc()
		if containsAnswer(message.Answer, "10.28.8.78") {
			dataWithAnswerCount.Inc()
			updateQueryNameCount(message.QueryName)
			// cnt := queryNameResponseCount.count[message.QueryName]
			// if cnt%100 == 1 {
			// 	updateQueryNameMainDomain(message.QueryName)
			// }
			updateRemoteAddressCount(message.RemoteAddress)
		}
	}
}

// updateUniqueIPs updates the unique IPs count for the current minute
func updateUniqueIPs(ip string) {
	currentMinute := time.Now().Unix() / 60
	minuteUniqueIPs.Lock.Lock()
	defer minuteUniqueIPs.Lock.Unlock()
	if _, exists := minuteUniqueIPs.Data[currentMinute]; !exists {
		minuteUniqueIPs.Data[currentMinute] = make(map[string]bool)
	}
	minuteUniqueIPs.Data[currentMinute][ip] = true
	dnsSourceIPsCount.Set(float64(len(minuteUniqueIPs.Data[currentMinute])))

	// Cleanup old data
	for minute := range minuteUniqueIPs.Data {
		if minute < currentMinute-1 {
			delete(minuteUniqueIPs.Data, minute)
		}
	}
}

// // 备份一下 仅统计数量
// // updateQueryNameCount updates the QueryName count for DNS responses with Answer containing the target IP
// func updateQueryNameCount(queryName string) {
// 	queryNameResponseCount.Lock.Lock()
// 	defer queryNameResponseCount.Lock.Unlock()
// 	queryNameResponseCount.count[queryName]++
// 	queryNameCount.WithLabelValues(queryName).Set(float64(queryNameResponseCount.count[queryName]))
// }

// updateQueryNameCount updates the QueryName count for DNS responses with Answer containing the target IP
func updateQueryNameCount(queryName string) {
	queryNameResponseCount.Lock.Lock()
	defer queryNameResponseCount.Lock.Unlock()
	queryNameMainDomain.Lock.Lock()
	defer queryNameMainDomain.Lock.Unlock()
	queryNameResponseCount.count[queryName]++

	if queryNameResponseCount.count[queryName]%100 == 1 { //101时，更新main_domain列表
		// 删除原有键
		if _, exists := queryNameMainDomain.Data[queryName]; exists {
			delete(queryNameMainDomain.Data, queryName)
		}
		// 生成新键
		queryNameMainDomain.Data[queryName] = make(map[string]bool)
		// 连接数据库读入主域名们
		mainDomains, err := getMainDomain(queryName)
		// fmt.Println(mainDomains)
		// 把主域名导入map去重
		if err != nil {
			fmt.Errorf("failed to scan result: %v", err)
			return
		}
		for i := 0; i < len(mainDomains); i++ {
			mainDomain := mainDomains[i]
			queryNameMainDomain.Data[queryName][mainDomain] = true
		}
	}
	strMainDomains := ""
	keys := make([]string, 0, len(queryNameMainDomain.Data[queryName]))

	// 收集所有键
	for key := range queryNameMainDomain.Data[queryName] {
		keys = append(keys, key)
	}

	// 对键进行排序
	sort.Strings(keys)

	// 拼接排序后的键
	strMainDomains = strings.Join(keys, ",")

	// 处理空值情况
	if strMainDomains == "" {
		strMainDomains = "null"
	}
	//序列化一下
	//strMainDomains := ""
	//for key := range queryNameMainDomain.Data[queryName] {
	//	strMainDomains += key
	//	strMainDomains += ","
	//}
	//if strMainDomains != "" {
	//	strMainDomains = strMainDomains[:len(strMainDomains)-1]
	//}else{
	//	strMainDomains = "null"
	//}
	queryNameCount.WithLabelValues(queryName, strMainDomains).Set(float64(queryNameResponseCount.count[queryName]))
}

// 查询MainDomain
func getMainDomain(queryName string) ([]string, error) {
	mainDomains := []string{}

	// First query: exact match on domain
	query := "SELECT main_domain FROM coredns_records WHERE domain=?"
	rows, err := db.Query(query, queryName)
	if err != nil {
		return nil, fmt.Errorf("failed to query database: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var mainDomain string
		if err := rows.Scan(&mainDomain); err != nil {
			return nil, fmt.Errorf("failed to scan result: %v", err)
		}
		mainDomains = append(mainDomains, mainDomain)
	}

	// If results found, return them
	if len(mainDomains) > 0 {
		return mainDomains, nil
	}

	// Second query: remove subdomain and query by zone
	parts := strings.Split(queryName, ".")
	if len(parts) > 1 {
		zone := parts[len(parts)-2] + "." + parts[len(parts)-1] + "."
		query = "SELECT main_domain FROM coredns_records WHERE zone=?"
		rows, err = db.Query(query, zone)
		if err != nil {
			return nil, fmt.Errorf("failed to query database: %v", err)
		}
		defer rows.Close()

		for rows.Next() {
			var mainDomain string
			if err := rows.Scan(&mainDomain); err != nil {
				return nil, fmt.Errorf("failed to scan result: %v", err)
			}
			mainDomains = append(mainDomains, mainDomain)
		}
	}

	return mainDomains, nil
}

// 更新这个域名的主域们
func updateQueryNameMainDomain(queryName string) {
	queryNameMainDomain.Lock.Lock()
	defer queryNameMainDomain.Lock.Unlock()
	//删除原有键
	if _, exists := queryNameMainDomain.Data[queryName]; exists {
		delete(queryNameMainDomain.Data, queryName)
	}
	//生成新键
	queryNameMainDomain.Data[queryName] = make(map[string]bool)
	//连接数据库读入主域名们
	mainDomains, err := getMainDomain(queryName)
	//把主域名导入map去重
	if err != nil {
		for i := 0; i < len(mainDomains); i++ {
			mainDomain := mainDomains[i]
			queryNameMainDomain.Data[queryName][mainDomain] = true
		}
	}
}

// updateRemoteAddressCount updates the RemoteAddress count for DNS responses with Answer containing the target IP
func updateRemoteAddressCount(remoteAddress string) {
	remoteAddressResponseCount.Lock.Lock()
	defer remoteAddressResponseCount.Lock.Unlock()
	remoteAddressResponseCount.count[remoteAddress]++
	remoteAddressCount.WithLabelValues(remoteAddress).Set(float64(remoteAddressResponseCount.count[remoteAddress]))
}

// containsAnswer checks if the Answer field contains a specific IP
func containsAnswer(answer []string, ip string) bool {
	for _, a := range answer {
		if a == ip {
			return true
		}
	}
	return false
}

// UpdateTopSourceIPs calculates and updates the top 10 source IPs by query volume
func UpdateTopSourceIPs() {
	sourceIPCount.RLock()
	defer sourceIPCount.RUnlock()
	type ipCount struct {
		IP    string
		Count int
	}
	topIPs := make([]ipCount, 0, len(sourceIPCount.count))
	for ip, count := range sourceIPCount.count {
		topIPs = append(topIPs, ipCount{IP: ip, Count: count})
	}

	sort.Slice(topIPs, func(i, j int) bool { return topIPs[i].Count > topIPs[j].Count })
	if len(topIPs) > 10 {
		topIPs = topIPs[:10]
	}
	dnsTopSourceIPs.Reset()
	for _, top := range topIPs {
		dnsTopSourceIPs.WithLabelValues(top.IP).Set(float64(top.Count))
	}
}

func main() {
	// Register Prometheus metrics
	prometheus.MustRegister(dnsQueriesTotal, dnsResponsesTotal, dnsTopSourceIPs, dnsSourceIPsCount, dataWithAnswerCount, queryNameCount, remoteAddressCount)

	// Setup database connection
	db, err := sql.Open("mysql", "root:112113114@tcp(127.0.0.1:3306)/coredns")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Kafka consumer setup
	config := kafka.ConfigMap{
		"bootstrap.servers": "10.26.31.143:9092",
		"group.id":          "dns_metrics_group",
		"auto.offset.reset": "earliest",
	}
	consumer, err := kafka.NewConsumer(&config)
	if err != nil {
		log.Fatalf("Failed to create Kafka consumer: %v", err)
	}
	defer consumer.Close()

	// Subscribe to topics
	err = consumer.SubscribeTopics([]string{"DNS_LOG_QUERY", "DNS_LOG"}, nil)
	if err != nil {
		log.Fatalf("Failed to subscribe to topics: %v", err)
	}

	// Start Prometheus HTTP server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Fatal(http.ListenAndServe(":9154", nil))
	}()

	// Periodically update top source IPs
	go func() {
		ticker := time.NewTicker(time.Minute)
		for range ticker.C {
			UpdateTopSourceIPs()
		}
	}()

	// Consume messages from Kafka
	for {
		msg, err := consumer.ReadMessage(-1)
		if err != nil {
			log.Printf("Consumer error: %v (%v)", err, msg)
			continue
		}

		var dnsMessage DNSMessage
		err = json.Unmarshal(msg.Value, &dnsMessage)
		if err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			continue
		}

		// Update metrics based on message content
		RecordMetrics(dnsMessage, *msg.TopicPartition.Topic)
	}
}
