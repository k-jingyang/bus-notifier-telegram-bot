package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/joho/godotenv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type BusArrivalInformation struct {
	BusStopName    string
	BusServiceNo   int
	NextBusMinutes int
}

var outgoingMessages chan tgbotapi.MessageConfig
var bot tgbotapi.BotAPI

func init() {
	outgoingMessages = make(chan tgbotapi.MessageConfig)

	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	botToken := os.Getenv("TELEGRAM_API_TOKEN")
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatal(err)
	}
	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)
}

func main() {
	go handleIncomingMessages()

	go handleStoredJobs()

	for outgoingMesage := range outgoingMessages {
		bot.Send(outgoingMesage)
		log.Println("SENT")
	}
}

func handleIncomingMessages() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updateMsgChannel, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Fatal(err)
	}

	for update := range updateMsgChannel {
		if update.Message == nil {
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
		msg.ReplyToMessageID = update.Message.MessageID
		outgoingMessages <- msg
	}
}

func handleStoredJobs() {
	fetchBusArrivalInformation("43411", "")
	busArrivalInformation := BusArrivalInformation{BusStopName: "Opp 628", BusServiceNo: 506, NextBusMinutes: 10}
	textMessage := constructBusArrivalMessage(busArrivalInformation)

	chatID, err := strconv.ParseInt(os.Getenv("CHAT_ID"), 10, 64)
	if err != nil {
		log.Fatal(err)
	}
	sendOutgoingMessage(chatID, textMessage)
}

func fetchBusArrivalInformation(busStopCode string, busServiceNo string) BusArrivalInformation {
	resp, err := http.DefaultClient.Do(buildBusArrivalAPIRequest(busStopCode, busServiceNo))
	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	log.Println(string(body))
	return BusArrivalInformation{}
}

func buildBusArrivalAPIRequest(busStopCode string, busServiceNo string) *http.Request {
	params := url.Values{}
	params.Add("BusStopCode", busStopCode)
	params.Add("ServiceNo", busServiceNo)

	url := url.URL{
		Scheme:   "http",
		Host:     "datamall2.mytransport.sg",
		Path:     "ltaodataservice/BusArrivalv2",
		RawQuery: params.Encode(),
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		log.Fatal(err)
	}
	ltaToken := os.Getenv("LTA_API_TOKEN")
	req.Header.Add("AccountKey", ltaToken)
	return req
}

func constructBusArrivalMessage(busArrivalInformation BusArrivalInformation) string {
	return busArrivalInformation.BusStopName + " | " + strconv.Itoa(busArrivalInformation.BusServiceNo) + " in " + strconv.Itoa(busArrivalInformation.NextBusMinutes) + " minutes"
}

func sendOutgoingMessage(chatID int64, textMessage string) {
	messageToSend := tgbotapi.NewMessage(chatID, textMessage)
	outgoingMessages <- messageToSend
}
