# Runtime Wiring Design

The `runtimewiring` package owns runtime adapter selection for Lambda entrypoints.

Deployed Lambda execution uses AWS SDK-backed DynamoDB, S3, SQS, and Secrets Manager clients. Local execution and tests can use deterministic in-memory clients by setting `AIRPATH_RUNTIME_BACKEND=memory`; command packages also default to memory when `AWS_LAMBDA_FUNCTION_NAME` is absent.

Shared runtime factories resolve backend mode, table names, queue URLs, GeoJSON bucket names, usage budget scope, HTTP adapter dependencies, polling dispatcher dependencies, and fetch processor dependencies in one package. Lambda command packages keep only handler-specific request handling and delegate adapter construction here.
