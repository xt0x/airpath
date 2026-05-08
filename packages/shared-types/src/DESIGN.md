# Shared Types Source Design

This source directory owns TypeScript domain contracts that are shared by the web app, API contract work, and fixtures. Domain model definitions live in `domain-types.ts`; `index.ts` is the public barrel that re-exports domain, helper, and API contract surfaces.

## L2-01 Domain Model Baseline

The initial domain model exports basic, dependency-free TypeScript types for flights, airports, positions, and polling-derived events. The free-allowance MVP intentionally excludes FlightAware Alerts, persistent watches, and WebSocket session subscriptions from the shared domain surface. These types intentionally keep nullable FlightAware-derived fields as `null` instead of defaulting to empty strings or zero values.

All L2 pure helper categories are represented in this package.

## L2-02 Flight Leg ID Generation

The TypeScript package exports deterministic pure functions for generating `provisionalFlightLegId` and `internalFlightLegId`. The generated IDs use the required `sched_` and `iflg_` prefixes, then append the first 12 hex characters of a shared FNV-1a 64-bit hash over the ordered specification fields.

The hash input order follows the specification:

- `provisionalFlightLegId`: `ident`, `originCode`, `destinationCode`, `scheduledOut`
- `internalFlightLegId`: `faFlightId`, `originCode`, `destinationCode`, `scheduledOut`, `legIndex`

The same algorithm is mirrored in Go under `services/internal/domain` so API and shared TypeScript contract tests can pin the same values.

## L2-03 Nullable Display Conversion

Nullable display helpers convert missing FlightAware-derived values into explicit labels while preserving meaningful present values such as progress `0`. The helpers cover generic nullable values plus focused display conversion for time strings, progress percentages, airport labels, aircraft type, and registration text.

Supported missing reasons map to English user-facing labels:

- `not_acquired`: `Not acquired`
- `not_announced`: `Not announced`
- `not_applicable`: `Not applicable`
- `unavailable`: `Unavailable`

The helpers never coerce nullish values to `0` or an empty string. Go mirrors the TypeScript helper behavior in `services/internal/domain`.

## L2-04 Time Normalization

Time normalization helpers convert FlightAware timestamps and UI local date-time input into UTC ISO 8601 strings without milliseconds. FlightAware values must include `Z` or a numeric offset, and UI input must provide both a local ISO date-time string, with or without seconds, and an IANA timezone.

The helpers reject invalid timestamps and invalid timezone names instead of guessing. Go mirrors the TypeScript behavior in `services/internal/domain`.

## L2-05 Position Metric Conversion

Position metric helpers normalize FlightAware-derived altitude, speed, and heading fields without inventing values when the source is missing. `altitudeFeet` is calculated only when `altitudeHundredsFeet` is present, using `altitudeHundredsFeet * 100`. API position DTOs also carry nullable altitude-change and update-type values when FlightAware supplies them.

Groundspeed remains in knots. Heading remains in degrees, with `360` normalized to `0` for display rotation because the specification treats those values as equivalent. Go mirrors the TypeScript helper behavior in `services/internal/domain`.

## L2-06 Flight Duration Calculation

Flight duration helpers calculate the best available duration using the specification priority order: actual runway time, estimated runway time, scheduled runway time, positive filed ETE, then actual gate-to-gate time.

The result keeps the duration kind and, when the duration came from timestamps, the normalized UTC start and end timestamps. Incomplete or non-positive timestamp pairs and non-positive filed ETE values are skipped rather than partially calculated. Go mirrors the TypeScript helper behavior in `services/internal/domain`.

## L2-07 Event Dedupe Key Generation

Event dedupe helpers generate stable hash keys for polling-derived events. The key uses a `polling` discriminator, `eventType`, `coalesce(faFlightId, provisionalFlightLegId)`, and `eventTimestamp`.

When `faFlightId` is unavailable for a provisional scheduled flight, the helper uses `provisionalFlightLegId`, matching the specification. Go mirrors the TypeScript helper behavior in `services/internal/domain`.

## API Contract Definition

The package exports an OpenAPI 3.1 MVP contract for the backend HTTP API. The contract defines only the free-allowance surfaces needed before UI and Go API implementation: airport departure/arrival boards, flight search, flight detail, map-data, positions, bounded refresh requests, and usage status.

The contract source is split by responsibility: `api-types.ts` owns TypeScript DTOs, `api-schemas.ts` owns reusable OpenAPI schema fragments, `api-paths.ts` owns route definitions, `api-validation.ts` owns dependency-free runtime validation against those schema fragments, and `api-contract.ts` assembles the public OpenAPI literal and re-exports the stable API surface.

Airport-board discovery is represented as bounded departure and arrival endpoints with an airport code path parameter and optional ISO date query. Flight search remains ident-only in the shared contract. Date-scoped ident search is intentionally not advertised until the backend search use case and cache/upstream lookup path accept an explicit date filter.

Map-data responses are returned as the full planned, actual, and current layer set. Position history exposes only `since` and `limit` query parameters. Layer filtering, map simplification, and position quality modes are intentionally not advertised until the backend application inputs implement those behaviors.

Every success response carries `CacheMetadata` so callers can distinguish fresh cache, stale cache, cache misses, derived data, local accounting, and FlightAware-backed data. Typed error responses cover validation failures, budget stop, rate limiting, stale-cache misses, upstream failures, and explicit FlightAware fetch disablement.

Usage status includes the monthly budget scope (`environment` and `month`) and `dailyUsage` entries with date, estimated USD cost, and estimated API call count so the frontend can render the selected month as a daily usage chart without deriving hidden contract fields.

`FetchTask` is defined as an SQS message schema with only these task types: `summary`, `position`, `route`, `track`, and `final_track`. Public refresh requests use the non-summary subset because summary fetches are seeded through search misses. WebSocket delivery and FlightAware Alerts remain explicitly outside the free-allowance MVP contract.

The package also exports TypeScript response interfaces and the `validateApiSchema` runtime helper for the API contract so frontend callers do not redefine response shapes independently from the shared contract source.

API response value objects that overlap with the shared domain model, such as airports and flight times, are derived from the domain exports instead of being redefined separately.

Shared FNV-1a hashing for deterministic IDs and dedupe keys lives in one package-private helper so hash behavior does not drift between TypeScript helper categories.

Contract fixtures under `packages/shared-types/fixtures` are owned by this package boundary. They are included in package metadata and covered by a fixture-focused TypeScript config so shared JSON-backed tests stay type checked without becoming production build output.

API route templates and concrete browser route builders live beside the OpenAPI path source. Frontend code imports those builders instead of hardcoding `/v1` endpoint strings, keeping the TypeScript contract and browser client aligned.

Unit tests for shared source live under `packages/shared-types/test` instead of `src`, with `tsconfig.test.json` keeping those tests type checked while `src` remains limited to production package source and this design note.
