# AWS Integration Design

The `awsintegration` package contains adapter code that translates application ports to AWS-shaped clients. Tests use in-memory clients with the same adapter contracts so CI can verify serialization, keys, dedupe, and secret handling without provisioning AWS resources.

- DynamoDB repository methods serialize cached flights, lookup rows, latest positions, and monthly usage budget state by environment and month.
- DynamoDB repository implementation is split by responsibility across flight lookup/cache, position history, and usage budget files while keeping one adapter facade for application ports.
- FlightAware API keys can be resolved from `FLIGHTAWARE_API_KEY` or `FLIGHTAWARE_API_KEY_SECRET_ARN`, with the raw key passed only to the FlightAware transport.
- DynamoDB can store normalized FlightAware `/position` responses as `FlightPositions`.
- The usage budget repository normalizes the default USD 4.00 soft stop threshold and can record local FlightAware call estimates before upstream fetches.
- Usage reconciliation compares local estimates with FlightAware account usage and keeps the higher month-to-date cost so delayed remote usage cannot reduce the local guard.
- S3 GeoJSON repository stores and loads route and track map layers under `routes/` and `tracks/` keys, including normalized FlightAware `/route` and `/track` responses.
- SQS fetch task queue performs local idempotency checks before sending task messages.
- F15 worker adapters list due polling flights, maintain flight-level fetch leases in the cached flight record, append position history, and send redacted fetch failure diagnostics to the configured diagnostic/DLQ queue.
- Secrets adapter returns raw secret values to callers while logging only secret references.
- FlightAware fetch adapters translate FlightAware transport DTOs and errors into application-owned DTOs and boundary errors before invoking fetch processor ports.
- AWS SDK v2 client adapters implement the same small DynamoDB, S3, SQS, and Secrets Manager client interfaces used by in-memory tests. DynamoDB prefix reads use scan-based filtering because the current personal-demo tables are optimized for simple key-value access rather than secondary-index prefix queries.

Lambda runtime wiring selects these AWS adapters in deployed environments and keeps memory adapters available for local tests.
