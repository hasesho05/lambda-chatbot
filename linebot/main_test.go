package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
)

func TestHandleRequest(t *testing.T) {
	DB := os.Getenv("DB")
	DB_USERNAME := os.Getenv("DB_USERNAME")
	DB_PASSWORD := os.Getenv("DB_PASSWORD")
	HOSTNAME := os.Getenv("HOSTNAME")
	DB_NAME := os.Getenv("DB_NAME")

	connectInfo := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", DB_USERNAME, DB_PASSWORD, HOSTNAME, DB_NAME)
	db, err := sql.Open(DB, connectInfo)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}
	defer db.Close()

}
