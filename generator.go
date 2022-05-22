package scru128

import (
	"bufio"
	"crypto/rand"
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

	// Status code reported at the last generation.
	lastStatus GeneratorStatus

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
	// use small buffer by default to avoid both occasional unbearable performance
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
// This method is thread-safe; multiple threads can call it concurrently. The
// method returns non-nil err only when the random number generator fails.
func (g *Generator) Generate() (id Id, err error) {
	g.lock.Lock()
	defer g.lock.Unlock()
	return g.GenerateCore(uint64(time.Now().UnixMilli()))
}

// Generates a new SCRU128 ID object with the timestamp passed.
//
// Unlike Generate(), this method is NOT thread-safe. The generator object
// should be protected from concurrent accesses using a mutex or other
// synchronization mechanism to avoid race conditions.
//
// This method panics if the argument is not a 48-bit positive integer and
// returns non-nil err if the random number generator fails.
func (g *Generator) GenerateCore(timestamp uint64) (id Id, err error) {
	if timestamp == 0 || timestamp > maxTimestamp {
		panic("`timestamp` must be a 48-bit positive integer")
	}

	var n uint32
	g.lastStatus = GeneratorStatusNewTimestamp
	if timestamp > g.timestamp {
		g.timestamp = timestamp
		n, err = g.randomUint32()
		if err != nil {
			goto Error
		}
		g.counterLo = n & maxCounterLo
	} else if timestamp+10_000 > g.timestamp {
		g.counterLo++
		g.lastStatus = GeneratorStatusCounterLoInc
		if g.counterLo > maxCounterLo {
			g.counterLo = 0
			g.counterHi++
			g.lastStatus = GeneratorStatusCounterHiInc
			if g.counterHi > maxCounterHi {
				g.counterHi = 0
				// increment timestamp at counter overflow
				g.timestamp++
				n, err = g.randomUint32()
				if err != nil {
					goto Error
				}
				g.counterLo = n & maxCounterLo
				g.lastStatus = GeneratorStatusTimestampInc
			}
		}
	} else {
		// reset state if clock moves back by ten seconds or more
		g.tsCounterHi = 0
		g.timestamp = timestamp
		n, err = g.randomUint32()
		if err != nil {
			goto Error
		}
		g.counterLo = n & maxCounterLo
		g.lastStatus = GeneratorStatusClockRollback
	}

	if g.timestamp-g.tsCounterHi >= 1_000 {
		g.tsCounterHi = g.timestamp
		n, err = g.randomUint32()
		if err != nil {
			goto Error
		}
		g.counterHi = n & maxCounterHi
	}

	n, err = g.randomUint32()
	if err != nil {
		goto Error
	}
	return FromFields(g.timestamp, g.counterHi, g.counterLo, n), nil

Error:
	g.lastStatus = GeneratorStatusError
	return Id{}, err
}

// Returns a GeneratorStatus code that indicates the internal state involved in
// the last generation of ID.
//
// Note that the generator object should be protected from concurrent accesses
// during the sequential calls to a generation method and this method to avoid
// race conditions.
func (g *Generator) LastStatus() GeneratorStatus {
	return g.lastStatus
}

// Status code returned by LastStatus() method.
type GeneratorStatus string

const (
	// Indicates that the generator has yet to generate an ID.
	GeneratorStatusNotExecuted GeneratorStatus = ""

	// Indicates that the latest timestamp was used because it was greater than
	// the previous one.
	GeneratorStatusNewTimestamp GeneratorStatus = "NewTimestamp"

	// Indicates that counter_lo was incremented because the latest timestamp was
	// no greater than the previous one.
	GeneratorStatusCounterLoInc GeneratorStatus = "CounterLoInc"

	// Indicates that counter_hi was incremented because counter_lo reached its
	// maximum value.
	GeneratorStatusCounterHiInc GeneratorStatus = "CounterHiInc"

	// Indicates that the previous timestamp was incremented because counter_hi
	// reached its maximum value.
	GeneratorStatusTimestampInc GeneratorStatus = "TimestampInc"

	// Indicates that the monotonic order of generated IDs was broken because the
	// latest timestamp was less than the previous one by ten seconds or more.
	GeneratorStatusClockRollback GeneratorStatus = "ClockRollback"

	// Indicates that the previous generation failed.
	GeneratorStatusError GeneratorStatus = "Error"
)

// Returns a random uint32 value.
func (g *Generator) randomUint32() (uint32, error) {
	b := make([]byte, 4)
	_, err := g.rng.Read(b)
	_ = b[3] // bounds check hint to compiler
	return uint32(b[3]) | uint32(b[2])<<8 | uint32(b[1])<<16 | uint32(b[0])<<24, err
}
