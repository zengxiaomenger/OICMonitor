package main

import "database/sql"

// MySQL connection parameters
const (
	dbUser     = "root"
	dbPassword = "112113114"
	dbHost     = "127.0.0.1"
	dbPort     = "3306"
	dbName     = "coredns"
)

var db *sql.DB
