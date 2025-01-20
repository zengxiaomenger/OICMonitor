package main

import (
	"fmt"
	"log"

	"github.com/go-redis/redis"
)

func setValue(key, value string) error {
	err := rdb.Set(key, value, 0).Err() // v7版本的Set方法不需要context
	if err != nil {
		log.Printf("Error setting key: %v", err)
		return err
	}
	return nil
}

func getValue(key string) (string, error) {
	val, err := rdb.Get(key).Result() // v7版本的Get方法不需要context
	if err != nil {
		log.Printf("Error getting key: %v", err)
		return "", err
	}
	return val, err
}

func modifyValue(key, newValue string) error {
	err := rdb.Set(key, newValue, 0).Err() // v7版本的Set方法不需要context
	if err != nil {
		log.Printf("Error setting key: %v", err)
		return err
	}
	return nil
}

func deleteValue(key string) error {
	err := rdb.Del(key).Err() // v7版本的Del方法不需要context
	if err != nil {
		log.Printf("Error deleting key: %v", err)
		return err
	}
	return nil
}

func hsetField(key, field, value string) {
	// 使用 HSet 命令设置哈希字段的值
	err := rdb.HSet(key, field, value).Err()
	if err != nil {
		log.Printf("Error setting field %s in hash %s: %v", field, key, err)
	}
}
func hgetField(key, field string) string {
	// 使用 HGet 命令获取哈希字段的值
	val, err := rdb.HGet(key, field).Result()
	if err == redis.Nil {
		fmt.Printf("Field %s does not exist in hash %s\n", field, key)
	} else if err != nil {
		log.Printf("Error getting field %s from hash %s: %v", field, key, err)
	}
	return val
}
func hsetFieldModify(key, field, newValue string) {
	// 使用 HSet 命令修改哈希字段的值
	err := rdb.HSet(key, field, newValue).Err()
	if err != nil {
		log.Printf("Error modifying field %s in hash %s: %v", field, key, err)
	}
}
func hdelField(key, field string) {
	// 使用 HDel 命令删除哈希字段
	err := rdb.HDel(key, field).Err()
	if err != nil {
		log.Printf("Error deleting field %s from hash %s: %v", field, key, err)
	}
}
func hgetAllFields(key string) map[string]string {
	// 使用 HGetAll 命令获取整个哈希的所有字段和值
	fields, err := rdb.HGetAll(key).Result()
	if err == redis.Nil {
		fmt.Printf("Hash %s does not exist\n", key)
	} else if err != nil {
		log.Printf("Error getting all fields from hash %s: %v", key, err)
	}
	return fields
}
