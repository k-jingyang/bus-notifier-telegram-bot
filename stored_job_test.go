package main

import (
	"testing"
	"time"
)

func TestCreateReadDeleteStoredJobs(t *testing.T) {
	timeToExecute := scheduledTime{17, 20}
	busInfoJob := BusInfoJob{12345, "43411", "506", timeToExecute, time.Monday}

	StoreJob(busInfoJob)

	storedJobs := GetJobsByChatID(12345)
	if len(storedJobs) != 1 || storedJobs[0].BusStopCode != "43411" || storedJobs[0].BusServiceNo != "506" || storedJobs[0].ScheduledTime != timeToExecute || storedJobs[0].Weekday != time.Monday {
		t.Errorf("Bus info job not stored correctly")
	}

	DeleteJob(busInfoJob)

	storedJobsByChatID := GetJobsByChatID(12345)
	storedJobsByDay := GetJobsByDay(time.Monday)
	if len(storedJobsByChatID) > 0 || len(storedJobsByDay) > 0 {
		t.Errorf("Bus info job not delete correctly")
	}
}
