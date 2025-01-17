package main

import (
	"database/sql"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/go-redis/redis"
)

// MySQL connection parameters
const (
	dbUser     = "root"
	dbPassword = "112113114"
	dbHost     = "127.0.0.1"
	dbPort     = "3306"
	dbName     = "coredns"
)

var db *sql.DB

const (
	kafkaServers         = "10.26.31.143:9092"
	kafkaGroupId         = "dns_metrics_group_test"
	kafkaAutoOffsetReset = "earliest"
)

var consumer *kafka.Consumer

const (
	redisServer   = "127.0.0.1:6379"
	redisPassword = "123456"
	redisDB       = 0
)

var rdb *redis.Client
