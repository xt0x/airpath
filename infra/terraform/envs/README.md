# Terraform Environment Root Policy

`dev`, `stg`, and `prod` are separate Terraform roots so state, variables, and
deployment permissions stay environment-scoped.

Only `dev` currently creates AWS resources. `stg` and `prod` are placeholders
that validate partial S3 backend wiring, provider configuration, and environment naming only. Their native
Terraform tests assert the root-specific environment default and reject other
environment names; the Vitest HCL contract test asserts backend, provider,
variable, and output shape and confirms they do not declare deployable
infrastructure.

The `dev` root wires API, fetcher, and dispatcher Lambda environment variables
from module outputs and root-level runtime controls. Fetch workflow Lambdas that
enqueue or execute FlightAware work must receive both
`FLIGHTAWARE_FETCH_ENABLED` and `FLIGHTAWARE_REAL_CALLS_ENABLED`; runtime code
treats either flag being false as an external-call stop. Dispatcher also receives
the table and queue settings needed to query due poll state, reserve fetch task
idempotency, and enqueue executable fetch tasks.

When `stg` or `prod` begin creating resources, introduce a shared environment module
for common Airpath infrastructure first, then keep each root limited to
environment-specific backend configuration, provider configuration, variables,
and module inputs. Do not copy the full `dev` resource graph into another root.
