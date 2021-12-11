package scru128

import (
	"bytes"
	"errors"
	"regexp"
	"strconv"
)

// Represents a SCRU128 ID and provides various converters and comparison
// operators.
type Id [16]byte

// Creates a SCRU128 ID object from field values.
func FromFields(
	timestamp uint64,
	counter uint32,
	perSecRandom uint32,
	perGenRandom uint32,
) Id {
	if timestamp < 0 ||
		counter < 0 ||
		perSecRandom < 0 ||
		perGenRandom < 0 ||
		timestamp > 0xFFF_FFFF_FFFF ||
		counter > maxCounter ||
		perSecRandom > maxPerSecRandom ||
		perGenRandom > 0xFFFF_FFFF {
		panic("invalid field value")
	}

	return Id{
		byte(timestamp >> 36),
		byte(timestamp >> 28),
		byte(timestamp >> 20),
		byte(timestamp >> 12),
		byte(timestamp >> 4),
		byte(timestamp<<4) | byte(counter>>24),
		byte(counter >> 16),
		byte(counter >> 8),
		byte(counter),
		byte(perSecRandom >> 16),
		byte(perSecRandom >> 8),
		byte(perSecRandom),
		byte(perGenRandom >> 24),
		byte(perGenRandom >> 16),
		byte(perGenRandom >> 8),
		byte(perGenRandom),
	}
}

// Creates a SCRU128 ID object from a 26-digit string representation.
func Parse(strValue string) (id Id, err error) {
	err = id.UnmarshalText([]byte(strValue))
	return
}

// Returns the 44-bit millisecond timestamp field value.
func (bs Id) Timestamp() uint64 {
	return bytesToUint64(bs[0:6]) >> 4
}

// Returns the 28-bit per-timestamp monotonic counter field value.
func (bs Id) Counter() uint32 {
	return uint32(bytesToUint64(bs[5:9])) & maxCounter
}

// Returns the 24-bit per-second randomness field value.
func (bs Id) PerSecRandom() uint32 {
	return uint32(bytesToUint64(bs[9:12]))
}

// Returns the 32-bit per-generation randomness field value.
func (bs Id) PerGenRandom() uint32 {
	return uint32(bytesToUint64(bs[12:16]))
}

// Returns the 26-digit canonical string representation.
func (bs Id) String() string {
	buffer, _ := bs.MarshalText()
	return string(buffer)
}

// Returns -1, 0, and 1 if the object is less than, equal to, and greater than
// the argument, respectively.
func (bs Id) Cmp(other Id) int {
	return bytes.Compare(bs[:], other[:])
}

// Translates a big-endian byte sequence into uint64.
func bytesToUint64(bigEndian []byte) uint64 {
	var buffer uint64
	for _, v := range bigEndian {
		buffer <<= 8
		buffer |= uint64(v)
	}
	return buffer
}

// See encoding.BinaryMarshaler
func (bs Id) MarshalBinary() (data []byte, err error) {
	return bs[:], nil
}

// See encoding.BinaryUnmarshaler
func (bs *Id) UnmarshalBinary(data []byte) error {
	if len(bs) != len(data) {
		return errors.New("not a 128-bit byte array")
	}

	copy(bs[:], data)
	return nil
}

// Digit characters used in the base 32 notation.
var digits = []byte("0123456789ABCDEFGHIJKLMNOPQRSTUV")

// See encoding.TextMarshaler
func (bs Id) MarshalText() (text []byte, err error) {
	text = make([]byte, 26)
	text[0] = digits[bs[0]>>5]
	text[1] = digits[bs[0]&31]

	// process three 40-bit (5-byte / 8-digit) groups
	for i := 0; i < 3; i++ {
		buffer := bytesToUint64(bs[1+i*5 : 6+i*5])
		for j := 0; j < 8; j++ {
			text[9+i*8-j] = digits[buffer&31]
			buffer >>= 5
		}
	}
	return
}

var validPattern = regexp.MustCompile(`^[0-7][0-9A-Va-v]{25}$`)

// See encoding.TextUnmarshaler
func (bs *Id) UnmarshalText(text []byte) error {
	if !validPattern.Match(text) {
		return errors.New("invalid string representation: " + string(text))
	}

	buffer, _ := strconv.ParseUint(string(text[0:2]), 32, 8)
	bs[0] = byte(buffer)

	// process three 40-bit (5-byte / 8-digit) groups
	for i := 0; i < 3; i++ {
		buffer, _ := strconv.ParseUint(string(text[2+i*8:10+i*8]), 32, 64)
		for j := 0; j < 5; j++ {
			bs[5+i*5-j] = byte(buffer)
			buffer >>= 8
		}
	}
	return nil
}
