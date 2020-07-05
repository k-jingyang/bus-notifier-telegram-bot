package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
	"github.com/robfig/cron"
)

var outgoingMessages chan tgbotapi.Chattable
var incomingMessages tgbotapi.UpdatesChannel
var bot *tgbotapi.BotAPI
var cronner *cron.Cron

func init() {
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

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	incomingMessages, err = bot.GetUpdatesChan(u)
	if err != nil {
		log.Fatal(err)
	}
	outgoingMessages = make(chan tgbotapi.Chattable)
}

func main() {
	// bootstrapJobsForTesting()

	go handleIncomingMessages()
	go handleStoredJobs()

	for outgoingMesage := range outgoingMessages {
		bot.Send(outgoingMesage)
		fmt.Println("Sent")
	}
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

func handleIncomingMessages() {
	for update := range incomingMessages {

		if update.Message == nil && update.CallbackQuery == nil {
			continue
		}

		outgoingMessages <- handleRegistration(update)
	}
}

func handleStoredJobs() {
	today := time.Now().Weekday()
	cronner = buildCronnerFromJobs(getJobsForDay(today), today)
	cronner.Start()

	// Debugging
	log.Print("Starting jobs: ")
	for _, entry := range cronner.Entries() {
		log.Println(entry)
	}

	// Daily jobs are loaded at midnight, so that cron does not contain all jobs
	masterCronner := cron.New()
	masterCronner.AddFunc("*0 0 * * *", func() {

		// Debugging
		log.Print("Old jobs: ")
		for _, entry := range cronner.Entries() {
			log.Println(entry)
		}
		cronner.Stop()

		newDay := time.Now().Weekday()
		cronner = buildCronnerFromJobs(getJobsForDay(newDay), newDay)
		cronner.Start()

		// Debugging
		log.Print("New jobs: ")
		for _, entry := range cronner.Entries() {
			log.Println(entry)
		}
	})
	masterCronner.Start()
}

func buildCronnerFromJobs(jobs []scheduledJobs, day time.Weekday) *cron.Cron {
	cronner := cron.New()
	for _, timeJobs := range jobs {
		cronExp := timeJobs.TimeToExecute.toCronExpression(day)
		cronner.AddFunc(cronExp, func() {
			for _, busJob := range timeJobs.BusInfoJobs {
				fetchAndPushInfo(busJob)
			}
		})
	}
	return cronner
}

func addJobToTodayCronner(cronner *cron.Cron, busInfoJob busInfoJob, timeToExecute scheduledTime) {
	cronner.AddFunc(timeToExecute.toCronExpression(time.Now().Weekday()), func() {
		fetchAndPushInfo(busInfoJob)
	})
}

func fetchAndPushInfo(busJob busInfoJob) {
	busArrivalInformation := fetchBusArrivalInformation(busJob.BusStopCode, busJob.BusServiceNo)
	textMessage := constructBusArrivalMessage(busArrivalInformation)
	sendOutgoingMessage(busJob.ChatID, textMessage)
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
