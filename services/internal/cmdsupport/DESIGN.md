# Command Support Design

The `cmdsupport` package owns tiny process-level helpers shared by Lambda command packages. It exists so command tests can replace or assert runtime-facing behavior without importing deeper adapter packages directly.

- `BackgroundContext` returns the process background context for command setup paths.
- `NowUTC` provides one UTC clock helper for command entrypoints that need invocation timestamps or worker ownership.
- `RuntimeBackend` delegates runtime backend selection to `runtimewiring.BackendFromEnv` while keeping command packages from depending on the full wiring surface for that decision.
- The package must stay small and must not own business rules, storage behavior, FlightAware transport logic, or environment parsing beyond delegation.
