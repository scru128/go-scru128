package scru128

import (
	"bytes"
	"database/sql"
	"encoding"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"
)

const maxUint48 uint64 = (1 << 48) - 1
const maxUint24 uint32 = (1 << 24) - 1
const maxUint32 uint32 = 0xFFFF_FFFF

// Encodes and decodes prepared cases correctly
func TestEncodeDecode(t *testing.T) {
	cases := []struct {
		timestamp uint64
		counterHi uint32
		counterLo uint32
		entropy   uint32
		string    string
	}{
		{0, 0, 0, 0, "0000000000000000000000000"},
		{maxUint48, 0, 0, 0, "F5LXX1ZZ5K6TP71GEEH2DB7K0"},
		{maxUint48, 0, 0, 0, "f5lxx1zz5k6tp71geeh2db7k0"},
		{0, maxUint24, 0, 0, "0000000005GV2R2KJWR7N8XS0"},
		{0, maxUint24, 0, 0, "0000000005gv2r2kjwr7n8xs0"},
		{0, 0, maxUint24, 0, "00000000000000JPIA7QL4HS0"},
		{0, 0, maxUint24, 0, "00000000000000jpia7ql4hs0"},
		{0, 0, 0, maxUint32, "0000000000000000001Z141Z3"},
		{0, 0, 0, maxUint32, "0000000000000000001z141z3"},
		{maxUint48, maxUint24, maxUint24, maxUint32, "F5LXX1ZZ5PNORYNQGLHZMSP33"},
		{maxUint48, maxUint24, maxUint24, maxUint32, "f5lxx1zz5pnorynqglhzmsp33"},
	}

	for _, e := range cases {
		var fromFields, fromString Id
		fromFields = FromFields(
			e.timestamp, e.counterHi, e.counterLo, e.entropy,
		)
		fromString, _ = Parse(e.string)

		caseStringAsBigInt, _ := new(big.Int).SetString(e.string, 36)
		if new(big.Int).SetBytes(fromFields[:]).Cmp(caseStringAsBigInt) != 0 ||
			fromFields.Timestamp() != e.timestamp ||
			fromFields.CounterHi() != e.counterHi ||
			fromFields.CounterLo() != e.counterLo ||
			fromFields.Entropy() != e.entropy ||
			fromFields.String() != strings.ToLower(e.string) {
			t.Fail()
		}
		if new(big.Int).SetBytes(fromString[:]).Cmp(caseStringAsBigInt) != 0 ||
			fromString.Timestamp() != e.timestamp ||
			fromString.CounterHi() != e.counterHi ||
			fromString.CounterLo() != e.counterLo ||
			fromString.Entropy() != e.entropy ||
			fromString.String() != strings.ToLower(e.string) {
			t.Fail()
		}
	}
}

// Returns error if an invalid string representation is supplied
func TestStringValidation(t *testing.T) {
	cases := []string{
		"",
		" 036z8puq4tsxsigk6o19y164q",
		"036z8puq54qny1vq3hcbrkweb ",
		" 036z8puq54qny1vq3helivwax ",
		"+036z8puq54qny1vq3hfcv3ss0",
		"-036z8puq54qny1vq3hhy8u1ch",
		"+36z8puq54qny1vq3hjq48d9p",
		"-36z8puq5a7j0ti08oz6zdrdy",
		"036z8puq5a7j0t_08p2cdz28v",
		"036z8pu-5a7j0ti08p3ol8ool",
		"036z8puq5a7j0ti08p4j 6cya",
		"f5lxx1zz5pnorynqglhzmsp34",
		"zzzzzzzzzzzzzzzzzzzzzzzzz",
		"039o\tvvklfmqlqe7fzllz7c7t",
		"039onvvklfmqlq漢字fgvd1",
		"039onvvkl🤣qe7fzr2hdoqu",
		"頭onvvklfmqlqe7fzrhtgcfz",
		"039onvvklfmqlqe7fztft5尾",
		"039漢字a52xp4bvf4sn94e09cja",
		"039ooa52xp4bv😘sn97642mwl",
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
		FromFields(maxUint48, 0, 0, 0),
		FromFields(0, maxUint24, 0, 0),
		FromFields(0, 0, maxUint24, 0),
		FromFields(0, 0, 0, maxUint32),
		FromFields(maxUint48, maxUint24, maxUint24, maxUint32),
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
			e.Timestamp(), e.CounterHi(), e.CounterLo(), e.Entropy(),
		) != e {
			t.Fail()
		}

		marshaledBinary, _ := e.MarshalBinary()
		marshaledText, _ := e.MarshalText()
		unmarshaled := new(Id)
		if unmarshaled.UnmarshalBinary(marshaledBinary) != nil || *unmarshaled != e {
			t.Fail()
		}
		if unmarshaled.UnmarshalBinary(marshaledText) != nil || *unmarshaled != e {
			t.Fail()
		}
		if unmarshaled.UnmarshalText(marshaledText) != nil || *unmarshaled != e {
			t.Fail()
		}

		scanned := new(Id)
		if scanned.Scan(e.String()) != nil || *scanned != e {
			t.Fail()
		}
		if scanned.Scan(marshaledBinary) != nil || *scanned != e {
			t.Fail()
		}
		if scanned.Scan(marshaledText) != nil || *scanned != e {
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
		FromFields(0, maxUint24, 0, 0),
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
		if !bytes.Equal(marshaled, strJson) || *unmarshaled != obj {
			t.Fail()
		}
	}
}

// Ensures compliance with interfaces.
func TestInterfaces(t *testing.T) {
	var x Id
	var _ fmt.Stringer = x
	var _ encoding.TextMarshaler = x
	var _ encoding.TextUnmarshaler = &x
	var _ encoding.BinaryMarshaler = x
	var _ encoding.BinaryUnmarshaler = &x
	var _ sql.Scanner = &x
}
