# Frontend Source Design

The frontend uses the Next.js `src/` directory convention for application route source.

- `src/app` owns App Router route segments, layouts, pages, and route-local styles.
- Shared UI components should be introduced under `src/components` when they have more than one caller.
- Feature-specific code should be introduced under `src/features/<feature-name>` when a route needs non-trivial state, adapters, or UI composition.
- Cross-cutting browser utilities should be introduced under `src/lib` only when they are independent of a single route or feature.

Keep Next.js configuration, package metadata, and environment examples at the `frontend/` root. Keep application route implementation inside `frontend/src/app`.

## Flight MVP UI

`src/features/flights` owns the FlightAware free-allowance MVP browser workflow. It contains a typed API client for the HTTP contract, server-renderable component tests for search/results, summary, refresh guard state, stale cache notices, and usage budget display, plus the client dashboard state container used by the app route.

The map surface consumes normalized route, track, and current-position layers from the API. Shared deck.gl layer construction lives in `@airpath/map-rendering`, while the component also renders a lightweight SVG fallback so server-rendered and tokenless local views remain inspectable.

The runtime dashboard starts from empty API state. Fixture data remains in feature-local test helpers and must not be used as production initial state.

The flight feature keeps the stateful dashboard container, presentational dashboard view, SVG map fallback, feature stylesheet, and formatting helpers in separate files so UI composition, map rendering, styles, and data formatting can evolve independently.

The dashboard container loads flight detail, map data, and usage status through one snapshot helper so search selection and manual refresh share the same API orchestration path.

Flight dashboard sample responses live under `frontend/test/fixtures` rather than `frontend/src` so production feature code cannot accidentally import test-only state. The presentational dashboard view exports only the route-facing view component; its panel components remain private implementation details exercised through the full view.

Flight feature styles use `flight-dashboard`-prefixed class selectors rather than generic element or utility names, keeping the globally imported Next.js stylesheet scoped to the feature surface.

The API client builds all `/v1` endpoint paths through shared route builders from `@airpath/shared-types`. `NEXT_PUBLIC_API_BASE_URL` is treated only as an optional origin/base URL, so frontend environment configuration cannot duplicate the API prefix. Successful JSON responses are validated at the client boundary before dashboard state consumes them; malformed upstream payloads become typed retryable API errors instead of unchecked TypeScript casts.

Workspace package imports resolve through package `exports` and built `dist` artifacts during frontend builds. The frontend `tsconfig` must not alias `@airpath/shared-types` or `@airpath/map-rendering` directly to package `src` files because those sources use NodeNext `.js` import specifiers that Turbopack does not resolve as TypeScript source paths.

## Personal Demo Notice

The dashboard topbar includes a short personal non-commercial low-frequency notice so the deployed dev demo does not present itself as a commercial or high-frequency tracking surface. The notice is static UI copy; runtime fetch control and real-call opt-in state still come from the backend usage and fetch-control APIs.
