package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
			updateRemoteAddressCount(message.RemoteAddress)
		}
	}
}

func main() {
	// Register Prometheus metrics
	prometheus.MustRegister(dnsQueriesTotal, dnsResponsesTotal, dnsTopSourceIPs, dnsSourceIPsCount, dataWithAnswerCount, queryNameCount, remoteAddressCount)

	// Setup database connection
	var err error
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dbUser, dbPassword, dbHost, dbPort, dbName)
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
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
