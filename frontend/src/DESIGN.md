# Frontend Source Design

The frontend uses the Next.js `src/` directory convention for application route source.

- `src/app` owns App Router route segments, layouts, pages, and route-local styles.
- Shared UI components should be introduced under `src/components` when they have more than one caller.
- Feature-specific code should be introduced under `src/features/<feature-name>` when a route needs non-trivial state, adapters, or UI composition.
- Cross-cutting browser utilities should be introduced under `src/lib` only when they are independent of a single route or feature.

Keep Next.js configuration, package metadata, and environment examples at the `frontend/` root. Keep application route implementation inside `frontend/src/app`.

## F14 Flight MVP UI

`src/features/flights` owns the FlightAware free-allowance MVP browser workflow. It contains a typed API client for the HTTP contract, server-renderable component tests for search/results, summary, refresh guard state, stale cache notices, and usage budget display, plus the client dashboard state container used by the app route.

The map surface consumes normalized route, track, and current-position layers from the API. Shared deck.gl layer construction lives in `@airpath/map-rendering`, while the component also renders a lightweight SVG fallback so server-rendered and tokenless local views remain inspectable.

The runtime dashboard starts from empty API state. Fixture data remains in feature-local test helpers and must not be used as production initial state.

## F17 Personal Demo Notice

The dashboard topbar includes a short personal non-commercial low-frequency notice so the deployed dev demo does not present itself as a commercial or high-frequency tracking surface. The notice is static UI copy; runtime fetch control and real-call opt-in state still come from the backend usage and fetch-control APIs.
