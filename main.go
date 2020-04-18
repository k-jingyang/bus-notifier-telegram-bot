package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var outgoingMessages chan tgbotapi.MessageConfig
var bot *tgbotapi.BotAPI

func init() {
	outgoingMessages = make(chan tgbotapi.MessageConfig)

	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	botToken := os.Getenv("TELEGRAM_API_TOKEN")
	bot, err = tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatal(err)
	}
	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)
}

func main() {
	// go handleIncomingMessages()

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
	// validate bus stop no, bus stop no when registering
}

func handleStoredJobs() {
	// busArrivalInformation := BusArrivalInformation{BusStopName: "Opp 628", BusServiceNo: 506, NextBusMinutes: 10}
	busArrivalInformation := fetchBusArrivalInformation("43411", "157")
	textMessage := constructBusArrivalMessage(busArrivalInformation)

	log.Println(textMessage)
	/*
		chatID, err := strconv.ParseInt(os.Getenv("CHAT_ID"), 10, 64)
		if err != nil {
			log.Fatal(err)
		}
		sendOutgoingMessage(chatID, textMessage)
	*/
}

func constructBusArrivalMessage(busArrivalInformation BusArrivalInformation) string {
	stringBuilder := strings.Builder{}
	stringBuilder.WriteString(busArrivalInformation.BusServiceNo)
	stringBuilder.WriteString(" @ ")
	stringBuilder.WriteString(busArrivalInformation.BusStopName)
	stringBuilder.WriteString(" | ")
	if busArrivalInformation.NextBusMinutes == 0 {
		stringBuilder.WriteString("Arr")
	} else {
		stringBuilder.WriteString(fmt.Sprintf("%.0f mins", busArrivalInformation.NextBusMinutes))
	}
	stringBuilder.WriteString(" | ")
	stringBuilder.WriteString(fmt.Sprintf("%.0f mins", busArrivalInformation.NextBusMinutes2))
	stringBuilder.WriteString(" | ")
	stringBuilder.WriteString(fmt.Sprintf("%.0f mins", busArrivalInformation.NextBusMinutes3))
	return stringBuilder.String()
}

func sendOutgoingMessage(chatID int64, textMessage string) {
	messageToSend := tgbotapi.NewMessage(chatID, textMessage)
	outgoingMessages <- messageToSend
}
