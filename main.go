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

var outgoingMessages chan tgbotapi.MessageConfig
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
	outgoingMessages = make(chan tgbotapi.MessageConfig)
}

func main() {
	// bootstrapJobsForTesting()

	go handleIncomingMessages()
	go handleStoredJobs()

	for outgoingMesage := range outgoingMessages {
		bot.Send(outgoingMesage)
		fmt.Println("SENT")
	}
}

func bootstrapJobsForTesting() {
	myChatIDStr := os.Getenv("CHAT_ID")
	myChatID, _ := strconv.ParseInt(myChatIDStr, 10, 64)
	busInfoJob := BusInfoJob{myChatID, "43411", "157"}
	timeToExecute := ScheduledTime{17, 20}
	addJob(busInfoJob, time.Sunday, timeToExecute)
	addJob(busInfoJob, time.Monday, ScheduledTime{19, 20})
}

func handleIncomingMessages() {
	for update := range incomingMessages {
		if update.Message == nil {
			continue
		}
		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		busInfoJob, timeToExecute, dayToExecute := chatMessageToScheduledBusJob(update.Message.Chat.ID, update.Message.Text)
		// Validate bus stop no, bus stop no when registering
		addJob(busInfoJob, dayToExecute, timeToExecute)
		if dayToExecute == time.Now().Weekday() {
			addJobToTodayCronner(cronner, busInfoJob, timeToExecute)
		}
		replyMessage := fmt.Sprintf("You will be reminded for bus %s at bus stop %s every %s %02d:%02d", busInfoJob.BusServiceNo, busInfoJob.BusStopCode, dayToExecute.String(), timeToExecute.Hour, timeToExecute.Minute)
		reply := tgbotapi.NewMessage(update.Message.Chat.ID, replyMessage)
		reply.ReplyToMessageID = update.Message.MessageID
		outgoingMessages <- reply
	}
}

func chatMessageToScheduledBusJob(chatID int64, message string) (BusInfoJob, ScheduledTime, time.Weekday) {
	// Format now is busStopNo,busNo,day[0-6],hour,minute
	textArr := strings.Split(message, ",")

	busInfoJob := BusInfoJob{ChatID: chatID, BusStopCode: textArr[0], BusServiceNo: textArr[1]}

	weekday, _ := strconv.Atoi(textArr[2])

	hour, _ := strconv.Atoi(textArr[3])
	minute, _ := strconv.Atoi(textArr[4])
	scheduledTime := ScheduledTime{Hour: hour, Minute: minute}

	return busInfoJob, scheduledTime, time.Weekday(weekday)
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

func buildCronnerFromJobs(jobs []ScheduledJobs, day time.Weekday) *cron.Cron {
	cronner := cron.New()
	for _, timeJobs := range jobs {
		cronExp := timeJobs.timeToExcuete.toCronExpression(day)
		cronner.AddFunc(cronExp, func() {
			for _, busJob := range timeJobs.busInfoJobs {
				fetchAndPushInfo(busJob)
			}
		})
	}
	return cronner
}

func addJobToTodayCronner(cronner *cron.Cron, busInfoJob BusInfoJob, timeToExecute ScheduledTime) {
	cronner.AddFunc(timeToExecute.toCronExpression(time.Now().Weekday()), func() {
		fetchAndPushInfo(busInfoJob)
	})
}

func fetchAndPushInfo(busJob BusInfoJob) {
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
