package main

import (
	"testing"

	"github.com/yi-jiayu/datamall/v3"
)

func TestNoArrivalInformationReturnsNegativeValue(t *testing.T) {
	arrivingBus := datamall.ArrivingBus{}
	minutes := getMinutesFromNow(arrivingBus)
	if minutes > 0 {
		t.Errorf("Minutes should be negative but it's not")
	}
}
