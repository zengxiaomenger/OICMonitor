package main

import (
	"fmt"
	"sort"
	"strings"
)

// 根据qName查询MainDomains，再去重、排序、变成字符串
func getStrMainDomains(qName string) (string, error) {
	//这个函数单纯从数据库获取主域名列表
	mainDomains := []string{}

	// First query: exact match on domain
	// 先查domain是这个值的
	query := "SELECT main_domain FROM coredns_records WHERE domain='?'"
	rows, err := db.Query(query, qName)
	if err != nil {
		return "null1", fmt.Errorf("failed to query database: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var mainDomain string
		if err := rows.Scan(&mainDomain); err != nil {
			return "null2", fmt.Errorf("failed to scan result: %v", err)
		}
		mainDomains = append(mainDomains, mainDomain)
	}

	// If results found, return them
	if len(mainDomains) > 0 {
		return uniqueSortedStr(mainDomains), nil
	}

	// Second query: remove subdomain and query by zone
	// 否则查zone等于二级域的
	parts := strings.Split(qName, ".")
	if len(parts) > 1 {
		zone := parts[len(parts)-2] + "." + parts[len(parts)-1] + "."
		query = "SELECT main_domain FROM coredns_records WHERE zone='?'"
		rows, err = db.Query(query, zone)
		if err != nil {
			return "null3", fmt.Errorf("failed to query database: %v", err)
		}
		defer rows.Close()

		for rows.Next() {
			var mainDomain string
			if err := rows.Scan(&mainDomain); err != nil {
				return "null4", fmt.Errorf("failed to scan result: %v", err)
			}
			mainDomains = append(mainDomains, mainDomain)
		}
	}
	return uniqueSortedStr(mainDomains), nil
}
func uniqueSortedStr(mainDomains []string) string {
	ans := ""
	// 放入map去重
	tp := make(map[string]bool)
	for i := 0; i < len(mainDomains); i++ {
		mainDomain := mainDomains[i]
		tp[mainDomain] = true
	}
	// 放入slice排序
	keys := make([]string, 0, len(tp))
	for key := range tp {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	// 拼接排序后的键
	ans = strings.Join(keys, ",")
	// 处理空值情况
	if ans == "" {
		ans = "null"
	}
	return ans
}

// 更新这个域名的主域们
// func updateQueryNameMainDomain(queryName string) {
// 	dicModifiedQnameMain.Lock.Lock()
// 	defer dicModifiedQnameMain.Lock.Unlock()
// 	//删除原有键
// 	if _, exists := dicModifiedQnameMain.Data[queryName]; exists {
// 		delete(dicModifiedQnameMain.Data, queryName)
// 	}
// 	//生成新键
// 	dicModifiedQnameMain.Data[queryName] = make(map[string]bool)
// 	//连接数据库读入主域名们
// 	mainDomains, err := getUniqueSortedMainDomainsStr(queryName)
// 	//把主域名导入map去重
// 	if err != nil {
// 		for i := 0; i < len(mainDomains); i++ {
// 			mainDomain := mainDomains[i]
// 			dicModifiedQnameMain.Data[queryName][mainDomain] = true
// 		}
// 	}
// }

// containsAnswer checks if the Answer field contains a specific IP
func containsAnswer(answer []string, ip string) bool {
	for _, a := range answer {
		if a == ip {
			return true
		}
	}
	return false
}
