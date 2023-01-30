package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	_ "github.com/go-sql-driver/mysql"
)

type PatientData struct {
	ErrorInfo struct {
		ErrorFlag    string      `json:"errorFlag"`
		ErrorCode    interface{} `json:"errorCode"`
		ErrorMessage interface{} `json:"errorMessage"`
	} `json:"errorInfo"`
	ItemList ItemList
}

type ItemList []struct {
	Date      string `json:"date"`
	NameJp    string `json:"name_jp"`
	Npatients string `json:"npatients"`
}

var DBClient *sql.DB

func HandleRequest() (string, error) {
	// データベース接続、初期化
	DB := os.Getenv("DB")
	DB_USERNAME := os.Getenv("DB_USERNAME")
	DB_PASSWORD := os.Getenv("DB_PASSWORD")
	HOSTNAME := os.Getenv("HOSTNAME")
	DB_NAME := os.Getenv("DB_NAME")

	connectInfo := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", DB_USERNAME, DB_PASSWORD, HOSTNAME, DB_NAME)
	db, err := sql.Open(DB, connectInfo)
	if err != nil {
		log.Fatal("failed to connect DB:", err.Error())
	}
	err = db.Ping()
	if err != nil {
		log.Fatal("failed to connect DB:", err.Error())
	}
	fmt.Println("接続完了")

	DBClient = db
	defer DBClient.Close()

	// 昨日のデータを取得
	day := time.Now().AddDate(0, 0, -1).Format("20060102")
	url := fmt.Sprintf("https://opendata.corona.go.jp/api/Covid19JapanAll?date=%s", day)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	var patientData PatientData
	err = json.Unmarshal(body, &patientData)
	if err != nil {
		log.Fatal(err)
	}

	// データベースに出力
	for _, v := range patientData.ItemList {
		_, err := DBClient.Query("INSERT INTO patients (prefName, npatients, date) VALUES (?, ?, ?);",
			v.NameJp,
			v.Npatients,
			v.Date,
		)
		if err != nil {
			log.Fatal(err)
		}
	}

	return "", nil
}

func main() {
	lambda.Start(HandleRequest)
}
