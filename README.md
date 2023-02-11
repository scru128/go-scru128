# SCRU128: Sortable, Clock and Random number-based Unique identifier

[![GitHub tag](https://img.shields.io/github/v/tag/scru128/go-scru128)](https://github.com/scru128/go-scru128)
[![License](https://img.shields.io/github/license/scru128/go-scru128)](https://github.com/scru128/go-scru128/blob/main/LICENSE)

SCRU128 ID is yet another attempt to supersede [UUID] for the users who need
decentralized, globally unique time-ordered identifiers. SCRU128 is inspired by
[ULID] and [KSUID] and has the following features:

- 128-bit unsigned integer type
- Sortable by generation time (as integer and as text)
- 25-digit case-insensitive textual representation (Base36)
- 48-bit millisecond Unix timestamp that ensures useful life until year 10889
- Up to 281 trillion time-ordered but unpredictable unique IDs per millisecond
- 80-bit three-layer randomness for global uniqueness

```go
import "github.com/scru128/go-scru128/v2"

// generate a new identifier object
x := scru128.New()
fmt.Println(x)    // e.g. "036Z951MHJIKZIK2GSL81GR7L"
fmt.Println(x[:]) // as a 128-bit unsigned integer in big-endian byte array

// generate a textual representation directly
fmt.Println(scru128.NewString()) // e.g. "036Z951MHZX67T63MQ9XE6Q0J"
```

See [SCRU128 Specification] for details.

[uuid]: https://en.wikipedia.org/wiki/Universally_unique_identifier
[ulid]: https://github.com/ulid/spec
[ksuid]: https://github.com/segmentio/ksuid
[scru128 specification]: https://github.com/scru128/spec

## License

Licensed under the Apache License, Version 2.0.

## See also

- [scru128 package - github.com/scru128/go-scru128/v2 - Go Packages](https://pkg.go.dev/github.com/scru128/go-scru128/v2)
