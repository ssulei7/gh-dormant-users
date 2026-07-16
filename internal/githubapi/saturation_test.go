package githubapi

import (
	"testing"
	"time"
)

func TestAdaptiveControllerUsesWindowSaturation(t *testing.T) {
	clock := &fakeClock{now: time.Unix(1000, 0)}
	gate := newConcurrencyGate(5, 15)
	controller := newAdaptiveController(gate, 5, 15, 10, clock.Now)

	for index := range adaptationSampleSize {
		clock.Advance(250 * time.Millisecond)
		controller.observe(time.Second, index < adaptationSampleSize-1)
	}
	if current := gate.snapshot().limit; current != 6 {
		t.Fatalf("expected majority-saturated window to grow, got %d", current)
	}

	for index := range adaptationSampleSize {
		clock.Advance(250 * time.Millisecond)
		controller.observe(time.Second, index == adaptationSampleSize-1)
	}
	if current := gate.snapshot().limit; current != 6 {
		t.Fatalf("single saturated sample should not grow concurrency, got %d", current)
	}
}
