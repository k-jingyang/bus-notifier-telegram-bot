package main

import (
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/robfig/cron/v3"
)

func handleStoredJobs() {
	cronner = cron.New()
	addTodayJobsToCronner(cronner)
	cronner.Start()

	// Debugging
	log.Print("Starting jobs: ")
	for _, entry := range cronner.Entries() {
		log.Println(entry)
	}

	refreshCronner := func() {
		log.Println("Old jobs:")
		for _, entry := range cronner.Entries() {
			// Debugging
			log.Println(entry)
			if entry.ID != refreshCronEntryID {
				cronner.Remove(entry.ID)
			}
		}

		addTodayJobsToCronner(cronner)

		log.Println("New jobs: ")
		for _, entry := range cronner.Entries() {
			log.Println(entry)
		}
	}

	// Daily jobs are loaded at midnight, so that cron does not contain all jobs
	refreshCronEntryID, _ = cronner.AddFunc("0 0 * * *", refreshCronner)
}

func addTodayJobsToCronner(cronner *cron.Cron) {
	today := time.Now().Weekday()
	jobs := storedJobDB.GetJobsByDay(today)
	for _, job := range jobs {
		// Debugging
		log.Println("Job:", job)
		log.Println("Cron expression", job.ScheduledTime.ToCronExpression(today))
		addJobtoCronner(cronner, job)
	}
}

func addJobtoCronner(cronner *cron.Cron, busInfoJob BusInfoJob) {
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
