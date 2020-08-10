package main

import (
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/robfig/cron/v3"
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
	masterCronner.AddFunc("0 0 * * *", func() {

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
	log.Println("Added", busInfoJob, "job to today's cronner")
	cronner.AddFunc(busInfoJob.ScheduledTime.ToCronExpression(time.Now().Weekday()), func() {
		fetchAndPushInfo(busInfoJob)
	})
}

func fetchAndPushInfo(busJob BusInfoJob) {
	log.Println("Fetching information to push")
	busArrivalInformation := fetchBusArrivalInformation(busJob.BusStopCode, busJob.BusServiceNo)
	textMessage := busArrivalInformation.toMessageString()
	sendOutgoingMessage(busJob.ChatID, textMessage)
}

func sendOutgoingMessage(chatID int64, textMessage string) {
	messageToSend := tgbotapi.NewMessage(chatID, textMessage)
	outgoingMessages <- messageToSend
}
