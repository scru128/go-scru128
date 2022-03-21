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
type Generator interface {
	// Generates a new SCRU128 ID object.
	Generate() (id Id, err error)

	// Specifies the logger object used by the generator.
	//
	// Logging is disabled by default. Set a logger object to enable logging.
	//
	// The Warn method accepts fmt.Print-style arguments. The interface is
	// compatible with logrus and zap.
	SetLogger(logger interface{ Warn(args ...interface{}) })
}

type generatorImpl struct {
	timestamp uint64
	counterHi uint32
	counterLo uint32

	// Timestamp at the last renewal of counter_hi field.
	tsCounterHi uint64

	// Random number generator used by the generator.
	rng io.Reader

	lock sync.Mutex

	// Logger object used by the generator.
	logger interface{ Warn(args ...interface{}) }
}

// Creates a generator object with the default random number generator.
//
// Generate() of the returned object is thread safe; multiple threads can call
// it concurrently. The method returns non-nil err only when crypto/rand fails.
//
// The crypto/rand random number generator is quite slow for small reads on some
// platforms. In such a case, wrapping crypto/rand with bufio.Reader may result
// in a drastic improvement in the throughput of generator. If the throughput is
// an important issue, check out the following benchmark tests and pass
// bufio.NewReader(rand.Reader) to NewGeneratorWithRng():
//
//     go test -bench Generator
func NewGenerator() Generator {
	// use small buffer to avoid both occasional unbearable performance
	// degradation and waste of time and space for unused buffer contents
	br := bufio.NewReaderSize(rand.Reader, 32)
	return NewGeneratorWithRng(br)
}

// Creates a generator object with a specified random number generator. The
// specified random number generator should be cryptographically strong and
// securely seeded.
//
// Generate() of the returned object is thread safe; multiple threads can call
// it concurrently. The method returns non-nil err only when the random number
// generator fails.
func NewGeneratorWithRng(rng io.Reader) Generator {
	return &generatorImpl{rng: rng}
}

// Generates a new SCRU128 ID object.
func (g *generatorImpl) Generate() (id Id, err error) {
	g.lock.Lock()
	defer g.lock.Unlock()
	return g.generateThreadUnsafe()
}

// Generates a new SCRU128 ID object without overhead for thread safety.
func (g *generatorImpl) generateThreadUnsafe() (id Id, err error) {
	ts := uint64(time.Now().UnixMilli())
	if ts > g.timestamp {
		g.timestamp = ts
		n, err := g.randomUint32()
		if err != nil {
			return Id{}, err
		}
		g.counterLo = n & maxCounterLo
		if ts-g.tsCounterHi >= 1000 {
			g.tsCounterHi = ts
			n, err := g.randomUint32()
			if err != nil {
				return Id{}, err
			}
			g.counterHi = n & maxCounterHi
		}
	} else {
		g.counterLo++
		if g.counterLo > maxCounterLo {
			g.counterLo = 0
			g.counterHi++
			if g.counterHi > maxCounterHi {
				g.counterHi = 0
				g.handleCounterOverflow()
				return g.generateThreadUnsafe()
			}
		}
	}

	n, err := g.randomUint32()
	if err != nil {
		return Id{}, err
	}
	return FromFields(g.timestamp, g.counterHi, g.counterLo, n), nil
}

// Returns a random uint32 value.
func (g *generatorImpl) randomUint32() (uint32, error) {
	buffer := make([]byte, 4)
	_, err := g.rng.Read(buffer)
	return binary.BigEndian.Uint32(buffer), err
}

// Defines the behavior on counter overflow.
//
// Currently, this method busy-waits for the next clock tick and, if the clock
// does not move forward for a while, reinitializes the generator state.
func (g *generatorImpl) handleCounterOverflow() {
	if g.logger != nil {
		g.logger.Warn("counter overflowing; will wait for next clock tick")
	}
	g.tsCounterHi = 0
	for i := 0; i < 1_000_000; i++ {
		if uint64(time.Now().UnixMilli()) > g.timestamp {
			return
		}
	}
	if g.logger != nil {
		g.logger.Warn("reset state as clock did not move for a while")
	}
	g.timestamp = 0
}

// Specifies the logger object used by the generator.
//
// Logging is disabled by default. Set a thread-safe logger to enable logging.
//
// The Warn method accepts fmt.Print-style arguments. The interface is
// compatible with logrus and zap.
func (g *generatorImpl) SetLogger(logger interface{ Warn(args ...interface{}) }) {
	g.logger = logger
}
