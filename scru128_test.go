package scru128

import (
	"fmt"
	"math"
	"regexp"
	"sync"
	"testing"
	"time"
)

var samples = make([]string, 100_000)

func init() {
	for i := 0; i < len(samples); i++ {
		samples[i] = NewString()
	}
}

// Generates 26-digit canonical string
func TestFormat(t *testing.T) {
	re := regexp.MustCompile(`^[0-7][0-9A-V]{25}$`)
	for _, e := range samples {
		if !re.MatchString(e) {
			t.Fail()
		}
	}
}

// Generates 100k identifiers without collision
func TestUniqueness(t *testing.T) {
	set := make(map[string]struct{}, len(samples))
	for _, e := range samples {
		set[e] = struct{}{}
	}
	if len(set) != len(samples) {
		t.Fail()
	}
}

// Generates sortable string representation by creation time
func TestOrder(t *testing.T) {
	for i := 1; i < len(samples); i++ {
		if samples[i-1] >= samples[i] {
			t.Fail()
		}
	}
}

// Encodes up-to-date timestamp
func TestTimestamp(t *testing.T) {
	epoch := time.Date(2020, 1, 1, 0, 0, 0, 0, time.FixedZone("UTC", 0))
	var g Generator = NewGenerator()
	for i := 0; i < 10_000; i++ {
		tsNow := time.Since(epoch).Milliseconds()
		x, _ := g.Generate()
		if math.Abs(float64(tsNow-int64(x.Timestamp()))) >= 16 {
			t.Fail()
		}
	}
}

// Encodes unique sortable pair of timestamp and counter
func TestTimestampAndCounter(t *testing.T) {
	prev, _ := Parse(samples[0])
	for _, e := range samples[1:] {
		curr, _ := Parse(e)
		if !(prev.Timestamp() < curr.Timestamp() ||
			(prev.Timestamp() == curr.Timestamp() &&
				prev.Counter() < curr.Counter())) {
			t.Fail()
		}
		prev = curr
	}
}

// Generates no IDs sharing same timestamp and counter under multithreading
func TestThreading(t *testing.T) {
	results := make(chan Id, 4*10_000)

	group := new(sync.WaitGroup)
	for i := 0; i < 4; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for i := 0; i < 10_000; i++ {
				results <- New()
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		set := make(map[string]struct{}, 4*10_000)
		for e := range results {
			set[fmt.Sprintf("%011x-%07x", e.Timestamp(), e.Counter())] = struct{}{}
		}
		if len(set) != 4*10_000 {
			t.Fail()
		}
	}()

	group.Wait()
	close(results)
	<-done
}
