package main

import (
	"log"
	"os"
	"time"

	"github.com/yi-jiayu/datamall/v3"
)

// Returns negative if arrival time is unknown
func getMinutesFromNow(arrivingBus datamall.ArrivingBus) float64 {
	if arrivingBus.EstimatedArrival.IsZero() {
		return -1
	}
	delta := arrivingBus.EstimatedArrival.Sub(time.Now()).Minutes()
	if delta < 1 {
		return 0
	}
	return delta
}

// NextBusMinutes, NextBusMinutes2, NextBusMinutes3 will contain negative value
// if arrival information is not available
type busArrivalInformation struct {
	BusStopCode     string
	BusServiceNo    string
	NextBusMinutes  float64
	NextBusMinutes2 float64
	NextBusMinutes3 float64
}

func fetchBusArrivalInformation(busStopCode string, busServiceNo string) busArrivalInformation {
	ltaToken := os.Getenv("LTA_API_TOKEN")
	apiClient := datamall.NewDefaultClient(ltaToken)
	resPayload, err := apiClient.GetBusArrival(busStopCode, busServiceNo)
	if err != nil {
		log.Fatalln(err)
	}

	busArrivalInfo := busArrivalInformation{}
	busArrivalInfo.BusStopCode = resPayload.BusStopCode
	busArrivalInfo.BusServiceNo = resPayload.Services[0].ServiceNo
	busArrivalInfo.NextBusMinutes = getMinutesFromNow(resPayload.Services[0].NextBus)
	busArrivalInfo.NextBusMinutes2 = getMinutesFromNow(resPayload.Services[0].NextBus2)
	busArrivalInfo.NextBusMinutes3 = getMinutesFromNow(resPayload.Services[0].NextBus3)

	return busArrivalInfo
}
