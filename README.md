<div>
  <h1>Airpath</h1>
  <p><strong>FlightAware AeroAPI route display system with a Next.js map UI, Go backend services, and AWS Terraform infrastructure.</strong></p>

</div>

## Overview

Airpath displays FlightAware-backed flight routes, tracks, airport boards, and usage limits. The repository combines a typed Next.js frontend, Clean Architecture Go services, shared TypeScript API contracts, and Terraform-managed AWS deployment roots.

The repository includes:

- Next.js App Router frontend with Mapbox route visualization
- Go HTTP/Lambda services for cache-first FlightAware API workflows
- Shared TypeScript API schemas, route builders, fixtures, and validation
- DynamoDB, S3, SQS, Secrets Manager, Lambda, and HTTP API adapters
- Terraform environments and CI policy checks
- Unit, contract, architecture, and Terraform tests

## Quick Start

### Prerequisites

- Node.js `>= 22.15.0`
- pnpm `10.10.0`
- Go `1.24.2`
- Terraform for infrastructure checks
- `gitleaks` for the full local CI gate

### Setup

```bash
corepack enable
pnpm install
```

### Run Locally

Start the frontend:

```bash
pnpm dev
```

Do not run `npx next dev` from the repository root; use the root `pnpm dev` script so the frontend workspace receives the right package context.

For local browser testing against the Go API, start the API server in a second terminal:

```bash
pnpm dev:api
```

Then configure the frontend API origin:

```env
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
```

Keep backend credentials such as `FLIGHTAWARE_API_KEY` in the root `.env` file. Do not put backend secrets in `NEXT_PUBLIC_*` variables.

## Test And Build

```bash
pnpm test
pnpm build
make ci
```

Useful focused checks:

```bash
make go-test
make terraform-check
make gitleaks
```

## Architecture

```text
.
├─ frontend/        # Next.js web app, Mapbox UI, and frontend tests
├─ packages/
│  └─ shared-types/ # Shared API contracts, schemas, fixtures, and helpers
├─ services/        # Go application, domain, adapter, and Lambda/local entrypoints
├─ infra/
│  ├─ ci/           # CI helper tests
│  └─ terraform/    # AWS bootstrap, environments, modules, and policy checks
├─ scripts/         # Build and CI support scripts
└─ Makefile         # Local development and CI command entrypoint
```

More detailed responsibility notes live in:

- [frontend/README.md](frontend/README.md)
- [services/README.md](services/README.md)
- [infra/terraform/README.md](infra/terraform/README.md)
- [packages/shared-types/src/DESIGN.md](packages/shared-types/src/DESIGN.md)

## Runtime Configuration

- `NEXT_PUBLIC_API_BASE_URL`: frontend API origin, usually `http://localhost:8080` for local API testing
- `NEXT_PUBLIC_MAPBOX_ACCESS_TOKEN`: publishable Mapbox browser token
- `FLIGHTAWARE_API_KEY`: backend-only FlightAware key for local or test use
- `FLIGHTAWARE_FETCH_ENABLED=true` and `FLIGHTAWARE_REAL_CALLS_ENABLED=true`: opt in to real FlightAware calls
- `AIRPATH_RUNTIME_BACKEND=memory`: force in-memory backend adapters outside AWS

API keys must come from environment variables or AWS Secrets Manager. They must not be hardcoded in source, Terraform variables, committed examples, or frontend public variables.
