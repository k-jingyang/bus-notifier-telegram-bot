package main

import (
	"bus-notifier/refdata"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/yi-jiayu/datamall/v3"
)

const refDataDBFile string = "../refdata.db"

// This GO code helps to download
// 1) Bus stops that each bus services
// 2) Road name of each bus stop
// into a boltdb file for consumption by the main app
func main() {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Downloading from LTA API...")
	rawBusRoutes := downloadAllBusRoutes()
	rawBusStops := downloadAllBusStops()

	log.Println("Processing data...")
	busRoutesInfo := processBusRoutes(rawBusRoutes, rawBusStops)
	busStopInfo := processBusStops(rawBusStops)

	log.Println("Storing data into reference data db...")
	refDataDB := refdata.NewRefDataDB(refDataDBFile)
	refDataDB.StoreBusRoutes(busRoutesInfo)
	refDataDB.StoreBusStops(busStopInfo)

	log.Println("Reference data downloaded and stored!")
}

// Both download functions can be generalised
func downloadAllBusRoutes() []datamall.BusRoute {
	ltaToken := os.Getenv("LTA_API_TOKEN")
	apiClient := datamall.NewDefaultClient(ltaToken)
	allBusRoutes := []datamall.BusRoute{}

	busRoutesChan := make(chan []datamall.BusRoute)
	stop := false
	offset := 0
	const concurrentFetches int = 3
	for !stop {
		for i := 0; i < concurrentFetches; i++ {
			go func() {
				response, _ := apiClient.GetBusRoutes(offset)
				busRoutesChan <- response.Value
			}()
			offset += 500
		}
		for i := 0; i < concurrentFetches; i++ {
			data := <-busRoutesChan
			if len(data) == 0 {
				stop = true
			}
			allBusRoutes = append(allBusRoutes, data...)
		}
	}
	return allBusRoutes
}

func processBusRoutes(rawBusRoutes []datamall.BusRoute, rawBusStops []datamall.BusStop) []refdata.BusRoute {
	var processedBusRoutes []refdata.BusRoute

	busStopCodeToDesc := make(map[string]string)
	for _, busStop := range rawBusStops {
		busStopCodeToDesc[busStop.BusStopCode] = busStop.Description
	}

	for _, busRoute := range rawBusRoutes {
		busStop := refdata.BusStop{BusStopCode: busRoute.BusStopCode, Description: busStopCodeToDesc[busRoute.BusStopCode]}
		processedBusRoute := refdata.BusRoute{BusServiceNo: busRoute.ServiceNo, Direction: busRoute.Direction, BusStop: busStop, StopSequence: busRoute.StopSequence}
		processedBusRoutes = append(processedBusRoutes, processedBusRoute)
	}

	return processedBusRoutes
}

func downloadAllBusStops() []datamall.BusStop {
	ltaToken := os.Getenv("LTA_API_TOKEN")
	apiClient := datamall.NewDefaultClient(ltaToken)
	allBusStops := []datamall.BusStop{}

	busStopsChan := make(chan []datamall.BusStop)
	stop := false
	offset := 0
	const concurrentFetches int = 3
	for !stop {
		for i := 0; i < concurrentFetches; i++ {
			go func() {
				response, _ := apiClient.GetBusStops(offset)
				busStopsChan <- response.Value
			}()
			offset += 500
		}
		for i := 0; i < concurrentFetches; i++ {
			data := <-busStopsChan
			if len(data) == 0 {
				stop = true
			}
			allBusStops = append(allBusStops, data...)
		}
	}
	return allBusStops
}

func processBusStops(rawBusStops []datamall.BusStop) []refdata.BusStop {

	var processedBusStops []refdata.BusStop

	for _, busStop := range rawBusStops {
		busStop := refdata.BusStop{BusStopCode: busStop.BusStopCode, Description: busStop.Description}
		processedBusStops = append(processedBusStops, busStop)
	}

	return processedBusStops
}
