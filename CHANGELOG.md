# Changelog

## v3.0.2 - 2023-09-17

### Added

- Early `nil` checks to `Generator` methods

### Maintenance

- Improved documentation about generator's clock rollback behavior
- Renamed source files: generator -> gen, identifier -> id

## v3.0.1 - 2023-07-17

Most notably, v3 switches the letter case of generated IDs from uppercase (e.g.,
"036Z951MHJIKZIK2GSL81GR7L") to lowercase (e.g., "036z951mhjikzik2gsl81gr7l"),
though it is technically not supposed to break existing code because SCRU128 is
a case-insensitive scheme. Other changes include the removal of deprecated APIs.

### Removed

- Deprecated items:
  - `Generator#GenerateCore()`
  - `Generator#LastStatus()` and `GeneratorStatus`

### Changed

- Letter case of generated IDs from uppercase to lowercase
- Edge case behavior of generator functions' rollback allowance handling

### Maintenance

- Upgraded minimum Go version to 1.20

## v2.3.2 - 2023-06-21

### Changed

- Error values returned by `Generator` and `Id` to improve error messages

## v2.3.1 - 2023-04-07

### Maintenance

- Tweaked docs and tests

## v2.3.0 - 2023-03-23

### Added

- `GenerateOrAbort()` and `GenerateOrAbortCore()` to `Generator` (formerly named
  as `GenerateNoRewind()` and `GenerateCoreNoRewind()`)
- `Generator#GenerateOrResetCore()`

### Deprecated

- `Generator#GenerateCore()`
- `Generator#LastStatus()` and `GeneratorStatus`

## v2.2.1 - 2023-03-19

### Added

- `GenerateNoRewind()` and `GenerateCoreNoRewind()` to `Generator` (experimental)

### Maintenance

- Improved documentation about generator method flavors

## v2.2.0 - 2023-02-12

### Changed

- `UnmarshalBinary()` behavior so it tries to parse byte slice also as textual
  representation, not only as 128-bit byte array
- `UnmarshalText()` and `UnmarshalBinary()` of `Id` now return error instead of
  panicking when called with nil receiver

### Added

- `sql.Scanner` interface implementation to `Id`

## v2.1.2 - 2022-06-11

### Fixed

- `GenerateCore()` to update `counter_hi` when `timestamp` passed < 1000

## v2.1.1 - 2022-05-23

### Fixed

- `GenerateCore()` to reject zero as `timestamp` value

## v2.1.0 - 2022-05-22

### Added

- `GenerateCore()` and `LastStatus()` to `Generator`

### Maintenance

- Updated README

## v2.0.0 - 2022-05-01

### Changed

- Textual representation: 26-digit Base32 -> 25-digit Base36
- Field structure: { `timestamp`: 44 bits, `counter`: 28 bits, `per_sec_random`:
  24 bits, `per_gen_random`: 32 bits } -> { `timestamp`: 48 bits, `counter_hi`:
  24 bits, `counter_lo`: 24 bits, `entropy`: 32 bits }
- Timestamp epoch: 2020-01-01 00:00:00.000 UTC -> 1970-01-01 00:00:00.000 UTC
- Counter overflow handling: stall generator -> increment timestamp
- Type of generator: Generator interface -> \*Generator struct

### Removed

- `Logger` as counter overflow is no longer likely to occur
- `TimestampBias`
- `Id#Counter()`, `Id#PerSecRandom()`, `Id#PerGenRandom()`

### Added

- `Id#CounterHi()`, `Id#CounterLo()`, `Id#Entropy()`

## v1.0.0 - 2022-01-03

- Initial stable release.
