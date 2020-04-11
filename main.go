package main

import (
	"log"
	"os"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type BusArrivalInformation struct {
	BusStopName    string
	BusServiceNo   int
	NextBusMinutes int
}

var outgoingMessages chan tgbotapi.MessageConfig

func init() {
	outgoingMessages = make(chan tgbotapi.MessageConfig)
}

func main() {
	botToken := os.Getenv("TELEGRAM_API_TOKEN")
	bot, err := tgbotapi.NewBotAPI(botToken)

	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Panic(err)
	}
	go handleIncomingMessages(updates)

	busArrivalInformation := BusArrivalInformation{BusStopName: "Opp 628", BusServiceNo: 506, NextBusMinutes: 10}
	textMessage := constructBusArrivalMessage(busArrivalInformation)
	go sendOutgoingMessage(, textMessage)

	for outgoingMesage := range outgoingMessages {
		bot.Send(outgoingMesage)
	}

}

func handleIncomingMessages(msgChannel tgbotapi.UpdatesChannel) {
	for update := range msgChannel {
		if update.Message == nil {
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
		msg.ReplyToMessageID = update.Message.MessageID
		outgoingMessages <- msg
	}
}

func constructBusArrivalMessage(busArrivalInformation BusArrivalInformation) string {
	return busArrivalInformation.BusStopName + " | " + strconv.Itoa(busArrivalInformation.BusServiceNo) + " in " + strconv.Itoa(busArrivalInformation.NextBusMinutes) + " minutes"
}

func sendOutgoingMessage(chatId int64, textMessage string) {
	messageToSend := tgbotapi.NewMessage(chatId, textMessage)
	outgoingMessages <- messageToSend
}
