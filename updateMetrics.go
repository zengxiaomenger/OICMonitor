package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func updateQueriesTotal() {
	// dnsQueriesTotal.Inc()
	updateCounter(1)
}
func updateResponsesTotal() error {
	dnsResponsesTotal.Inc()
	return nil
}

// updateUniqueIPs updates the unique IPs count for the current minute
func updateUniqueIPs(ip string) {
	currentMinute := time.Now().Unix() / 60
	dicMinuteUniqueIPs.Lock.Lock()
	defer dicMinuteUniqueIPs.Lock.Unlock()
	if _, exists := dicMinuteUniqueIPs.Data[currentMinute]; !exists {
		dicMinuteUniqueIPs.Data[currentMinute] = make(map[string]bool)
	}
	dicMinuteUniqueIPs.Data[currentMinute][ip] = true
	dnsUniqueIPCount.Set(float64(len(dicMinuteUniqueIPs.Data[currentMinute])))

	// Cleanup old data
	for minute := range dicMinuteUniqueIPs.Data {
		if minute < currentMinute-1 {
			delete(dicMinuteUniqueIPs.Data, minute)
		}
	}
}
func updateModifiedResponseCount() {
	dnsModifiedResponseCount.Inc()
}

// updateQueryNameCount updates the QueryName count for DNS responses with Answer containing the target IP
func updateQueryNameCount(queryName string) {
	dicModifiedQnameMain.Lock.Lock()
	defer dicModifiedQnameMain.Lock.Unlock()

	// 不存在就导入新键
	if _, exists := dicModifiedQnameMain.Data[queryName]; !exists {
		// delete(dicModifiedQnameMain.Data, queryName)
		// 生成新键
		dicModifiedQnameMain.Data[queryName] = make(map[string]bool)
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
			dicModifiedQnameMain.Data[queryName][mainDomain] = true
		}
	}

	strMainDomains := ""
	keys := make([]string, 0, len(dicModifiedQnameMain.Data[queryName]))

	// 收集所有键
	for key := range dicModifiedQnameMain.Data[queryName] {
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
	dnsModifiedQnameInfo.WithLabelValues(queryName, strMainDomains).Inc()
}

// gauge版
// func updateQueryNameCount(queryName string) {
// 	dicModifiedQnameCount.Lock.Lock()
// 	defer dicModifiedQnameCount.Lock.Unlock()
// 	dicModifiedQnameMain.Lock.Lock()
// 	defer dicModifiedQnameMain.Lock.Unlock()

// 	dicModifiedQnameCount.count[queryName]++

// 	if dicModifiedQnameCount.count[queryName]%100 == 1 { //101时，更新main_domain列表
// 		// 删除原有键
// 		if _, exists := dicModifiedQnameMain.Data[queryName]; exists {
// 			delete(dicModifiedQnameMain.Data, queryName)
// 		}
// 		// 生成新键
// 		dicModifiedQnameMain.Data[queryName] = make(map[string]bool)
// 		// 连接数据库读入主域名们
// 		mainDomains, err := getMainDomain(queryName)
// 		// fmt.Println(mainDomains)
// 		// 把主域名导入map去重
// 		if err != nil {
// 			fmt.Errorf("failed to scan result: %v", err)
// 			return
// 		}
// 		for i := 0; i < len(mainDomains); i++ {
// 			mainDomain := mainDomains[i]
// 			dicModifiedQnameMain.Data[queryName][mainDomain] = true
// 		}
// 	}
// 	strMainDomains := ""
// 	keys := make([]string, 0, len(dicModifiedQnameMain.Data[queryName]))

// 	// 收集所有键
// 	for key := range dicModifiedQnameMain.Data[queryName] {
// 		keys = append(keys, key)
// 	}
// 	// 对键进行排序
// 	sort.Strings(keys)
// 	// 拼接排序后的键
// 	strMainDomains = strings.Join(keys, ",")
// 	// 处理空值情况
// 	if strMainDomains == "" {
// 		strMainDomains = "null"
// 	}
// 	dnsModifiedQnameInfo.WithLabelValues(queryName, strMainDomains).Set(float64(dicModifiedQnameCount.count[queryName]))
// }

// updateModifiedQnameIPCount updates the RemoteAddress count for DNS responses with Answer containing the target IP
func updateModifiedQnameIPCount(remoteAddress string) {
	dnsModifiedQnameIPCount.WithLabelValues(remoteAddress).Inc()
}
