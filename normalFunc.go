package main

import (
	"fmt"
	"strings"
)

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

// containsAnswer checks if the Answer field contains a specific IP
func containsAnswer(answer []string, ip string) bool {
	for _, a := range answer {
		if a == ip {
			return true
		}
	}
	return false
}
