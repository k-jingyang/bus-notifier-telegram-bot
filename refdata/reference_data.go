package refdata

import (
	"encoding/json"
	"log"

	"github.com/boltdb/bolt"
)

// BusRoute contains information about a single bus service at a particular bus stop, including the bus stop's description
type BusRoute struct {
	BusServiceNo string
	BusStop
	Direction    int
	StopSequence int
}

// BusStop contains information about a single bus stop
type BusStop struct {
	BusStopCode string
	Description string
}

// DB contains the operations to store store/retrieve reference data
type DB struct {
	dbFile         string
	busRouteBucket string
	busStopBucket  string
}

// NewRefDataDB returns an initialised instance of the reference data db
func NewRefDataDB(dbFile string) DB {
	return DB{dbFile: dbFile, busRouteBucket: "routes", busStopBucket: "busstops"}
}

// StoreBusRoutes saves bus routes information into the referece data db
func (refDataDB *DB) StoreBusRoutes(busRoutes []BusRoute) {
	busToBusRoutes := make(map[string][]BusRoute)
	for _, busRoute := range busRoutes {
		busToBusRoutes[busRoute.BusServiceNo] = append(busToBusRoutes[busRoute.BusServiceNo], busRoute)
	}

	db, err := bolt.Open(refDataDB.dbFile, 0600, nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(refDataDB.busRouteBucket))
		if err != nil {
			return err
		}

		for busServiceNo, busRoute := range busToBusRoutes {
			key := []byte(busServiceNo)

			value, err := json.Marshal(busRoute)
			if err != nil {
				return err
			}

			b.Put(key, value)
		}

		return nil
	})
}

// StoreBusStops saves bus stop information into the referece data db
func (refDataDB *DB) StoreBusStops(busStops []BusStop) {
	db, err := bolt.Open(refDataDB.dbFile, 0600, nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(refDataDB.busStopBucket))
		if err != nil {
			return err
		}

		for _, busStop := range busStops {
			key := []byte(busStop.BusStopCode)

			value, err := json.Marshal(busStop)
			if err != nil {
				return err
			}

			b.Put(key, value)
		}

		return nil
	})

}

// GetBusRoutesByBusService retrieves the routes (both directions) of a bus service
func (refDataDB *DB) GetBusRoutesByBusService(busServiceNo string) []BusRoute {
	var busRoutes []BusRoute

	db, err := bolt.Open(refDataDB.dbFile, 0600, nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(refDataDB.busRouteBucket))
		if b == nil {
			return nil
		}

		v := b.Get([]byte(busServiceNo))
		json.Unmarshal(v, &busRoutes)

		return nil
	})
	return busRoutes
}

// GetBusStopByBusStopCode retrieves information about a bus stop
func (refDataDB *DB) GetBusStopByBusStopCode(busStopCode string) BusStop {
	var busStops BusStop

	db, err := bolt.Open(refDataDB.dbFile, 0600, nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(refDataDB.busStopBucket))
		if b == nil {
			return nil
		}

		v := b.Get([]byte(busStopCode))
		json.Unmarshal(v, &busStops)

		return nil
	})
	return busStops
}
