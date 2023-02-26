package scru128

import (
	"bufio"
	"crypto/rand"
	"errors"
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
//	go test -bench Generator
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

// Generates a new SCRU128 ID object from the current timestamp.
//
// This method returns monotonically increasing IDs unless the up-to-date
// timestamp is significantly (by ten seconds or more) smaller than the one
// embedded in the immediately preceding ID. If such a significant clock
// rollback is detected, this method resets the generator state and returns a
// new ID based on the up-to-date timestamp.
//
// This method is thread-safe; multiple threads can call it concurrently. The
// method returns a non-nil err only when the random number generator fails.
func (g *Generator) Generate() (id Id, err error) {
	g.lock.Lock()
	defer g.lock.Unlock()
	return g.GenerateCore(uint64(time.Now().UnixMilli()))
}

// Generates a new SCRU128 ID object from the current timestamp, guaranteeing
// that the returned ID is greater than the immediately preceding one generated
// by the same generator.
//
// This method returns monotonically increasing IDs unless the up-to-date
// timestamp is significantly (by ten seconds or more) smaller than the one
// embedded in the immediately preceding ID. If such a significant clock
// rollback is detected, this method returns ErrClockRollback as err and keeps
// the generator state untouched.
//
// This method is thread-safe; multiple threads can call it concurrently. The
// method returns a non-nil err if the random number generator fails or the
// clock rollback discussed above is detected..
func (g *Generator) GenerateMonotonic() (id Id, err error) {
	g.lock.Lock()
	defer g.lock.Unlock()
	return g.GenerateCoreMonotonic(uint64(time.Now().UnixMilli()))
}

// Generates a new SCRU128 ID object from the timestamp passed.
//
// This method returns monotonically increasing IDs unless a given timestamp is
// significantly (by ten seconds or more) smaller than the one embedded in the
// immediately preceding ID. If such a significant clock rollback is detected,
// this method resets the generator state and returns a new ID based on the
// given argument.
//
// Unlike Generate(), this method is NOT thread-safe. The generator object
// should be protected from concurrent accesses using a mutex or other
// synchronization mechanism to avoid race conditions.
//
// This method panics if the argument is not a 48-bit positive integer and
// returns a non-nil err if the random number generator fails.
func (g *Generator) GenerateCore(timestamp uint64) (id Id, err error) {
	id, err = g.GenerateCoreMonotonic(timestamp)
	if err == ErrClockRollback {
		// reset state and resume
		g.timestamp = 0
		g.tsCounterHi = 0
		id, err = g.GenerateCoreMonotonic(timestamp)
		g.lastStatus = GeneratorStatusClockRollback
	}
	return
}

// Generates a new SCRU128 ID object from the timestamp passed, guaranteeing
// that the returned ID is greater than the immediately preceding one generated
// by the same generator.
//
// This method returns monotonically increasing IDs unless a given timestamp is
// significantly (by ten seconds or more) smaller than the one embedded in the
// immediately preceding ID. If such a significant clock rollback is detected,
// this method returns ErrClockRollback as err and keeps the generator state
// untouched.
//
// Unlike GenerateMonotonic(), this method is NOT thread-safe. The generator
// object should be protected from concurrent accesses using a mutex or other
// synchronization mechanism to avoid race conditions.
//
// This method panics if the argument is not a 48-bit positive integer and
// returns a non-nil err if the random number generator fails or the clock
// rollback discussed above is detected.
func (g *Generator) GenerateCoreMonotonic(timestamp uint64) (id Id, err error) {
	const rollbackAllowance = 10_000 // 10 seconds

	if timestamp == 0 || timestamp > maxTimestamp {
		panic("`timestamp` must be a 48-bit positive integer")
	}

	var n uint32
	if timestamp > g.timestamp {
		g.timestamp = timestamp
		n, err = g.randomUint32()
		if err != nil {
			goto RngError
		}
		g.counterLo = n & maxCounterLo
		g.lastStatus = GeneratorStatusNewTimestamp
	} else if timestamp+rollbackAllowance > g.timestamp {
		// go on with previous timestamp if new one is not much smaller
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
					goto RngError
				}
				g.counterLo = n & maxCounterLo
				g.lastStatus = GeneratorStatusTimestampInc
			}
		}
	} else {
		// abort if clock moves back to unbearable extent
		return Id{}, ErrClockRollback
	}

	if g.timestamp-g.tsCounterHi >= 1_000 || g.tsCounterHi == 0 {
		g.tsCounterHi = g.timestamp
		n, err = g.randomUint32()
		if err != nil {
			goto RngError
		}
		g.counterHi = n & maxCounterHi
	}

	n, err = g.randomUint32()
	if err != nil {
		goto RngError
	}
	return FromFields(g.timestamp, g.counterHi, g.counterLo, n), nil

RngError:
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

// Error value returned by GenerateMonotonic() and GenerateCoreMonotonic() when
// the relevant timestamp is significantly smaller than the one embedded in the
// immediately preceding ID generated by the generator.
var ErrClockRollback = errors.New("scru128: detected unbearable clock rollback")

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
