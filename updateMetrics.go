package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

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
	queryNameCount.WithLabelValues(queryName, strMainDomains).Set(float64(queryNameResponseCount.count[queryName]))
}

// updateRemoteAddressCount updates the RemoteAddress count for DNS responses with Answer containing the target IP
func updateRemoteAddressCount(remoteAddress string) {
	remoteAddressResponseCount.Lock.Lock()
	defer remoteAddressResponseCount.Lock.Unlock()
	remoteAddressResponseCount.count[remoteAddress]++
	remoteAddressCount.WithLabelValues(remoteAddress).Set(float64(remoteAddressResponseCount.count[remoteAddress]))
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
