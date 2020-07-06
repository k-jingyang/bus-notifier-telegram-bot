package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

type dataMallPayload struct {
	Services    []services
	BusStopCode string
}

type services struct {
	ServiceNo string
	NextBus   nextBus
	NextBus2  nextBus
	NextBus3  nextBus
}

type nextBus struct {
	EstimatedArrival time.Time
}

// Returns negative if arrival time is unknown
func (nextBus nextBus) getMinutesFromNow() float64 {
	if nextBus.EstimatedArrival.IsZero() {
		return -1
	}
	delta := nextBus.EstimatedArrival.Sub(time.Now()).Minutes()
	if delta < 1 {
		return 0
	}
	return delta
}

// NextBusMinutes, NextBusMinutes2, NextBusMinutes3 will contain negative value
// if arrival information is not available
type busArrivalInformation struct {
	BusStopName     string
	BusServiceNo    string
	NextBusMinutes  float64
	NextBusMinutes2 float64
	NextBusMinutes3 float64
}

func fetchBusArrivalInformation(busStopCode string, busServiceNo string) busArrivalInformation {
	var respPayload dataMallPayload
	json.Unmarshal(sendBusArrivalAPIRequest(busStopCode, busServiceNo), &respPayload)

	busArrivalInfo := busArrivalInformation{}
	busArrivalInfo.BusStopName = respPayload.BusStopCode
	busArrivalInfo.BusServiceNo = respPayload.Services[0].ServiceNo
	busArrivalInfo.NextBusMinutes = respPayload.Services[0].NextBus.getMinutesFromNow()
	busArrivalInfo.NextBusMinutes2 = respPayload.Services[0].NextBus2.getMinutesFromNow()
	busArrivalInfo.NextBusMinutes3 = respPayload.Services[0].NextBus3.getMinutesFromNow()

	return busArrivalInfo
}

func sendBusArrivalAPIRequest(busStopCode string, busServiceNo string) []byte {
	resp, err := http.DefaultClient.Do(buildBusArrivalAPIRequest(busStopCode, busServiceNo))
	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return body
}

func buildBusArrivalAPIRequest(busStopCode string, busServiceNo string) *http.Request {
	params := url.Values{}
	params.Add("BusStopCode", busStopCode)
	params.Add("ServiceNo", busServiceNo)

	url := url.URL{
		Scheme:   "http",
		Host:     "datamall2.mytransport.sg",
		Path:     "ltaodataservice/BusArrivalv2",
		RawQuery: params.Encode(),
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		log.Fatal(err)
	}
	ltaToken := os.Getenv("LTA_API_TOKEN")
	req.Header.Add("AccountKey", ltaToken)
	return req
}
