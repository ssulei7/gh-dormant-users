package githubapi

import (
	"context"
	"math"
	"sort"
	"sync"
	"time"
)

const (
	adaptationSampleSize = 20
	adaptationWindow     = 2 * time.Second
	concurrencyCooldown  = time.Minute
)

type concurrencyGate struct {
	mu       sync.Mutex
	limit    int
	maximum  int
	inFlight int
	peak     int
	changed  chan struct{}
}

type gateSnapshot struct {
	limit    int
	inFlight int
	peak     int
}

func newConcurrencyGate(initial, maximum int) *concurrencyGate {
	return &concurrencyGate{
		limit:   initial,
		maximum: maximum,
		changed: make(chan struct{}),
	}
}

func (g *concurrencyGate) acquire(ctx context.Context) error {
	for {
		g.mu.Lock()
		if g.inFlight < g.limit {
			g.inFlight++
			if g.inFlight > g.peak {
				g.peak = g.inFlight
			}
			g.mu.Unlock()
			return nil
		}
		changed := g.changed
		g.mu.Unlock()

		select {
		case <-changed:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (g *concurrencyGate) release() {
	g.mu.Lock()
	g.inFlight--
	g.signal()
	g.mu.Unlock()
}

func (g *concurrencyGate) setLimit(limit int) {
	g.mu.Lock()
	if limit < 1 {
		limit = 1
	}
	if limit > g.maximum {
		limit = g.maximum
	}
	if limit != g.limit {
		g.limit = limit
		g.signal()
	}
	g.mu.Unlock()
}

func (g *concurrencyGate) snapshot() gateSnapshot {
	g.mu.Lock()
	defer g.mu.Unlock()
	return gateSnapshot{
		limit:    g.limit,
		inFlight: g.inFlight,
		peak:     g.peak,
	}
}

func (g *concurrencyGate) signal() {
	close(g.changed)
	g.changed = make(chan struct{})
}

type adaptiveController struct {
	mu sync.Mutex

	gate        *concurrencyGate
	initial     int
	maximum     int
	targetRate  float64
	now         func() time.Time
	cooldownEnd time.Time
	reductions  int

	windowStart     time.Time
	windowLatencies []time.Duration
	windowSaturated int
	allLatencies    []time.Duration
	firstCompletion time.Time
	lastCompletion  time.Time
	completions     int
}

type adaptiveSnapshot struct {
	initial      int
	current      int
	peak         int
	reductions   int
	p50          time.Duration
	p95          time.Duration
	achievedRate float64
}

func newAdaptiveController(gate *concurrencyGate, initial, maximum int, targetRate float64, now func() time.Time) *adaptiveController {
	return &adaptiveController{
		gate:       gate,
		initial:    initial,
		maximum:    maximum,
		targetRate: targetRate,
		now:        now,
	}
}

func (c *adaptiveController) observe(latency time.Duration, saturated bool) {
	if latency < 0 {
		latency = 0
	}
	now := c.now()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.firstCompletion.IsZero() {
		c.firstCompletion = now
	}
	c.lastCompletion = now
	c.completions++
	c.allLatencies = appendBounded(c.allLatencies, latency, 1024)

	if c.maximum <= c.initial || c.targetRate <= 0 {
		return
	}
	if c.windowStart.IsZero() {
		c.windowStart = now
	}
	c.windowLatencies = append(c.windowLatencies, latency)
	if saturated {
		c.windowSaturated++
	}
	elapsed := now.Sub(c.windowStart)
	if len(c.windowLatencies) < adaptationSampleSize || elapsed < adaptationWindow {
		return
	}

	throughput := float64(len(c.windowLatencies)) / elapsed.Seconds()
	p95 := percentile(c.windowLatencies, 0.95)
	current := c.gate.snapshot().limit
	desired := int(math.Ceil(c.targetRate * p95.Seconds() * 1.1))
	if desired < c.initial {
		desired = c.initial
	}
	if desired > c.maximum {
		desired = c.maximum
	}

	if !now.Before(c.cooldownEnd) &&
		throughput < c.targetRate*0.9 &&
		c.windowSaturated*2 >= len(c.windowLatencies) &&
		desired > current {
		c.gate.setLimit(current + 1)
	}

	c.windowStart = now
	c.windowLatencies = c.windowLatencies[:0]
	c.windowSaturated = 0
}

func (c *adaptiveController) onSecondaryLimit() {
	now := c.now()
	c.mu.Lock()
	defer c.mu.Unlock()

	if now.Before(c.cooldownEnd) {
		return
	}
	current := c.gate.snapshot().limit
	reduced := current / 2
	if reduced < 1 {
		reduced = 1
	}
	if reduced < current {
		c.gate.setLimit(reduced)
		c.reductions++
	}
	c.cooldownEnd = now.Add(concurrencyCooldown)
	c.windowStart = now
	c.windowLatencies = c.windowLatencies[:0]
	c.windowSaturated = 0
}

func (c *adaptiveController) snapshot() adaptiveSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	gate := c.gate.snapshot()
	achievedRate := float64(0)
	elapsed := c.lastCompletion.Sub(c.firstCompletion)
	if c.completions > 1 && elapsed > 0 {
		achievedRate = float64(c.completions-1) / elapsed.Seconds()
	}
	return adaptiveSnapshot{
		initial:      c.initial,
		current:      gate.limit,
		peak:         gate.peak,
		reductions:   c.reductions,
		p50:          percentile(c.allLatencies, 0.50),
		p95:          percentile(c.allLatencies, 0.95),
		achievedRate: achievedRate,
	}
}

func appendBounded(values []time.Duration, value time.Duration, maximum int) []time.Duration {
	if len(values) < maximum {
		return append(values, value)
	}
	copy(values, values[1:])
	values[len(values)-1] = value
	return values
}

func percentile(values []time.Duration, percentileValue float64) time.Duration {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]time.Duration(nil), values...)
	sort.Slice(sorted, func(left, right int) bool {
		return sorted[left] < sorted[right]
	})
	index := int(math.Ceil(percentileValue*float64(len(sorted)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}
