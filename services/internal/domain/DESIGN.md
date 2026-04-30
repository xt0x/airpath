# Domain Layer Design

The `domain` package owns dependency-free backend model types and deterministic domain helpers. It has no AWS, HTTP, FlightAware transport, runtime, or presentation dependencies.

- Flight, airport, time, position, event, polling, and identifier types define the shared backend domain shape used by application and adapter packages.
- `Flight` contains public flight facts plus internal operational state: internal/provisional/FlightAware IDs, diversion metadata, artifact keys, latest-position metadata, poll state, next poll timestamps, idle time, leases, update time, and TTL.
- `FlightPosition` stores normalized coordinate, altitude, speed, heading, timestamp, update type, source, identity, and TTL data for position-history persistence.
- `FlightEvent` stores event identity, dedupe key, event type, timestamp, source, payload, application status, and creation time for future polling-derived events.
- ID generation, UTC time normalization, local-time conversion, flight duration, position metric normalization, and event dedupe helpers are pure functions with unit coverage.
- Provisional flight IDs are derived from public schedule facts; internal FlightAware-backed leg IDs are derived from stable upstream flight facts so they do not depend on response order.
- Position metric normalization converts FlightAware altitude hundreds to feet, preserves missing speed and heading values, and normalizes heading 360 degrees to 0.
- Local-time conversion accepts local ISO date-times with or without seconds and normalizes them to UTC seconds.
- Flight duration prefers actual runway times, then other complete positive time pairs by priority, then positive filed ETE, then actual gate time when no earlier positive source exists.
- Missing-value reasons are stable domain codes. Localized labels and display strings live in the `display` package.
- Golden fixture tests compare backend helper output against shared fixture expectations so TypeScript and Go domain behavior drift visibly.
