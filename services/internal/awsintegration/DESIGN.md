# AWS Integration Design

The `awsintegration` package contains adapter code that translates application ports to AWS-shaped clients. Tests use in-memory clients with the same adapter contracts so CI can verify serialization, keys, dedupe, and secret handling without provisioning AWS resources.

- DynamoDB repository methods serialize cached flights, lookup rows, latest positions, and usage budget state.
- S3 GeoJSON repository stores and loads route and track map layers under `routes/` and `tracks/` keys.
- SQS fetch task queue performs local idempotency checks before sending task messages.
- Secrets adapter returns raw secret values to callers while logging only secret references.

Real AWS SDK clients can be wrapped behind these small interfaces in the later deployment integration work.
