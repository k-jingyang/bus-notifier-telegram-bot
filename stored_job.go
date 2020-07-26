package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
)

const db string = "job.db"

type scheduledTime struct {
	Hour   int
	Minute int
}

func (s *scheduledTime) toCronExpression(day time.Weekday) string {
	return fmt.Sprintf("%d %d * * %d", s.Minute, s.Hour, day)
}

type busInfoJob struct {
	ChatID        int64
	BusStopCode   string
	BusServiceNo  string
	ScheduledTime scheduledTime
	Weekday       time.Weekday
}

func addJob(newBusInfoJob busInfoJob) {

	db, err := bolt.Open(db, 0600, nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	db.Update(func(tx *bolt.Tx) error {

		storeJob(newBusInfoJob, tx)
		storeJobForLookup(newBusInfoJob, tx)
		return nil
	})
}

func storeJob(newBusInfoJob busInfoJob, tx *bolt.Tx) {
	// User bucket: ChatID (Key) -> Registered jobs for this user (Value)
	userKey := []byte(strconv.FormatInt(newBusInfoJob.ChatID, 10))
	b, err := tx.CreateBucketIfNotExists([]byte("Users"))
	if err != nil {
		log.Fatalln(err)
	}
	storedJobs := b.Get(userKey)
	if storedJobs == nil {
		encBusInfoJobs, err := json.Marshal([]busInfoJob{newBusInfoJob})
		if err != nil {
			log.Fatalln(err)
		}
		log.Println("New job:", newBusInfoJob)
		b.Put(userKey, encBusInfoJobs)
	} else {
		existingBusInfoJobs := []busInfoJob{}
		json.Unmarshal(storedJobs, &existingBusInfoJobs)

		for _, s := range existingBusInfoJobs {
			if newBusInfoJob == s {
				log.Println("Job already exists:", newBusInfoJob)
			}
		}
		encBusInfoJobs, err := json.Marshal(append(existingBusInfoJobs, newBusInfoJob))
		if err != nil {
			log.Fatalln(err)
		}

		log.Println("Adding to existing jobs", append(existingBusInfoJobs, newBusInfoJob))
		b.Put(userKey, encBusInfoJobs)
	}

}

// Lookup bucket: Weekday (Key) -> Chat IDs with jobs for the day (Value)
func storeJobForLookup(newBusInfoJob busInfoJob, tx *bolt.Tx) error {
	dayKey := []byte(newBusInfoJob.Weekday.String())
	b, err := tx.CreateBucketIfNotExists([]byte("Jobs"))
	if err != nil {
		log.Fatalln(err)
	}
	storedChatIDs := b.Get(dayKey)
	if storedChatIDs == nil {
		encChatID, err := json.Marshal(newBusInfoJob.ChatID)
		if err != nil {
			log.Fatalln(err)
		}
		log.Println("New Chat ID:", newBusInfoJob.ChatID)
		b.Put(dayKey, encChatID)
	} else {
		existingChatIDs := []int64{}
		json.Unmarshal(storedChatIDs, &existingChatIDs)

		for _, s := range existingChatIDs {
			if newBusInfoJob.ChatID == s {
				log.Println("Chat ID already exists in the lookup at this key:", newBusInfoJob)
				return nil
			}
		}
		encChatIDs, err := json.Marshal(append(existingChatIDs, newBusInfoJob.ChatID))
		if err != nil {
			log.Fatalln(err)
		}

		log.Println("Adding to existing jobs", append(encChatIDs))
		b.Put(dayKey, encChatIDs)
	}
	return nil
}

func getJobsForDay(weekday time.Weekday) []busInfoJob {
	dayKey := []byte(weekday.String())
	jobsOnDay := []busInfoJob{}

	db, err := bolt.Open(db, 0600, nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	err = db.View(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte("Jobs"))
		if b == nil {
			return nil
		}

		// List of Chat IDs that has jobs for the day
		storedChatIDs := b.Get(dayKey)

		if len(storedChatIDs) == 0 {
			return nil
		}

		decodedChatIDs := []int64{}
		json.Unmarshal(storedChatIDs, &decodedChatIDs)
		for _, chatID := range decodedChatIDs {
			jobsOnDay = append(jobsOnDay, getJobsByChatIDandDay(chatID, weekday, tx)...)
		}
		return nil
	})

	if err != nil {
		log.Fatalln(err)
	}

	return jobsOnDay
}

func getJobsByChatIDandDay(chatID int64, weekday time.Weekday, tx *bolt.Tx) []busInfoJob {
	b := tx.Bucket([]byte("Users"))
	if b == nil {
		return nil
	}

	userKey := []byte(strconv.FormatInt(chatID, 10))
	v := b.Get(userKey)
	storedJobs := []busInfoJob{}
	json.Unmarshal(v, &storedJobs)

	storedJobsForDay := []busInfoJob{}
	for _, job := range storedJobs {
		if job.Weekday == weekday {
			storedJobsForDay = append(storedJobsForDay, job)
		}
	}

	return storedJobsForDay
}
