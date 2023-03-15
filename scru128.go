// SCRU128: Sortable, Clock and Random number-based Unique identifier
//
// SCRU128 ID is yet another attempt to supersede UUID for the users who need
// decentralized, globally unique time-ordered identifiers. SCRU128 is inspired
// by ULID and KSUID and has the following features:
//
//   - 128-bit unsigned integer type
//   - Sortable by generation time (as integer and as text)
//   - 25-digit case-insensitive textual representation (Base36)
//   - 48-bit millisecond Unix timestamp that ensures useful life until year
//     10889
//   - Up to 281 trillion time-ordered but unpredictable unique IDs per
//     millisecond
//   - 80-bit three-layer randomness for global uniqueness
//
// See SCRU128 Specification for details: https://github.com/scru128/spec
package scru128

// Maximum value of 48-bit timestamp field.
const maxTimestamp uint64 = 0xffff_ffff_ffff

// Maximum value of 24-bit counter_hi field.
const maxCounterHi uint32 = 0xff_ffff

// Maximum value of 24-bit counter_lo field.
const maxCounterLo uint32 = 0xff_ffff

var globalGenerator = NewGenerator()

// Generates a new SCRU128 ID object using the global generator, or panics if
// crypto/rand fails.
//
// This function is thread-safe; multiple threads can call it concurrently.
func New() Id {
	id, err := globalGenerator.Generate()
	if err != nil {
		panic(err)
	}
	return id
}

// Generates a new SCRU128 ID encoded in the 26-digit canonical string
// representation using the global generator, or panics if crypto/rand fails.
//
// This function is thread-safe. Use this to quickly get a new SCRU128 ID as a
// string.
func NewString() string {
	return New().String()
}
