package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/line/line-bot-sdk-go/linebot"

	_ "github.com/go-sql-driver/mysql"
)

func UnmarshalLineRequest(data []byte) (LineRequest, error) {
	var r LineRequest
	err := json.Unmarshal(data, &r)
	return r, err
}

type LineRequest struct {
	Events      []*linebot.Event `json:"events"`
	Destination string           `json:"destination"`
}

type Message struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Text string `json:"text"`
}

type Source struct {
	UserID string `json:"userId"`
	Type   string `json:"type"`
}

type Patient struct {
	ID        int
	PrefName  string
	Npatients int
	Date      string
}

var singlePatient struct {
	ID        int
	PrefName  string
	Npatients int
	Date      string
}

var prefectures []string
var greetingWords []string
var otherWords []string
var confessionWords []string

func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	initializeDBConnection()
	defer DBClient.Close()

	myLineRequest, err := UnmarshalLineRequest([]byte(request.Body))
	if err != nil {
		log.Fatal(err)
	}

	prefectures = []string{"北海", "青森", "秋田", "宮城", "山形", "福島", "静岡", "栃木", "茨城", "群馬", "埼玉", "千葉", "東京", "神奈", "新潟", "長野", "山梨", "静岡", "富山", "石川", "福井", "岐阜", "愛知", "滋賀", "三重", "京都", "兵庫", "大阪", "奈良", "和歌", "鳥取", "島根", "岡山", "広島", "山口", "香川", "徳島", "高知", "愛媛", "福岡", "佐賀", "長崎", "大分", "熊本", "宮崎", "鹿児", "沖縄"}
	greetingWords = []string{"おはよう", "こんにちは", "こんばんは"}
	otherWords = []string{"ありがとう"}
	confessionWords = []string{"すき", "好き"}

	bot, err := linebot.New(
		os.Getenv("CHANNELSECRET"),
		os.Getenv("ACCESSTOKEN"),
	)
	if err != nil {
		log.Fatal(err)
	}

	backMsg := "都道府県情報を入力してください。"

	for _, e := range myLineRequest.Events {
		if e.Type == linebot.EventTypeMessage {
			switch message := e.Message.(type) {
			case *linebot.TextMessage: //テキストで受信した場合
				replyMessage := message.Text
				if Contains(prefectures, replyMessage) { //都道府県情報を取得した場合
					getPrefectureInfo(bot, e, message)
				} else if Contains(greetingWords, replyMessage) { //挨拶を取得した場合
					sendGreetingWords(bot, e)
				} else if Contains(otherWords, replyMessage) { //お礼を取得した場合
					sendOtherWords(bot, e)
				} else if Contains(confessionWords, replyMessage) { //その他を取得した場合
					sendConfessionWords(bot, e)
				} else {
					_, err = bot.ReplyMessage(e.ReplyToken, linebot.NewTextMessage(backMsg)).Do()
					if err != nil {
						log.Print(err)
					}
				}
			case *linebot.LocationMessage: //位置情報を受信した場合
				locationInfoBack(bot, e)
			default: //位置情報以外を受信した場合
				_, err = bot.ReplyMessage(e.ReplyToken, linebot.NewTextMessage(backMsg)).Do()
				if err != nil {
					log.Print(err)
				}
			}
		}
	}
	return events.APIGatewayProxyResponse{Body: request.Body, StatusCode: 200}, nil
}
func main() {
	lambda.Start(Handler)
}

func locationInfoBack(bot *linebot.Client, e *linebot.Event) {
	msg := e.Message.(*linebot.LocationMessage)
	if !strings.Contains(msg.Address, "日本") {
		bot.ReplyMessage(e.ReplyToken, linebot.NewTextMessage("国外はサービス対象外です！")).Do()
		return
	}
	address := strings.Split(msg.Address, " ")[1]
	prefName := regexp.MustCompile("[都,道,府,県]").Split(address, -1)[0]
	message := searchPatientInfo(prefName)

	bot.ReplyMessage(e.ReplyToken, linebot.NewTextMessage(message)).Do()
}

func getPrefectureInfo(bot *linebot.Client, e *linebot.Event, text *linebot.TextMessage) {
	prefName := cutOutCharacters(text.Text, 2)
	if !Contains(prefectures, prefName) {
		bot.ReplyMessage(e.ReplyToken, linebot.NewTextMessage("正しい都道府県を入力してください！")).Do()
		return
	}
	message := searchPatientInfo(prefName)

	bot.ReplyMessage(e.ReplyToken, linebot.NewTextMessage(message)).Do()
}

func searchPatientInfo(prefName string) (m string) {
	rows, err := DBClient.Query("SELECT id, prefName, npatients, date FROM patients WHERE prefName LIKE CONCAT(?, '%') LIMIT 2", prefName)
	if err != nil {
		log.Fatal(err)
	}
	var patients []Patient

	for rows.Next() {
		if err := rows.Scan(&singlePatient.ID, &singlePatient.PrefName, &singlePatient.Npatients, &singlePatient.Date); err != nil {
			log.Fatal(err)
		}
		patients = append(patients, singlePatient)
	}
	prefecture := patients[0].PrefName
	todaysPatient := strconv.Itoa(patients[0].Npatients - patients[1].Npatients)
	totalPatients := strconv.Itoa(patients[0].Npatients)

	message := fmt.Sprintf("%sの累計感染者数は%s人、昨日の感染者数は%s人でした！\n\n本日も感染に気をつけてがんばりましょう！☺️",
		prefecture, totalPatients, todaysPatient)

	return message
}

func sendGreetingWords(bot *linebot.Client, e *linebot.Event) {
	message := "こんにちは！今日も一日頑張りましょう！☺️"
	bot.ReplyMessage(e.ReplyToken, linebot.NewTextMessage(message)).Do()
}

func sendOtherWords(bot *linebot.Client, e *linebot.Event) {
	message := "どういたしまして☺️\n\nいつも使っていただきありがとうございます！"
	bot.ReplyMessage(e.ReplyToken, linebot.NewTextMessage(message)).Do()
}

func sendConfessionWords(bot *linebot.Client, e *linebot.Event) {
	message := "ごめんなさい😢"
	bot.ReplyMessage(e.ReplyToken, linebot.NewTextMessage(message)).Do()
}

//先頭から指定文字数だけを切りだす("abcde",3)　→　"abc"
func cutOutCharacters(s string, count int) string {
	if utf8.RuneCountInString(s) > count {
		return string([]rune(s)[:count])
	}
	return s
}

// 配列に要素があるか確認する
func Contains(s []string, e string) bool {
	for _, v := range s {
		if strings.Contains(e, v) {
			return true
		}
	}
	return false
}

var DBClient *sql.DB

func initializeDBConnection() {
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
}
