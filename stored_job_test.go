package main

import (
	"log"
	"os"
	"testing"
	"time"
)

func TestCreateReadDeleteStoredJobs(t *testing.T) {
	storedJobDB = NewJobDB("test.db")

	timeToExecute := ScheduledTime{17, 20}
	busInfoJob := BusInfoJob{12345, "43411", "506", timeToExecute, time.Monday}

	storedJobDB.StoreJob(busInfoJob)

	storedJobs := storedJobDB.GetJobsByChatID(12345)
	if len(storedJobs) != 1 || storedJobs[0].BusStopCode != "43411" || storedJobs[0].BusServiceNo != "506" || storedJobs[0].ScheduledTime != timeToExecute || storedJobs[0].Weekday != time.Monday {
		t.Errorf("Bus info job not stored correctly")
	}

	storedJobDB.DeleteJob(busInfoJob)

	storedJobsByChatID := storedJobDB.GetJobsByChatID(12345)
	storedJobsByDay := storedJobDB.GetJobsByDay(time.Monday)
	if len(storedJobsByChatID) > 0 || len(storedJobsByDay) > 0 {
		log.Println("storedJobsByChatID: {}", storedJobsByChatID)
		log.Println("storedJobsByDay: {}", storedJobsByDay)
		t.Errorf("Bus info job not deleted correctly")
	}
	os.Remove("test.db")
}
