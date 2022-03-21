// SCRU128: Sortable, Clock and Random number-based Unique identifier
//
// SCRU128 ID is yet another attempt to supersede UUID in the use cases that
// need decentralized, globally unique time-ordered identifiers. SCRU128 is
// inspired by ULID and KSUID and has the following features:
//
//   - 128-bit unsigned integer type
//   - Sortable by generation time (as integer and as text)
//   - 26-digit case-insensitive portable textual representation
//   - 44-bit biased millisecond timestamp that ensures remaining life of 550
//     years
//   - Up to 268 million time-ordered but unpredictable unique IDs per
//     millisecond
//   - 84-bit _layered_ randomness for collision resistance
//
// See SCRU128 Specification for details: https://github.com/scru128/spec
package scru128

// Maximum value of 24-bit counter_hi field.
const maxCounterHi uint32 = 0xff_ffff

// Maximum value of 24-bit counter_lo field.
const maxCounterLo uint32 = 0xff_ffff

var defaultGenerator = NewGenerator()

// Generates a new SCRU128 ID object, or panics if crypto/rand fails.
//
// This function is thread safe; multiple threads can call it concurrently.
func New() Id {
	id, err := defaultGenerator.Generate()
	if err != nil {
		panic(err)
	}
	return id
}

// Generates a new SCRU128 ID encoded in the 26-digit canonical string
// representation, or panics if crypto/rand fails.
//
// This function is thread safe. Use this to quickly get a new SCRU128 ID as a
// string.
func NewString() string {
	return New().String()
}
