# Shared Types Source Design

This source directory owns TypeScript domain contracts that are shared by the web app, API contract work, and fixtures.

## L2-01 Domain Model Baseline

The initial domain model exports basic, dependency-free TypeScript types for flights, airports, positions, events, persistent user watches, and WebSocket session subscriptions. These types intentionally keep nullable FlightAware-derived fields as `null` instead of defaulting to empty strings or zero values.

Event dedupe key generation is left to a later L2 work item.

## L2-02 Flight Leg ID Generation

The TypeScript package exports deterministic pure functions for generating `provisionalFlightLegId` and `internalFlightLegId`. The generated IDs use the required `sched_` and `iflg_` prefixes, then append the first 12 hex characters of a shared FNV-1a 64-bit hash over the ordered specification fields.

The hash input order follows the specification:

- `provisionalFlightLegId`: `ident`, `originCode`, `destinationCode`, `scheduledOut`
- `internalFlightLegId`: `faFlightId`, `originCode`, `destinationCode`, `scheduledOut`, `legIndex`

The same algorithm is mirrored in Go under `services/internal/domain` so API and shared TypeScript contract tests can pin the same values.

## L2-03 Nullable Display Conversion

Nullable display helpers convert missing FlightAware-derived values into explicit labels while preserving meaningful present values such as progress `0`. The helpers cover generic nullable values plus focused display conversion for time strings, progress percentages, airport labels, aircraft type, and registration text.

Supported missing reasons map to the user-facing labels from the specification:

- `not_acquired`: `未取得`
- `not_announced`: `未発表`
- `not_applicable`: `対象外`
- `unavailable`: `取得不可`

The helpers never coerce nullish values to `0` or an empty string. Go mirrors the TypeScript helper behavior in `services/internal/domain`.

## L2-04 Time Normalization

Time normalization helpers convert FlightAware timestamps and UI local date-time input into UTC ISO 8601 strings without milliseconds. FlightAware values must include `Z` or a numeric offset, and UI input must provide both a local ISO date-time string and an IANA timezone.

The helpers reject invalid timestamps and invalid timezone names instead of guessing. Go mirrors the TypeScript behavior in `services/internal/domain`.

## L2-05 Position Metric Conversion

Position metric helpers normalize FlightAware-derived altitude, speed, and heading fields without inventing values when the source is missing. `altitudeFeet` is calculated only when `altitudeHundredsFeet` is present, using `altitudeHundredsFeet * 100`.

Groundspeed remains in knots. Heading remains in degrees, with `360` normalized to `0` for display rotation because the specification treats those values as equivalent. Go mirrors the TypeScript helper behavior in `services/internal/domain`.

## L2-06 Flight Duration Calculation

Flight duration helpers calculate the best available duration using the specification priority order: actual runway time, estimated runway time, scheduled runway time, filed ETE, then actual gate-to-gate time.

The result keeps the duration kind and, when the duration came from timestamps, the normalized UTC start and end timestamps. Incomplete timestamp pairs are skipped rather than partially calculated. Go mirrors the TypeScript helper behavior in `services/internal/domain`.
