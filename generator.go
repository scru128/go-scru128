package scru128

import (
	"crypto/rand"
	"encoding/binary"
	"io"
	"sync"
	"time"
)

// Represents a SCRU128 ID generator that encapsulates the monotonic counter and
// other internal states.
type Generator interface {
	// Generates a new SCRU128 ID object.
	Generate() (id Id, err error)
}

type generatorImpl struct {
	// Timestamp at last generation.
	tsLastGen uint64

	// Counter at last generation.
	counter uint32

	// Timestamp at last renewal of perSecRandom.
	tsLastSec uint64

	// Per-second random value at last generation.
	perSecRandom uint32

	// Maximum number of checking the system clock until it goes forward.
	nClockCheckMax int

	lock sync.Mutex
	rng  io.Reader
}

// Creates a generator object with the default random number generator.
//
// Generate() of the returned object is thread safe; multiple threads can call
// it concurrently. The method returns non-nil err only when crypto/rand fails.
//
// The underlying crypto/rand random number generator is quite slow for small
// reads on some platforms. Wrapping crypto/rand with bufio may drastically
// improve the throughput of generator in such a case. Check the following
// benchmark results and use NewGeneratorWithRng() if necessary:
//
//     go test -bench Generator
func NewGenerator() Generator {
	return NewGeneratorWithRng(rand.Reader)
}

// Creates a generator object with a specified random number generator. The
// specified random number generator should be cryptographically strong and
// securely seeded.
//
// Generate() of the returned object is thread safe; multiple threads can call
// it concurrently. The method returns non-nil err only when the random number
// generator fails.
func NewGeneratorWithRng(rng io.Reader) Generator {
	return &generatorImpl{nClockCheckMax: 1_000_000, rng: rng}
}

// Generates a new SCRU128 ID object.
func (g *generatorImpl) Generate() (id Id, err error) {
	g.lock.Lock()
	defer g.lock.Unlock()
	return g.generateThreadUnsafe()
}

// Generates a new SCRU128 ID object without overhead for thread safety.
func (g *generatorImpl) generateThreadUnsafe() (id Id, err error) {
	// update timestamp and counter
	tsNow := uint64(time.Now().UnixMilli())
	if tsNow > g.tsLastGen {
		g.tsLastGen = tsNow
		n, err := g.randomUint32()
		if err != nil {
			return Id{}, err
		}
		g.counter = n & maxCounter
	} else if g.counter++; g.counter > maxCounter {
		if Logger != nil {
			Logger.Info("counter limit reached; will wait until clock goes forward")
		}
		nClockCheck := 0
		for tsNow <= g.tsLastGen {
			tsNow = uint64(time.Now().UnixMilli())
			if nClockCheck++; nClockCheck > g.nClockCheckMax {
				if Logger != nil {
					Logger.Warn("reset state as clock did not go forward")
				}
				g.tsLastSec = 0
				break
			}
		}

		g.tsLastGen = tsNow
		n, err := g.randomUint32()
		if err != nil {
			return Id{}, err
		}
		g.counter = n & maxCounter
	}

	// update perSecRandom
	if g.tsLastGen-g.tsLastSec > 1_000 {
		g.tsLastSec = g.tsLastGen
		n, err := g.randomUint32()
		if err != nil {
			return Id{}, err
		}
		g.perSecRandom = n & maxPerSecRandom
	}

	n, err := g.randomUint32()
	if err != nil {
		return Id{}, err
	}
	return FromFields(tsNow-TimestampBias, g.counter, g.perSecRandom, n), nil
}

// Returns a random uint32 value.
func (g *generatorImpl) randomUint32() (uint32, error) {
	buffer := make([]byte, 4)
	_, err := g.rng.Read(buffer)
	return binary.BigEndian.Uint32(buffer), err
}

// Specifies the logger object used in the package.
//
// Logging is disabled by default. Set a thread-safe logger to enable logging.
//
// Each method accepts fmt.Print-style arguments. The interface is compatible
// with logrus and zap.
var Logger interface {
	Error(args ...interface{})
	Warn(args ...interface{})
	Info(args ...interface{})
} = nil
