# Web App

This directory contains the Airpath Next.js / TypeScript web application.

Next.js-specific configuration, dependencies, and environment examples live at the `frontend/` root. Application route source code lives under `frontend/src/app`.

## Structure

- `package.json`: Package metadata and scripts for the frontend app.
- `src/app`: Next.js App Router routes, layouts, and route-local styles.
- `next.config.ts`: Next.js configuration for the frontend app.
- `.env.example`: Documented environment variables for local frontend development.
- `tsconfig.json`: TypeScript settings for the frontend app.

`NEXT_PUBLIC_API_BASE_URL` is an optional API origin/base URL. Leave it empty
for same-origin `/v1` routes, or set an origin such as `https://api.example.com`.
The shared API route builders already include the `/v1` prefix.

The frontend imports workspace packages through their package exports. Its
verification and build commands run `build:deps` first, which generates the
`@airpath/shared-types` and `@airpath/map-rendering` `dist` artifacts needed by
TypeScript on clean CI workers.

## Commands

Run these commands from `frontend`:

```sh
pnpm dev
pnpm build
pnpm start
pnpm typecheck
```

Run these commands from the repository root:

```sh
pnpm --filter @airpath/web dev
pnpm --filter @airpath/web build
```

The frontend app root is `frontend`. App Router files belong under `frontend/src/app`.
