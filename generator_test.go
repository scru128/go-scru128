package scru128

import (
	"bufio"
	crand "crypto/rand"
	mrand "math/rand"
	"testing"
	"time"
)

// Generates increasing IDs even with decreasing or constant timestamp
func TestDecreasingOrConstantTimestamp(t *testing.T) {
	var ts uint64 = 0x0123_4567_89ab
	var g *Generator = NewGenerator()
	if g.LastStatus() != GeneratorStatusNotExecuted {
		t.Fail()
	}

	prev, _ := g.GenerateCore(ts)
	if g.LastStatus() != GeneratorStatusNewTimestamp {
		t.Fail()
	}
	if prev.Timestamp() != ts {
		t.Fail()
	}

	for i := uint64(0); i < 100_000; i++ {
		var curr Id
		if i < 9_998 {
			curr, _ = g.GenerateCore(ts - i)
		} else {
			curr, _ = g.GenerateCore(ts - 9_998)
		}
		if g.LastStatus() != GeneratorStatusCounterLoInc &&
			g.LastStatus() != GeneratorStatusCounterHiInc &&
			g.LastStatus() != GeneratorStatusTimestampInc {
			t.Fail()
		}
		if prev.Cmp(curr) >= 0 {
			t.Fail()
		}
		prev = curr
	}
	if prev.Timestamp() < ts {
		t.Fail()
	}
}

// Breaks increasing order of IDs if timestamp moves backward a lot
func TestTimestampRollback(t *testing.T) {
	var ts uint64 = 0x0123_4567_89ab
	var g *Generator = NewGenerator()
	if g.LastStatus() != GeneratorStatusNotExecuted {
		t.Fail()
	}

	prev, _ := g.GenerateCore(ts)
	if g.LastStatus() != GeneratorStatusNewTimestamp {
		t.Fail()
	}
	if prev.Timestamp() != ts {
		t.Fail()
	}

	curr, _ := g.GenerateCore(ts - 10_000)
	if g.LastStatus() != GeneratorStatusClockRollback {
		t.Fail()
	}
	if prev.Cmp(curr) <= 0 {
		t.Fail()
	}
	if curr.Timestamp() != ts-10_000 {
		t.Fail()
	}

	prev = curr
	curr, _ = g.GenerateCore(ts - 10_001)
	if g.LastStatus() != GeneratorStatusCounterLoInc &&
		g.LastStatus() != GeneratorStatusCounterHiInc &&
		g.LastStatus() != GeneratorStatusTimestampInc {
		t.Fail()
	}
	if prev.Cmp(curr) >= 0 {
		t.Fail()
	}
}

// Generates increasing IDs even with decreasing or constant timestamp
func TestDecreasingOrConstantTimestampNoRewind(t *testing.T) {
	var ts uint64 = 0x0123_4567_89ab
	var g *Generator = NewGenerator()
	if g.LastStatus() != GeneratorStatusNotExecuted {
		t.Fail()
	}

	prev, err := g.GenerateCoreNoRewind(ts, 10_000)
	if err == ErrClockRollback {
		t.Fail()
	}
	if g.LastStatus() != GeneratorStatusNewTimestamp {
		t.Fail()
	}
	if prev.Timestamp() != ts {
		t.Fail()
	}

	for i := uint64(0); i < 100_000; i++ {
		var curr Id
		if i < 9_998 {
			curr, err = g.GenerateCoreNoRewind(ts-i, 10_000)
		} else {
			curr, err = g.GenerateCoreNoRewind(ts-9_998, 10_000)
		}
		if err == ErrClockRollback {
			t.Fail()
		}
		if g.LastStatus() != GeneratorStatusCounterLoInc &&
			g.LastStatus() != GeneratorStatusCounterHiInc &&
			g.LastStatus() != GeneratorStatusTimestampInc {
			t.Fail()
		}
		if prev.Cmp(curr) >= 0 {
			t.Fail()
		}
		prev = curr
	}
	if prev.Timestamp() < ts {
		t.Fail()
	}
}

// Returns ErrClockRollback if timestamp moves backward a lot
func TestTimestampRollbackNoRewind(t *testing.T) {
	var ts uint64 = 0x0123_4567_89ab
	var g *Generator = NewGenerator()
	if g.LastStatus() != GeneratorStatusNotExecuted {
		t.Fail()
	}

	prev, err := g.GenerateCoreNoRewind(ts, 10_000)
	if err == ErrClockRollback {
		t.Fail()
	}
	if g.LastStatus() != GeneratorStatusNewTimestamp {
		t.Fail()
	}
	if prev.Timestamp() != ts {
		t.Fail()
	}

	_, err = g.GenerateCoreNoRewind(ts-10_000, 10_000)
	if err != ErrClockRollback {
		t.Fail()
	}
	if g.LastStatus() != GeneratorStatusNewTimestamp {
		t.Fail()
	}

	_, err = g.GenerateCoreNoRewind(ts-10_001, 10_000)
	if err != ErrClockRollback {
		t.Fail()
	}
	if g.LastStatus() != GeneratorStatusNewTimestamp {
		t.Fail()
	}
}

func BenchmarkGeneratorDefault(b *testing.B) {
	g := NewGenerator()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.Generate()
	}
}

func BenchmarkGeneratorBufferedCryptoRand(b *testing.B) {
	g := NewGeneratorWithRng(bufio.NewReader(crand.Reader))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.Generate()
	}
}

func BenchmarkGeneratorInsecureMathRand(b *testing.B) {
	g := NewGeneratorWithRng(mrand.New(mrand.NewSource(time.Now().UnixMilli())))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.Generate()
	}
}
