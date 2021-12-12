package scru128

import (
	"bufio"
	crand "crypto/rand"
	mrand "math/rand"
	"testing"
	"time"
)

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
