package githubapi

import (
	"context"
	"testing"
	"time"
)

type fakeClock struct {
	now time.Time
}

func (clock *fakeClock) Now() time.Time {
	return clock.now
}

func (clock *fakeClock) Advance(duration time.Duration) {
	clock.now = clock.now.Add(duration)
}

func TestConcurrencyGateCanGrowWhileRequestsWait(t *testing.T) {
	gate := newConcurrencyGate(1, 3)
	if err := gate.acquire(context.Background()); err != nil {
		t.Fatalf("first acquire returned error: %v", err)
	}

	acquired := make(chan struct{})
	go func() {
		_ = gate.acquire(context.Background())
		close(acquired)
	}()

	select {
	case <-acquired:
		t.Fatal("second request acquired before the gate grew")
	case <-time.After(10 * time.Millisecond):
	}

	gate.setLimit(2)
	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("second request did not acquire after the gate grew")
	}
	gate.release()
	gate.release()
}

func TestAdaptiveControllerStaysAtFiveWhenRateCapIsSaturated(t *testing.T) {
	clock := &fakeClock{now: time.Unix(1000, 0)}
	gate := newConcurrencyGate(5, 15)
	controller := newAdaptiveController(gate, 5, 15, 10, clock.Now)

	for range 50 {
		clock.Advance(100 * time.Millisecond)
		controller.observe(100*time.Millisecond, true)
	}

	if current := gate.snapshot().limit; current != 5 {
		t.Fatalf("expected concurrency to remain 5, got %d", current)
	}
}

func TestAdaptiveControllerGrowsForSlowSaturatedRequests(t *testing.T) {
	clock := &fakeClock{now: time.Unix(1000, 0)}
	gate := newConcurrencyGate(5, 15)
	controller := newAdaptiveController(gate, 5, 15, 10, clock.Now)

	for range 5 {
		for range adaptationSampleSize {
			clock.Advance(250 * time.Millisecond)
			controller.observe(time.Second, true)
		}
	}

	if current := gate.snapshot().limit; current != 10 {
		t.Fatalf("expected concurrency to grow to 10, got %d", current)
	}
}

func TestAdaptiveControllerHalvesConcurrencyAndHonorsCooldown(t *testing.T) {
	clock := &fakeClock{now: time.Unix(1000, 0)}
	gate := newConcurrencyGate(5, 15)
	gate.setLimit(12)
	controller := newAdaptiveController(gate, 5, 15, 10, clock.Now)

	controller.onSecondaryLimit()
	if current := gate.snapshot().limit; current != 6 {
		t.Fatalf("expected concurrency to halve to 6, got %d", current)
	}
	controller.onSecondaryLimit()
	if current := gate.snapshot().limit; current != 6 {
		t.Fatalf("concurrent secondary response reduced concurrency twice: %d", current)
	}

	for range adaptationSampleSize {
		clock.Advance(250 * time.Millisecond)
		controller.observe(time.Second, true)
	}

	if current := gate.snapshot().limit; current != 6 {
		t.Fatalf("concurrency grew during cooldown: %d", current)
	}

	clock.Advance(concurrencyCooldown)
	for range adaptationSampleSize {
		clock.Advance(250 * time.Millisecond)
		controller.observe(time.Second, true)
	}
	if current := gate.snapshot().limit; current != 7 {
		t.Fatalf("expected concurrency to resume additive growth, got %d", current)
	}
	if reductions := controller.snapshot().reductions; reductions != 1 {
		t.Fatalf("expected one concurrency reduction, got %d", reductions)
	}
}

func TestCoordinatorValidatesAdaptiveConcurrencyBounds(t *testing.T) {
	_, err := NewCoordinator(Config{
		InitialConcurrency: 5,
		MaxConcurrency:     4,
		RateLimitReserve:   0.1,
	})
	if err == nil {
		t.Fatal("expected initial concurrency above maximum to fail")
	}

	_, err = NewCoordinator(Config{
		InitialConcurrency: 5,
		MaxConcurrency:     16,
		RateLimitReserve:   0.1,
	})
	if err == nil {
		t.Fatal("expected maximum concurrency above 15 to fail")
	}
}
