# Runtime Wiring Design

The `runtimewiring` package owns runtime adapter selection for Lambda entrypoints.

Deployed Lambda execution uses AWS SDK-backed DynamoDB, S3, SQS, and Secrets Manager clients. Local execution and tests can use deterministic in-memory clients by setting `AIRPATH_RUNTIME_BACKEND=memory`; command packages also default to memory when `AWS_LAMBDA_FUNCTION_NAME` is absent.
