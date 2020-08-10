package main

import (
	"bufio"
	"bus-notifier/refdata"
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
)

const jobDBFile string = "job.db"
const userStateDBFile string = "user_state.db"

var outgoingMessages chan tgbotapi.Chattable
var outgoingCallbackResponses chan tgbotapi.CallbackConfig
var incomingMessages tgbotapi.UpdatesChannel
var bot *tgbotapi.BotAPI
var todayCronner *cron.Cron
var busServiceLookUp map[string]bool
var refDataDB refdata.DB
var storedJobDB JobDB
var userStateDB UserStateDB

func initTelegramAPI() {
	botToken := os.Getenv("TELEGRAM_API_TOKEN")
	newBot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatalln(err)
	}
	bot = newBot
	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	incomingMessages, err = bot.GetUpdatesChan(u)
	if err != nil {
		log.Fatalln(err)
	}
}

func initRefData() {
	busServiceLookUp = make(map[string]bool)

	file, err := os.Open("refdata/bus_services.txt")
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

	refDataDB = refdata.NewRefDataDB("refdata/refdata.db")
}

func initOutgoingChannels() {
	outgoingMessages = make(chan tgbotapi.Chattable)
	outgoingCallbackResponses = make(chan tgbotapi.CallbackConfig)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalln(err)
	}

	initTelegramAPI()
	initRefData()
	initOutgoingChannels()

	storedJobDB = NewJobDB(jobDBFile)
	userStateDB = NewUserStateDB(userStateDBFile)

	// bootstrapJobsForTesting()
	go func() {
		for outgoingMessage := range outgoingMessages {
			bot.Send(outgoingMessage)
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

func handleIncomingMessages() {
	for update := range incomingMessages {
		if update.Message == nil && update.CallbackQuery == nil {
			continue
		}

		registrationReply := handleRegistration(update)

		if registrationReply.replyMessage != nil {
			outgoingMessages <- registrationReply.replyMessage
		}

		zero := tgbotapi.CallbackConfig{}
		if registrationReply.callbackResponse != zero {
			outgoingCallbackResponses <- registrationReply.callbackResponse
		}
	}
}
