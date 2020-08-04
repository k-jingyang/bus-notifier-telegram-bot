package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
)

type ScheduledTime struct {
	Hour   int
	Minute int
}

func (s *ScheduledTime) ToString() string {
	return fmt.Sprintf("%02d:%02d", s.Hour, s.Minute)
}

func (s *ScheduledTime) ToCronExpression(day time.Weekday) string {
	return fmt.Sprintf("%d %d * * %d", s.Minute, s.Hour, day)
}

// BusInfoJob contains all information of a registered bus alarm
type BusInfoJob struct {
	ChatID        int64
	BusStopCode   string
	BusServiceNo  string
	ScheduledTime ScheduledTime
	Weekday       time.Weekday
}

// StoredJob contains the operations to store/retrieve/delete registered bus alarm jobs
type StoredJob struct {
	dbFile     string
	userBucket string
	jobBucket  string
}

// NewStoredJob returns an initialised instance of StoredJob
func NewStoredJob(dbFile string) StoredJob {
	return StoredJob{dbFile: dbFile, userBucket: "Users", jobBucket: "Jobs"}
}

// StoreJob stores the registered bus alarm into the database
func (s *StoredJob) StoreJob(newBusInfoJob BusInfoJob) {

	db, err := bolt.Open(s.dbFile, 0600, nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	db.Update(func(tx *bolt.Tx) error {

		s.storeJob(newBusInfoJob, tx)
		s.storeJobForLookup(newBusInfoJob, tx)
		return nil
	})
}

// User bucket: ChatID (Key) -> Registered jobs for this user (Value)
func (s *StoredJob) storeJob(newBusInfoJob BusInfoJob, tx *bolt.Tx) {
	userKey := []byte(strconv.FormatInt(newBusInfoJob.ChatID, 10))
	b, err := tx.CreateBucketIfNotExists([]byte(s.userBucket))
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
func (s *StoredJob) storeJobForLookup(newBusInfoJob BusInfoJob, tx *bolt.Tx) error {
	dayKey := []byte(newBusInfoJob.Weekday.String())
	b, err := tx.CreateBucketIfNotExists([]byte(s.jobBucket))
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
func (s *StoredJob) GetJobsByDay(weekday time.Weekday) []BusInfoJob {
	jobsOnDay := []BusInfoJob{}

	db, err := bolt.Open(s.dbFile, 0600, nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	err = db.View(func(tx *bolt.Tx) error {

		chatIDs := s.getChatIDsByDay(weekday, tx)

		if len(chatIDs) == 0 {
			return nil
		}

		for _, chatID := range chatIDs {
			userJobsOnDay := s.getJobsByChatIDandDay(chatID, weekday, tx)
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

func (s *StoredJob) getChatIDsByDay(weekday time.Weekday, tx *bolt.Tx) []int64 {
	dayKey := []byte(weekday.String())

	b := tx.Bucket([]byte(s.jobBucket))
	if b == nil {
		return nil
	}

	// List of Chat IDs that has jobs for the day
	storedChatIDs := b.Get(dayKey)

	decodedChatIDs := []int64{}
	json.Unmarshal(storedChatIDs, &decodedChatIDs)

	return decodedChatIDs
}

func (s *StoredJob) getJobsByChatIDandDay(chatID int64, weekday time.Weekday, tx *bolt.Tx) []BusInfoJob {
	b := tx.Bucket([]byte(s.userBucket))
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
func (s *StoredJob) GetJobsByChatID(chatID int64) []BusInfoJob {
	userKey := []byte(strconv.FormatInt(chatID, 10))
	storedJobs := []BusInfoJob{}

	db, err := bolt.Open(s.dbFile, 0600, nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	err = db.View(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte(s.userBucket))
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
func (s *StoredJob) DeleteJob(jobToDelete BusInfoJob) {
	chatID := jobToDelete.ChatID

	userKey := []byte(strconv.FormatInt(chatID, 10))

	db, err := bolt.Open(s.dbFile, 0600, nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte(s.userBucket))
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
		remainingJobsForDay := s.getJobsByChatIDandDay(chatID, removedJobDay, tx)

		if len(remainingJobsForDay) == 0 {
			s.deleteChatIDFromDayLookup(chatID, removedJobDay, tx)
		}

		return nil
	})
}

func (s *StoredJob) deleteChatIDFromDayLookup(chatIDToDelete int64, weekday time.Weekday, tx *bolt.Tx) {
	dayKey := []byte(weekday.String())

	b := tx.Bucket([]byte(s.jobBucket))
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
