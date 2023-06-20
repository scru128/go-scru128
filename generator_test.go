package scru128

import (
	"bufio"
	crand "crypto/rand"
	mrand "math/rand"
	"testing"
	"time"
)

// Generates increasing IDs even with decreasing or constant timestamp
func TestDecreasingOrConstantTimestampReset(t *testing.T) {
	var ts uint64 = 0x0123_4567_89ab
	var g *Generator = NewGenerator()

	prev, _ := g.GenerateOrResetCore(ts, 10_000)
	if prev.Timestamp() != ts {
		t.Fail()
	}

	for i := uint64(0); i < 100_000; i++ {
		var curr Id
		if i < 9_998 {
			curr, _ = g.GenerateOrResetCore(ts-i, 10_000)
		} else {
			curr, _ = g.GenerateOrResetCore(ts-9_998, 10_000)
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

// Breaks increasing order of IDs if timestamp went backwards a lot
func TestTimestampRollbackReset(t *testing.T) {
	var ts uint64 = 0x0123_4567_89ab
	var g *Generator = NewGenerator()

	prev, _ := g.GenerateOrResetCore(ts, 10_000)
	if prev.Timestamp() != ts {
		t.Fail()
	}

	curr, _ := g.GenerateOrResetCore(ts-10_000, 10_000)
	if prev.Cmp(curr) <= 0 {
		t.Fail()
	}
	if curr.Timestamp() != ts-10_000 {
		t.Fail()
	}

	prev = curr
	curr, _ = g.GenerateOrResetCore(ts-10_001, 10_000)
	if prev.Cmp(curr) >= 0 {
		t.Fail()
	}
}

// Generates increasing IDs even with decreasing or constant timestamp
func TestDecreasingOrConstantTimestampAbort(t *testing.T) {
	var ts uint64 = 0x0123_4567_89ab
	var g *Generator = NewGenerator()

	prev, err := g.GenerateOrAbortCore(ts, 10_000)
	if err == ErrClockRollback {
		t.Fail()
	}
	if prev.Timestamp() != ts {
		t.Fail()
	}

	for i := uint64(0); i < 100_000; i++ {
		var curr Id
		if i < 9_998 {
			curr, err = g.GenerateOrAbortCore(ts-i, 10_000)
		} else {
			curr, err = g.GenerateOrAbortCore(ts-9_998, 10_000)
		}
		if err == ErrClockRollback {
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

// Returns error if timestamp went backwards a lot
func TestTimestampRollbackAbort(t *testing.T) {
	var ts uint64 = 0x0123_4567_89ab
	var g *Generator = NewGenerator()

	prev, err := g.GenerateOrAbortCore(ts, 10_000)
	if err == ErrClockRollback {
		t.Fail()
	}
	if prev.Timestamp() != ts {
		t.Fail()
	}

	_, err = g.GenerateOrAbortCore(ts-10_000, 10_000)
	if err != ErrClockRollback {
		t.Fail()
	}

	_, err = g.GenerateOrAbortCore(ts-10_001, 10_000)
	if err != ErrClockRollback {
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
