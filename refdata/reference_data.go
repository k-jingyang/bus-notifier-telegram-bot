package refdata

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
func (db *DB) StoreBusRoutes(busRoutes []BusRoute) {

}

// StoreBusStops saves bus stop information into the referece data db
func (db *DB) StoreBusStops(busStops []BusStop) {

}

// GetBusRoutesByBusService retrieves the routes (both directions) of a bus service
func (db *DB) GetBusRoutesByBusService(busServiceNo string) []BusRoute {
	return nil
}

// GetBusStopByBusStopCode retrieves information about a bus stop
func (db *DB) GetBusStopByBusStopCode(busStopCode string) []BusStop {
	return nil
}
