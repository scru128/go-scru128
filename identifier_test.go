package scru128

import (
	"bytes"
	"encoding/json"
	"math/big"
	"strings"
	"testing"
)

const maxUint44 uint64 = (1 << 44) - 1
const maxUint28 uint32 = (1 << 28) - 1
const maxUint24 uint32 = (1 << 24) - 1
const maxUint32 uint32 = 0xFFFF_FFFF

// Encodes and decodes prepared cases correctly
func TestEncodeDecode(t *testing.T) {
	cases := []struct {
		timestamp    uint64
		counter      uint32
		perSecRandom uint32
		perGenRandom uint32
		string       string
	}{
		{0, 0, 0, 0, "00000000000000000000000000"},
		{maxUint44, 0, 0, 0, "7VVVVVVVVG0000000000000000"},
		{maxUint44, 0, 0, 0, "7vvvvvvvvg0000000000000000"},
		{0, maxUint28, 0, 0, "000000000FVVVVU00000000000"},
		{0, maxUint28, 0, 0, "000000000fvvvvu00000000000"},
		{0, 0, maxUint24, 0, "000000000000001VVVVS000000"},
		{0, 0, maxUint24, 0, "000000000000001vvvvs000000"},
		{0, 0, 0, maxUint32, "00000000000000000003VVVVVV"},
		{0, 0, 0, maxUint32, "00000000000000000003vvvvvv"},
		{maxUint44, maxUint28, maxUint24, maxUint32, "7VVVVVVVVVVVVVVVVVVVVVVVVV"},
		{maxUint44, maxUint28, maxUint24, maxUint32, "7vvvvvvvvvvvvvvvvvvvvvvvvv"},
	}

	for _, e := range cases {
		var fromFields, fromString Id
		fromFields = FromFields(
			e.timestamp, e.counter, e.perSecRandom, e.perGenRandom,
		)
		fromString, _ = Parse(e.string)

		caseStringAsBigInt, _ := new(big.Int).SetString(e.string, 32)
		if new(big.Int).SetBytes(fromFields[:]).Cmp(caseStringAsBigInt) != 0 ||
			fromFields.Timestamp() != e.timestamp ||
			fromFields.Counter() != e.counter ||
			fromFields.PerSecRandom() != e.perSecRandom ||
			fromFields.PerGenRandom() != e.perGenRandom ||
			fromFields.String() != strings.ToUpper(e.string) {
			t.Fail()
		}
		if new(big.Int).SetBytes(fromString[:]).Cmp(caseStringAsBigInt) != 0 ||
			fromString.Timestamp() != e.timestamp ||
			fromString.Counter() != e.counter ||
			fromString.PerSecRandom() != e.perSecRandom ||
			fromString.PerGenRandom() != e.perGenRandom ||
			fromString.String() != strings.ToUpper(e.string) {
			t.Fail()
		}
	}
}

// Returns error if an invalid string representation is supplied
func TestStringValidation(t *testing.T) {
	cases := []string{
		"",
		" 00SCT4FL89GQPRHN44C4LFM0OV",
		"00SCT4FL89GQPRJN44C7SQO381 ",
		" 00SCT4FL89GQPRLN44C4BGCIIO ",
		"+00SCT4FL89GQPRNN44C4F3QD24",
		"-00SCT4FL89GQPRPN44C7H4E5RC",
		"+0SCT4FL89GQPRRN44C55Q7RVC",
		"-0SCT4FL89GQPRTN44C6PN0A2R",
		"00SCT4FL89WQPRVN44C41RGVMM",
		"00SCT4FL89GQPS1N4_C54QDC5O",
		"00SCT4-L89GQPS3N44C602O0K8",
		"00SCT4FL89GQPS N44C7VHS5QJ",
		"80000000000000000000000000",
		"VVVVVVVVVVVVVVVVVVVVVVVVVV",
	}

	for _, e := range cases {
		var err error
		_, err = Parse(e)
		if err == nil {
			t.Fail()
		}
	}
}

// Has symmetric converters from/to various values
func TestSymmetricConverters(t *testing.T) {
	cases := []Id{
		FromFields(0, 0, 0, 0),
		FromFields(maxUint44, 0, 0, 0),
		FromFields(0, maxUint28, 0, 0),
		FromFields(0, 0, maxUint24, 0),
		FromFields(0, 0, 0, maxUint32),
		FromFields(maxUint44, maxUint28, maxUint24, maxUint32),
	}

	g := NewGenerator()
	for i := 0; i < 1_000; i++ {
		e, _ := g.Generate()
		cases = append(cases, e)
	}

	for _, e := range cases {
		if x, _ := Parse(e.String()); x != e {
			t.Fail()
		}
		if FromFields(
			e.Timestamp(), e.Counter(), e.PerSecRandom(), e.PerGenRandom(),
		) != e {
			t.Fail()
		}

		marshaledBinary, _ := e.MarshalBinary()
		unmarshaledBinary := new(Id)
		unmarshaledBinary.UnmarshalBinary(marshaledBinary)
		if *unmarshaledBinary != e {
			t.Fail()
		}

		marshaledText, _ := e.MarshalText()
		unmarshaledText := new(Id)
		unmarshaledText.UnmarshalText(marshaledText)
		if *unmarshaledText != e {
			t.Fail()
		}
	}
}

// Supports comparison methods
func TestComparisonMethods(t *testing.T) {
	ordered := []Id{
		FromFields(0, 0, 0, 0),
		FromFields(0, 0, 0, 1),
		FromFields(0, 0, 0, maxUint32),
		FromFields(0, 0, 1, 0),
		FromFields(0, 0, maxUint24, 0),
		FromFields(0, 1, 0, 0),
		FromFields(0, maxUint28, 0, 0),
		FromFields(1, 0, 0, 0),
		FromFields(2, 0, 0, 0),
	}

	g := NewGenerator()
	for i := 0; i < 1_000; i++ {
		e, _ := g.Generate()
		ordered = append(ordered, e)
	}

	prev := ordered[0]
	for _, curr := range ordered[1:] {
		if curr == prev || curr.Cmp(prev) < 0 || prev.Cmp(curr) > 0 {
			t.Fail()
		}

		clone := curr
		if curr != clone || curr.Cmp(clone) != 0 || clone.Cmp(curr) != 0 {
			t.Fail()
		}

		prev = curr
	}
}

// Serializes and deserializes an object using the canonical string
// representation
func TestSerializedForm(t *testing.T) {
	g := NewGenerator()
	for i := 0; i < 1_000; i++ {
		obj, _ := g.Generate()
		strJson := []byte(`"` + obj.String() + `"`)

		marshaled, _ := json.Marshal(obj)
		unmarshaled := new(Id)
		json.Unmarshal(strJson, unmarshaled)
		if bytes.Compare(marshaled, strJson) != 0 || *unmarshaled != obj {
			t.Fail()
		}
	}
}
