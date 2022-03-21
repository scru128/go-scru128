package scru128

import (
	"bytes"
	"errors"
)

// Represents a SCRU128 ID and provides converters and comparison operators.
type Id [16]byte

// Creates a SCRU128 ID object from field values.
//
// This function panics if any argument is out of the value range of the field.
func FromFields(
	timestamp uint64,
	counterHi uint32,
	counterLo uint32,
	entropy uint32,
) Id {
	if timestamp > 0xffff_ffff_ffff ||
		counterHi > maxCounterHi ||
		counterLo > maxCounterLo {
		panic("invalid field value")
	}

	return Id{
		byte(timestamp >> 40),
		byte(timestamp >> 32),
		byte(timestamp >> 24),
		byte(timestamp >> 16),
		byte(timestamp >> 8),
		byte(timestamp),
		byte(counterHi >> 16),
		byte(counterHi >> 8),
		byte(counterHi),
		byte(counterLo >> 16),
		byte(counterLo >> 8),
		byte(counterLo),
		byte(entropy >> 24),
		byte(entropy >> 16),
		byte(entropy >> 8),
		byte(entropy),
	}
}

// Creates a SCRU128 ID object from a 25-digit string representation.
func Parse(strValue string) (id Id, err error) {
	err = id.UnmarshalText([]byte(strValue))
	return
}

// Returns the 48-bit timestamp field value.
func (bs Id) Timestamp() uint64 {
	return bytesToUint64(bs[0:6])
}

// Returns the 24-bit counter_hi field value.
func (bs Id) CounterHi() uint32 {
	return uint32(bytesToUint64(bs[6:9]))
}

// Returns the 24-bit counter_lo field value.
func (bs Id) CounterLo() uint32 {
	return uint32(bytesToUint64(bs[9:12]))
}

// Returns the 32-bit entropy field value.
func (bs Id) Entropy() uint32 {
	return uint32(bytesToUint64(bs[12:16]))
}

// Returns the 25-digit canonical string representation.
func (bs Id) String() string {
	buffer, _ := bs.MarshalText()
	return string(buffer)
}

// Returns -1, 0, or 1 if the object is less than, equal to, or greater than the
// argument, respectively.
func (bs Id) Cmp(other Id) int {
	return bytes.Compare(bs[:], other[:])
}

// Translates a big-endian byte sequence into uint64.
func bytesToUint64(bigEndian []byte) uint64 {
	var buffer uint64
	for _, v := range bigEndian {
		buffer = (buffer << 8) | uint64(v)
	}
	return buffer
}

// Translates a Base36 digit value array into uint64.
func base36ToUint64(digits []byte) uint64 {
	var buffer uint64
	for _, v := range digits {
		buffer = (buffer * 36) + uint64(v)
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

// Digit characters used in the Base36 notation.
var digits = []byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")

// See encoding.TextMarshaler
func (bs Id) MarshalText() (text []byte, err error) {
	text = make([]byte, 25)
	minIndex := 99 // any number greater than size of output array
	for i := -2; i < 16; i += 6 {
		// implement Base36 using 48-bit words
		var carry uint64
		if i == -2 {
			carry = bytesToUint64(bs[0 : i+6])
		} else {
			carry = bytesToUint64(bs[i : i+6])
		}

		// iterate over output array from right to left while carry != 0 but at
		// least up to place already filled
		j := len(text) - 1
		for ; carry > 0 || j > minIndex; j-- {
			carry += uint64(text[j]) << 48
			text[j] = byte(carry % 36)
			carry = carry / 36
		}
		minIndex = j
	}

	for i, v := range text {
		text[i] = digits[v]
	}
	return
}

// O(1) map from ASCII code points to Base36 digit values.
var decodeMap = [256]byte{
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x00, 0x01, 0x02, 0x03,
	0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16,
	0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20, 0x21, 0x22, 0x23,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
	0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d,
	0x1e, 0x1f, 0x20, 0x21, 0x22, 0x23, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
}

// See encoding.TextUnmarshaler
func (bs *Id) UnmarshalText(text []byte) error {
	if len(text) != 25 {
		return errors.New("invalid length")
	}

	src := make([]byte, 25)
	for i, v := range text {
		src[i] = decodeMap[v]
		if src[i] == 0xff {
			return errors.New("invalid digit")
		}
	}

	for i := range bs {
		bs[i] = 0
	}

	minIndex := 99 // any number greater than size of output array
	for i := -2; i < 25; i += 9 {
		// implement Base36 using 9-digit words
		var carry uint64
		if i == -2 {
			carry = base36ToUint64(src[0 : i+9])
		} else {
			carry = base36ToUint64(src[i : i+9])
		}

		// iterate over output array from right to left while carry != 0 but at
		// least up to place already filled
		j := len(bs) - 1
		for ; carry > 0 || j > minIndex; j-- {
			if j < 0 {
				return errors.New("out of 128-bit value range")
			}
			carry += uint64(bs[j]) * 101559956668416 // 36^9
			bs[j] = byte(carry & 0xff)
			carry = carry >> 8
		}
		minIndex = j
	}
	return nil
}
