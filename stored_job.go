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
const userBucket string = "Users"
const jobBucket string = "Jobs"

type scheduledTime struct {
	Hour   int
	Minute int
}

func (s *scheduledTime) toString() string {
	return fmt.Sprintf("%02d:%02d", s.Hour, s.Minute)
}

func (s *scheduledTime) toCronExpression(day time.Weekday) string {
	return fmt.Sprintf("%d %d * * %d", s.Minute, s.Hour, day)
}

// BusInfoJob contains all information of a registered bus alarm
type BusInfoJob struct {
	ChatID        int64
	BusStopCode   string
	BusServiceNo  string
	ScheduledTime scheduledTime
	Weekday       time.Weekday
}

// StoreJob stores the registered bus alarm into the database
func StoreJob(newBusInfoJob BusInfoJob) {

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

// User bucket: ChatID (Key) -> Registered jobs for this user (Value)
func storeJob(newBusInfoJob BusInfoJob, tx *bolt.Tx) {
	userKey := []byte(strconv.FormatInt(newBusInfoJob.ChatID, 10))
	b, err := tx.CreateBucketIfNotExists([]byte(userBucket))
	if err != nil {
		log.Fatalln(err)
	}
	storedJobs := b.Get(userKey)
	if storedJobs == nil {
		encBusInfoJobs, err := json.Marshal([]BusInfoJob{newBusInfoJob})
		if err != nil {
			log.Fatalln(err)
		}
		log.Println("New job:", newBusInfoJob)
		b.Put(userKey, encBusInfoJobs)
	} else {
		existingBusInfoJobs := []BusInfoJob{}
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
func storeJobForLookup(newBusInfoJob BusInfoJob, tx *bolt.Tx) error {
	dayKey := []byte(newBusInfoJob.Weekday.String())
	b, err := tx.CreateBucketIfNotExists([]byte(jobBucket))
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

// GetJobsByDay retrieves all bus alarms for the particular given day
func GetJobsByDay(weekday time.Weekday) []BusInfoJob {
	jobsOnDay := []BusInfoJob{}

	db, err := bolt.Open(db, 0600, nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	err = db.View(func(tx *bolt.Tx) error {

		chatIDs := getChatIDsByDay(weekday, tx)

		if len(chatIDs) == 0 {
			return nil
		}

		for _, chatID := range chatIDs {
			userJobsOnDay := getJobsByChatIDandDay(chatID, weekday, tx)
			if len(userJobsOnDay) == 0 {
				log.Panicln("Desync of information between the two buckets")
			}
			jobsOnDay = append(jobsOnDay, userJobsOnDay...)
		}
		return nil
	})

	if err != nil {
		log.Fatalln(err)
	}

	return jobsOnDay
}

func getChatIDsByDay(weekday time.Weekday, tx *bolt.Tx) []int64 {
	dayKey := []byte(weekday.String())

	b := tx.Bucket([]byte(jobBucket))
	if b == nil {
		return nil
	}

	// List of Chat IDs that has jobs for the day
	storedChatIDs := b.Get(dayKey)

	decodedChatIDs := []int64{}
	json.Unmarshal(storedChatIDs, &decodedChatIDs)

	return decodedChatIDs
}

func getJobsByChatIDandDay(chatID int64, weekday time.Weekday, tx *bolt.Tx) []BusInfoJob {
	b := tx.Bucket([]byte(userBucket))
	if b == nil {
		return nil
	}

	userKey := []byte(strconv.FormatInt(chatID, 10))
	v := b.Get(userKey)
	storedJobs := []BusInfoJob{}
	json.Unmarshal(v, &storedJobs)

	storedJobsForDay := []BusInfoJob{}
	for _, job := range storedJobs {
		if job.Weekday == weekday {
			storedJobsForDay = append(storedJobsForDay, job)
		}
	}

	return storedJobsForDay
}

// GetJobsByChatID retrieves all bus alarms registered by a user identified by a ChatID
func GetJobsByChatID(chatID int64) []BusInfoJob {
	userKey := []byte(strconv.FormatInt(chatID, 10))
	storedJobs := []BusInfoJob{}

	db, err := bolt.Open(db, 0600, nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	err = db.View(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte(userBucket))
		if b == nil {
			return nil
		}

		v := b.Get(userKey)
		json.Unmarshal(v, &storedJobs)
		return nil
	})

	if err != nil {
		log.Fatalln(err)
	}

	return storedJobs
}

// DeleteJob deletes the given job from the database
func DeleteJob(jobToDelete BusInfoJob) {
	chatID := jobToDelete.ChatID

	userKey := []byte(strconv.FormatInt(chatID, 10))

	db, err := bolt.Open(db, 0600, nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte(userBucket))
		if b == nil {
			return nil
		}

		v := b.Get(userKey)
		storedJobs := []BusInfoJob{}
		json.Unmarshal(v, &storedJobs)

		// Remove job and store the remaining back to the key
		remainingJobs := storedJobs[:0]
		for _, job := range storedJobs {
			if job != jobToDelete {
				remainingJobs = append(remainingJobs, job)
			}
		}
		encRemainingJobs, err := json.Marshal(remainingJobs)
		if err != nil {
			log.Fatalln(err)
		}
		b.Put(userKey, encRemainingJobs)

		// Check and remove from the other Job bucket if ChatID has no jobs for that day anymore
		removedJobDay := jobToDelete.Weekday
		remainingJobsForDay := getJobsByChatIDandDay(chatID, removedJobDay, tx)

		if len(remainingJobsForDay) == 0 {
			deleteChatIDFromDayLookup(chatID, removedJobDay, tx)
		}

		return nil
	})
}

func deleteChatIDFromDayLookup(chatIDToDelete int64, weekday time.Weekday, tx *bolt.Tx) {
	dayKey := []byte(weekday.String())

	b := tx.Bucket([]byte(jobBucket))
	if b == nil {
		log.Fatalln("Unable to open job bucket deletion")
	}

	b.Get(dayKey)
	storedChatIDs := []int64{}
	json.Unmarshal(b.Get(dayKey), &storedChatIDs)

	remainingChatIDs := storedChatIDs[:0]
	for _, chatID := range storedChatIDs {
		if chatID != chatIDToDelete {
			remainingChatIDs = append(remainingChatIDs, chatID)
		}
	}
	encRemainingIDs, err := json.Marshal(remainingChatIDs)
	if err != nil {
		log.Fatalln(err)
	}
	b.Put(dayKey, encRemainingIDs)
}
