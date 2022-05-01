package scru128

import (
	"bufio"
	"crypto/rand"
	"encoding/binary"
	"io"
	"sync"
	"time"
)

// Represents a SCRU128 ID generator that encapsulates the monotonic counters
// and other internal states.
//
// This structure must be instantiated by one of the dedicated constructors:
// NewGenerator() or NewGeneratorWithRng(rng io.Reader).
type Generator struct {
	timestamp uint64
	counterHi uint32
	counterLo uint32

	// Timestamp at the last renewal of counter_hi field.
	tsCounterHi uint64

	// Random number generator used by the generator.
	rng io.Reader

	lock sync.Mutex
}

// Creates a generator object with the default random number generator.
//
// The crypto/rand random number generator is quite slow for small reads on some
// platforms. In such a case, wrapping crypto/rand with bufio.Reader may result
// in a drastic improvement in the throughput of generator. If the throughput is
// an important issue, check out the following benchmark tests and pass
// bufio.NewReader(rand.Reader) to NewGeneratorWithRng():
//
//     go test -bench Generator
func NewGenerator() *Generator {
	// use small buffer to avoid both occasional unbearable performance
	// degradation and waste of time and space for unused buffer contents
	br := bufio.NewReaderSize(rand.Reader, 32)
	return NewGeneratorWithRng(br)
}

// Creates a generator object with a specified random number generator. The
// specified random number generator should be cryptographically strong and
// securely seeded.
func NewGeneratorWithRng(rng io.Reader) *Generator {
	return &Generator{rng: rng}
}

// Generates a new SCRU128 ID object.
//
// This method is thread safe; multiple threads can call it concurrently. The
// method returns non-nil err only when the random number generator fails.
func (g *Generator) Generate() (id Id, err error) {
	g.lock.Lock()
	defer g.lock.Unlock()
	return g.generateThreadUnsafe()
}

// Generates a new SCRU128 ID object without overhead for thread safety.
func (g *Generator) generateThreadUnsafe() (id Id, err error) {
	ts := uint64(time.Now().UnixMilli())
	if ts > g.timestamp {
		g.timestamp = ts
		n, err := g.randomUint32()
		if err != nil {
			return Id{}, err
		}
		g.counterLo = n & maxCounterLo
	} else if ts+10_000 > g.timestamp {
		g.counterLo++
		if g.counterLo > maxCounterLo {
			g.counterLo = 0
			g.counterHi++
			if g.counterHi > maxCounterHi {
				g.counterHi = 0
				// increment timestamp at counter overflow
				g.timestamp++
				n, err := g.randomUint32()
				if err != nil {
					return Id{}, err
				}
				g.counterLo = n & maxCounterLo
			}
		}
	} else {
		// reset state if clock moves back more than ten seconds
		g.tsCounterHi = 0
		g.timestamp = ts
		n, err := g.randomUint32()
		if err != nil {
			return Id{}, err
		}
		g.counterLo = n & maxCounterLo
	}

	if g.timestamp-g.tsCounterHi >= 1_000 {
		g.tsCounterHi = g.timestamp
		n, err := g.randomUint32()
		if err != nil {
			return Id{}, err
		}
		g.counterHi = n & maxCounterHi
	}

	n, err := g.randomUint32()
	if err != nil {
		return Id{}, err
	}
	return FromFields(g.timestamp, g.counterHi, g.counterLo, n), nil
}

// Returns a random uint32 value.
func (g *Generator) randomUint32() (uint32, error) {
	buffer := make([]byte, 4)
	_, err := g.rng.Read(buffer)
	return binary.BigEndian.Uint32(buffer), err
}
