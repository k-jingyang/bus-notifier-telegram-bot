package main

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
	"github.com/robfig/cron"
)

var outgoingMessages chan tgbotapi.Chattable
var outgoingCallbackResponses chan tgbotapi.CallbackConfig
var incomingMessages tgbotapi.UpdatesChannel
var bot *tgbotapi.BotAPI
var todayCronner *cron.Cron
var busServiceLookUp map[string]bool

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalln(err)
	}
	botToken := os.Getenv("TELEGRAM_API_TOKEN")
	bot, err = tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatalln(err)
	}
	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	incomingMessages, err = bot.GetUpdatesChan(u)
	if err != nil {
		log.Fatalln(err)
	}
	outgoingMessages = make(chan tgbotapi.Chattable)
	outgoingCallbackResponses = make(chan tgbotapi.CallbackConfig)
	initBusServiceLookUp()
}

func initBusServiceLookUp() {
	busServiceLookUp = make(map[string]bool)

	file, err := os.Open("bus_services.txt")
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		busServiceLookUp[scanner.Text()] = true
	}
	if err := scanner.Err(); err != nil {
		log.Fatalln(err)
	}
}

func main() {
	// bootstrapJobsForTesting()
	go func() {
		for outgoingMesage := range outgoingMessages {
			bot.Send(outgoingMesage)
		}
	}()
	go func() {
		for outgoingCallbackResponse := range outgoingCallbackResponses {
			bot.AnswerCallbackQuery(outgoingCallbackResponse)
		}
	}()

	go handleStoredJobs()
	handleIncomingMessages()
}

func bootstrapJobsForTesting() {
	myChatIDStr := os.Getenv("CHAT_ID")
	myChatID, _ := strconv.ParseInt(myChatIDStr, 10, 64)
	busInfoJob := busInfoJob{myChatID, "43411", "506"}
	timeToExecute := scheduledTime{17, 20}

	addJob(busInfoJob, time.Monday, timeToExecute)
	// addJob(busInfoJob, time.Monday, scheduledTime{9, 45})
	// addJob(busInfoJob, time.Monday, scheduledTime{9, 50})
	// addJob(busInfoJob, time.Monday, scheduledTime{10, 00})
}
