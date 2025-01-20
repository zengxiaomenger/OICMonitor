package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/go-redis/redis"
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

func registerPrometheusMetrics() {
	prometheus.MustRegister(
		dnsQueriesTotal,
		dnsResponsesTotal,
		dnsUniqueIPCount,
		dnsModifiedResponseCount,
		dnsModifiedQnameInfo,
		dnsModifiedQnameIPCount,
	)
}

func connectMysql() {
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
}

func connectRedis() {
	rdb = redis.NewClient(&redis.Options{
		Addr:     redisServer,   // Redis 服务器地址
		Password: redisPassword, // Redis 密码
		DB:       0,             // 默认数据库
	})

	// 测试 Redis 连接：Ping 命令
	pong, err := rdb.Ping().Result() // 使用 Ping() 时不需要传入 context
	if err != nil {
		log.Fatalf("Redis connection failed: %v", err)
	}
	fmt.Println("Redis connected, Ping response:", pong)
}

func initializeMetrics() { // 使用redis更新metrics
	// 初始化dns_queries_total
	redisKey := "dns_queries_total"

	// Check if the counter value exists in Redis
	val, err := rdb.Get(redisKey).Result()
	if err == redis.Nil {
		// Key does not exist, initialize counter with 0
		log.Println("No value found in Redis, initializing counter to 0.")
	} else if err != nil {
		// Other Redis error
		log.Fatalf("Error reading from Redis: %v", err)
	} else {
		// Key exists, set Prometheus counter to the Redis value
		initialValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			log.Fatalf("Error parsing Redis value to float: %v", err)
		}
		dnsQueriesTotal.Add(initialValue)
		log.Printf("Loaded initial counter value from Redis: %f", initialValue)
	}
}

// updateCounter updates the Prometheus counter and Redis value.
func updateCounter(increment int) {
	// Update Prometheus counter
	dnsQueriesTotal.Add(float64(increment))

	// Update Redis with the new counter value
	redisKey := "dns_queries_total"

	// Watch the Redis key to ensure we don't have a race condition
	rdb.Watch(func(tx *redis.Tx) error {
		// Get the current value of the counter in Redis
		val, err := tx.Get(redisKey).Result()
		if err != nil && err != redis.Nil {
			return err
		}

		// Parse the current value or initialize to 0 if not found
		currentValue := 0.0
		if val != "" {
			currentValue, err = strconv.ParseFloat(val, 64)
			if err != nil {
				return fmt.Errorf("error parsing Redis value to float: %v", err)
			}
		}

		// Calculate the new value
		newValue := currentValue + float64(increment)

		// Start the Redis pipeline to set the new value
		pipe := tx.Pipeline()
		pipe.Set(redisKey, fmt.Sprintf("%f", newValue), 0)

		// Execute the transaction
		_, err = pipe.Exec()
		return err
	}, redisKey)

	log.Printf("Counter updated: %f", dnsQueriesTotal)
}

func connectKafka() {
	config := kafka.ConfigMap{
		"bootstrap.servers": kafkaServers,
		"group.id":          kafkaGroupId,
		"auto.offset.reset": kafkaAutoOffsetReset,
	}
	var err error
	consumer, err = kafka.NewConsumer(&config)
	if err != nil {
		log.Fatalf("Failed to create Kafka consumer: %v", err)
	}
}

func consumeKafka() {
	// Subscribe to topics
	err := consumer.SubscribeTopics([]string{"DNS_LOG_QUERY", "DNS_LOG"}, nil)
	if err != nil {
		log.Fatalf("Failed to subscribe to topics: %v", err)
	}
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

// RecordMetrics updates Prometheus metrics based on a DNS log message
func RecordMetrics(message DNSMessage, topic string) {
	if topic == "DNS_LOG_QUERY" {
		updateQueriesTotal()
		updateUniqueIPs(message.RemoteAddress)
	} else if topic == "DNS_LOG" {
		updateResponsesTotal()
		if containsAnswer(message.Answer, "10.28.8.78") {
			updateModifiedResponseCount()
			updateQueryNameCount(message.QueryName)
			updateModifiedQnameIPCount(message.RemoteAddress)
		}
	}
}
func main() {
	// 1 register prometheus metrics
	registerPrometheusMetrics()

	// 2 Start Prometheus HTTP server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Fatal(http.ListenAndServe(":9155", nil))
	}()

	// 3 connect mysql
	connectMysql()
	defer db.Close()

	// 4 connect redis
	connectRedis()

	// 5 initialize metrics using redis
	initializeMetrics()

	// 6 connect kafka
	connectKafka()
	defer consumer.Close()

	// 7 comsume kafka
	consumeKafka()

}
