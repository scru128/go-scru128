package scru128

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"io"
	"sync"
	"time"
)

// Represents a SCRU128 ID generator that encapsulates the monotonic counters
// and other internal states.
//
// This structure must be instantiated by one of the dedicated constructors:
// [NewGenerator] or [NewGeneratorWithRng].
//
// # Generator functions
//
// The generator offers four different methods to generate a SCRU128 ID:
//
//	| Flavor              | Timestamp | Thread- | On big clock rewind |
//	| ------------------- | --------- | ------- | ------------------- |
//	| Generate            | Now       | Safe    | Resets generator    |
//	| GenerateOrAbort     | Now       | Safe    | Returns error       |
//	| GenerateOrResetCore | Argument  | Unsafe  | Resets generator    |
//	| GenerateOrAbortCore | Argument  | Unsafe  | Returns error       |
//
// All of these methods return monotonically increasing IDs unless a `timestamp`
// provided is significantly (by default, more than ten seconds) smaller than
// the one embedded in the immediately preceding ID. If such a significant clock
// rollback is detected, the `Generate` (OrReset) method resets the generator
// and returns a new ID based on the given `timestamp`, while the `OrAbort`
// variants abort and return the [ErrClockRollback] error value. The `Core`
// functions offer low-level thread-unsafe primitives.
type Generator struct {
	timestamp uint64
	counterHi uint32
	counterLo uint32

	// The timestamp at the last renewal of counter_hi field.
	tsCounterHi uint64

	// The random number generator used by the generator.
	rng io.Reader

	lock sync.Mutex
}

// Creates a generator object with the default random number generator.
//
// The crypto/rand random number generator is quite slow for small reads on some
// platforms. In such a case, wrapping crypto/rand with bufio.Reader may result
// in a drastic improvement in the throughput of generator. If the throughput is
// an important issue, check out the following benchmark tests and pass
// bufio.NewReader(rand.Reader) to [NewGeneratorWithRng]:
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

// Generates a new SCRU128 ID object from the current `timestamp`, or resets the
// generator upon significant timestamp rollback.
//
// See the [Generator] type documentation for the description.
//
// This method returns a non-nil err if the random number generator fails.
func (g *Generator) Generate() (id Id, err error) {
	g.lock.Lock()
	defer g.lock.Unlock()
	return g.GenerateOrResetCore(
		uint64(time.Now().UnixMilli()),
		defaultRollbackAllowance,
	)
}

// Generates a new SCRU128 ID object from the current `timestamp`, or returns an
// error upon significant timestamp rollback.
//
// See the [Generator] type documentation for the description.
//
// This method returns a non-nil err if the random number generator fails or
// returns the [ErrClockRollback] err upon significant clock rollback.
func (g *Generator) GenerateOrAbort() (id Id, err error) {
	g.lock.Lock()
	defer g.lock.Unlock()
	return g.GenerateOrAbortCore(
		uint64(time.Now().UnixMilli()),
		defaultRollbackAllowance,
	)
}

// Generates a new SCRU128 ID object from the `timestamp` passed, or resets the
// generator upon significant timestamp rollback.
//
// See the [Generator] type documentation for the description.
//
// The `rollbackAllowance` parameter specifies the amount of timestamp rollback
// that is considered significant. A suggested value is `10_000` (milliseconds).
//
// Unlike [Generator.Generate], this method is NOT thread-safe. The generator
// object should be protected from concurrent accesses using a mutex or other
// synchronization mechanism to avoid race conditions.
//
// This method returns a non-nil err if the random number generator fails.
//
// This method panics if `timestamp` is not a 48-bit positive integer.
func (g *Generator) GenerateOrResetCore(
	timestamp uint64,
	rollbackAllowance uint64,
) (id Id, err error) {
	id, err = g.GenerateOrAbortCore(timestamp, rollbackAllowance)
	if err == ErrClockRollback {
		// reset state and resume
		g.timestamp = 0
		g.tsCounterHi = 0
		id, err = g.GenerateOrAbortCore(timestamp, rollbackAllowance)
	}
	return
}

// Generates a new SCRU128 ID object from the `timestamp` passed, or returns an
// error upon significant timestamp rollback.
//
// See the [Generator] type documentation for the description.
//
// The `rollbackAllowance` parameter specifies the amount of timestamp rollback
// that is considered significant. A suggested value is `10_000` (milliseconds).
//
// Unlike [Generator.GenerateOrAbort], this method is NOT thread-safe. The
// generator object should be protected from concurrent accesses using a mutex
// or other synchronization mechanism to avoid race conditions.
//
// This method returns a non-nil err if the random number generator fails or
// returns the [ErrClockRollback] err upon significant clock rollback.
//
// This method panics if `timestamp` is not a 48-bit positive integer.
func (g *Generator) GenerateOrAbortCore(
	timestamp uint64,
	rollbackAllowance uint64,
) (id Id, err error) {
	if timestamp == 0 || timestamp > maxTimestamp {
		panic("`timestamp` must be a 48-bit positive integer")
	} else if rollbackAllowance > maxTimestamp {
		panic("`rollbackAllowance` out of reasonable range")
	}

	var n uint32
	if timestamp > g.timestamp {
		g.timestamp = timestamp
		n, err = g.randomUint32()
		if err != nil {
			return Id{}, err
		}
		g.counterLo = n & maxCounterLo
	} else if timestamp+rollbackAllowance >= g.timestamp {
		// go on with previous timestamp if new one is not much smaller
		g.counterLo++
		if g.counterLo > maxCounterLo {
			g.counterLo = 0
			g.counterHi++
			if g.counterHi > maxCounterHi {
				g.counterHi = 0
				// increment timestamp at counter overflow
				g.timestamp++
				n, err = g.randomUint32()
				if err != nil {
					return Id{}, err
				}
				g.counterLo = n & maxCounterLo
			}
		}
	} else {
		// abort if clock went backwards to unbearable extent
		return Id{}, ErrClockRollback
	}

	if g.timestamp-g.tsCounterHi >= 1_000 || g.tsCounterHi == 0 {
		g.tsCounterHi = g.timestamp
		n, err = g.randomUint32()
		if err != nil {
			return Id{}, err
		}
		g.counterHi = n & maxCounterHi
	}

	n, err = g.randomUint32()
	if err != nil {
		return Id{}, err
	}
	return FromFields(g.timestamp, g.counterHi, g.counterLo, n), nil
}

// The default timestamp rollback allowance.
const defaultRollbackAllowance = 10_000 // 10 seconds

// The error value returned by [Generator.GenerateOrAbort] and
// [Generator.GenerateOrAbortCore] when the relevant timestamp is significantly
// smaller than the one embedded in the immediately preceding ID generated by
// the generator.
var ErrClockRollback = fmt.Errorf(
	"scru128.Generator: detected unbearable clock rollback")

// Returns a random uint32 value.
func (g *Generator) randomUint32() (uint32, error) {
	b := make([]byte, 4)
	_, err := g.rng.Read(b)
	if err != nil {
		err = fmt.Errorf("scru128.Generator: random number generator error: %w", err)
	}
	_ = b[3] // bounds check hint to compiler
	return uint32(b[3]) | uint32(b[2])<<8 | uint32(b[1])<<16 | uint32(b[0])<<24, err
}
