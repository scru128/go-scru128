# Changelog

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
