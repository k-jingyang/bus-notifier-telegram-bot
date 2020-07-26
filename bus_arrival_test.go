package main

import (
	"testing"
)

func TestNoArrivalInformationReturnsNegativeValue(t *testing.T) {
	nextBus := nextBus{}
	minutes := nextBus.getMinutesFromNow()
	if minutes > 0 {
		t.Errorf("Minutes should be negative but it's not")
	}
}
