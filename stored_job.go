package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/boltdb/bolt"
)

const db string = "job.db"

type scheduledJobs struct {
	TimeToExecute scheduledTime
	BusInfoJobs   []busInfoJob
}

type scheduledTime struct {
	Hour   int
	Minute int
}

func (s *scheduledTime) toCronExpression(day time.Weekday) string {
	return fmt.Sprintf("%d %d * * %d", s.Minute, s.Hour, day)
}

func (s *scheduledTime) toKey() []byte {
	// Year, Month, Day is arbitrary. Hour and minute are encapsulated inside time.Time to make it sortable.
	key, err := time.Date(1991, 2, 4, s.Hour, s.Minute, 0, 0, time.Local).MarshalText()
	if err != nil {
		log.Fatalln(err)
	}
	return key
}

func (s *scheduledTime) fromKey(key []byte) {
	var storedTime time.Time
	err := storedTime.UnmarshalText(key)
	if err != nil {
		log.Fatalln(err)
	}
	s.Hour = storedTime.Hour()
	s.Minute = storedTime.Minute()
}

type busInfoJob struct {
	ChatID       int64
	BusStopCode  string
	BusServiceNo string
}

func addJob(newBusInfoJob busInfoJob, weekday time.Weekday, timeToExecute scheduledTime) {
	key := timeToExecute.toKey()
	log.Println("Time of new job:", string(key))

	db, err := bolt.Open(db, 0600, nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(weekday.String()))
		if err != nil {
			log.Fatalln(err)
		}

		storedJobs := b.Get(key)
		if storedJobs == nil {
			encBusInfoJobs, err := json.Marshal([]busInfoJob{newBusInfoJob})
			if err != nil {
				log.Fatalln(err)
			}
			log.Println("New job:", newBusInfoJob)
			b.Put(key, encBusInfoJobs)
		} else {
			existingBusInfoJobs := []busInfoJob{}
			json.Unmarshal(storedJobs, &existingBusInfoJobs)

			for _, s := range existingBusInfoJobs {
				if newBusInfoJob == s {
					log.Println("Job already exists:", newBusInfoJob)
					return nil
				}
			}
			encBusInfoJobs, err := json.Marshal(append(existingBusInfoJobs, newBusInfoJob))
			if err != nil {
				log.Fatalln(err)
			}

			log.Println("Adding to existing jobs", append(existingBusInfoJobs, newBusInfoJob))
			b.Put(key, encBusInfoJobs)
		}

		return nil
	})
}

func getJobsForDay(weekday time.Weekday) []scheduledJobs {
	jobsOnDay := []scheduledJobs{}

	db, err := bolt.Open(db, 0600, nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(weekday.String()))
		if b == nil {
			return nil
		}

		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			storedBusInfoJobs := []busInfoJob{}
			json.Unmarshal(v, &storedBusInfoJobs)
			var time scheduledTime
			time.fromKey(k)
			jobsAtTime := scheduledJobs{time, storedBusInfoJobs}
			jobsOnDay = append(jobsOnDay, jobsAtTime)
		}
		return nil
	})

	if err != nil {
		log.Fatalln(err)
	}

	return jobsOnDay
}
