package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/robfig/cron"
)

func handleStoredJobs() {
	today := time.Now().Weekday()
	todayCronner = buildCronnerFromJobs(storedJobDB.GetJobsByDay(today), today)
	todayCronner.Start()

	// Debugging
	log.Print("Starting jobs: ")
	for _, entry := range todayCronner.Entries() {
		log.Println(entry)
	}

	// Daily jobs are loaded at midnight, so that cron does not contain all jobs
	masterCronner := cron.New()
	masterCronner.AddFunc("*0 0 * * *", func() {

		// Debugging
		log.Print("Old jobs: ")
		for _, entry := range todayCronner.Entries() {
			log.Println(entry)
		}
		todayCronner.Stop()

		newDay := time.Now().Weekday()
		todayCronner = buildCronnerFromJobs(storedJobDB.GetJobsByDay(newDay), newDay)
		todayCronner.Start()

		// Debugging
		log.Print("New jobs: ")
		for _, entry := range todayCronner.Entries() {
			log.Println(entry)
		}
	})
	masterCronner.Start()
}

func buildCronnerFromJobs(jobs []BusInfoJob, day time.Weekday) *cron.Cron {
	cronner := cron.New()
	for _, job := range jobs {
		cronExp := job.ScheduledTime.ToCronExpression(day)
		cronner.AddFunc(cronExp, func() {
			fetchAndPushInfo(job)
		})
	}
	return cronner
}

func addJobToTodayCronner(cronner *cron.Cron, busInfoJob BusInfoJob) {
	cronner.AddFunc(busInfoJob.ScheduledTime.ToCronExpression(time.Now().Weekday()), func() {
		fetchAndPushInfo(busInfoJob)
	})
}

func fetchAndPushInfo(busJob BusInfoJob) {
	busArrivalInformation := fetchBusArrivalInformation(busJob.BusStopCode, busJob.BusServiceNo)
	textMessage := constructBusArrivalMessage(busArrivalInformation)
	sendOutgoingMessage(busJob.ChatID, textMessage)
}

func constructBusArrivalMessage(busArrivalInformation busArrivalInformation) string {
	stringBuilder := strings.Builder{}
	stringBuilder.WriteString(busArrivalInformation.BusServiceNo)
	stringBuilder.WriteString(" @ ")
	stringBuilder.WriteString(busArrivalInformation.BusStopCode)
	stringBuilder.WriteString(" | ")
	if busArrivalInformation.NextBusMinutes == 0 {
		stringBuilder.WriteString("Arr")
	} else {
		stringBuilder.WriteString(fmt.Sprintf("%.0f mins", busArrivalInformation.NextBusMinutes))
	}
	if busArrivalInformation.NextBusMinutes2 > 0 {
		stringBuilder.WriteString(" | ")
		stringBuilder.WriteString(fmt.Sprintf("%.0f mins", busArrivalInformation.NextBusMinutes2))
	}
	if busArrivalInformation.NextBusMinutes3 > 0 {
		stringBuilder.WriteString(" | ")
		stringBuilder.WriteString(fmt.Sprintf("%.0f mins", busArrivalInformation.NextBusMinutes3))
	}
	return stringBuilder.String()
}

func sendOutgoingMessage(chatID int64, textMessage string) {
	messageToSend := tgbotapi.NewMessage(chatID, textMessage)
	outgoingMessages <- messageToSend
}
