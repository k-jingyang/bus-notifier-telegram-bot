package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/boltdb/bolt"
)

type ScheduledJobs struct {
	timeToExcuete ScheduledTime
	busInfoJobs   []BusInfoJob
}

type ScheduledTime struct {
	Hour   int
	Minute int
}

func (s *ScheduledTime) toKey() []byte {
	// Year, Month, Day is arbitrary. Hour and minute are encapsulated inside time.Time to make it sortable.
	key, err := time.Date(1991, 2, 4, s.Hour, s.Minute, 0, 0, time.Local).MarshalText()
	if err != nil {
		log.Fatal(err)
	}
	return key
}

func (s *ScheduledTime) fromKey(key []byte) {
	var storedTime time.Time
	err := storedTime.UnmarshalText(key)
	if err != nil {
		log.Fatal(err)
	}
	s.Hour = storedTime.Hour()
	s.Minute = storedTime.Minute()
}

type BusInfoJob struct {
	ChatID       string
	BusStopCode  string
	BusServiceNo string
}

func addJob(newBusInfoJob BusInfoJob, weekday time.Weekday, timeToExecute ScheduledTime) {
	key := timeToExecute.toKey()
	log.Println("Time:", string(key))

	db, err := bolt.Open("my.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(weekday.String()))
		if err != nil {
			log.Fatal(err)
		}

		storedJobs := b.Get(key)
		if storedJobs == nil {
			encBusInfoJobs, err := json.Marshal([]BusInfoJob{newBusInfoJob})
			if err != nil {
				log.Fatal(err)
			}
			log.Println("New job", newBusInfoJob)
			b.Put(key, encBusInfoJobs)
		} else {
			existingBusInfoJobs := []BusInfoJob{}
			json.Unmarshal(storedJobs, &existingBusInfoJobs)

			for _, s := range existingBusInfoJobs {
				if newBusInfoJob == s {
					log.Println("Job already exists:", newBusInfoJob)
					return nil
				}
			}
			encBusInfoJobs, err := json.Marshal(append(existingBusInfoJobs, newBusInfoJob))
			if err != nil {
				log.Fatal(err)
			}

			log.Println("Adding to existing jobs", append(existingBusInfoJobs, newBusInfoJob))
			b.Put(key, encBusInfoJobs)
		}

		return nil
	})
}

func getJobsForDay(weekday time.Weekday) []ScheduledJobs {
	jobsOnDay := []ScheduledJobs{}

	db, err := bolt.Open("my.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(weekday.String()))
		if b == nil {
			log.Println("No scheduled jobs on", weekday.String())
			return nil
		}

		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			storedBusInfoJobs := []BusInfoJob{}
			json.Unmarshal(v, &storedBusInfoJobs)
			var time ScheduledTime
			time.fromKey(k)
			jobsAtTime := ScheduledJobs{time, storedBusInfoJobs}
			jobsOnDay = append(jobsOnDay, jobsAtTime)
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	return jobsOnDay
}
