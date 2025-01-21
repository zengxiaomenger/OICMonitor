package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/go-redis/redis"
	"github.com/prometheus/client_golang/prometheus"
)

// 转成redis键格式
func addPrefix(metric string) string {
	return "_metrics:" + metric
}

func initializeCounter(counter *prometheus.Counter, redisKey string) { // 使用redis更新metrics
	redisKey = addPrefix(redisKey)
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
		(*counter).Add(float64(initialValue))
		log.Printf("Loaded initial counter value from Redis: %f", initialValue)
	}
}

// updateCounter updates the Prometheus counter and Redis value.
func addCounter(counter *prometheus.Counter, redisKey string) {
	(*counter).Inc()
	redisKey = addPrefix(redisKey)
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
		newValue := currentValue + float64(1)

		// Start the Redis pipeline to set the new value
		pipe := tx.Pipeline()
		pipe.Set(redisKey, fmt.Sprintf("%f", newValue), 0)

		// Execute the transaction
		_, err = pipe.Exec()
		return err
	}, redisKey)

	log.Printf("Counter updated: %s", redisKey)
}
