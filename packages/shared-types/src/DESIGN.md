# Shared Types Source Design

This source directory owns TypeScript domain contracts that are shared by the web app, API contract work, and fixtures.

## L2-01 Domain Model Baseline

The initial domain model exports basic, dependency-free TypeScript types for flights, airports, positions, events, persistent user watches, and WebSocket session subscriptions. These types intentionally keep nullable FlightAware-derived fields as `null` instead of defaulting to empty strings or zero values.

ID generation, nullable display helpers, time normalization, unit conversion, duration calculation, and event dedupe key generation are left to later L2 work items.

## L2-02 Flight Leg ID Generation

The TypeScript package exports deterministic pure functions for generating `provisionalFlightLegId` and `internalFlightLegId`. The generated IDs use the required `sched_` and `iflg_` prefixes, then append the first 12 hex characters of a shared FNV-1a 64-bit hash over the ordered specification fields.

The hash input order follows the specification:

- `provisionalFlightLegId`: `ident`, `originCode`, `destinationCode`, `scheduledOut`
- `internalFlightLegId`: `faFlightId`, `originCode`, `destinationCode`, `scheduledOut`, `legIndex`

The same algorithm is mirrored in Go under `services/internal/domain` so API and shared TypeScript contract tests can pin the same values.
