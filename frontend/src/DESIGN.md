# Frontend Source Design

The frontend uses the Next.js `src/` directory convention for application route source.

- `src/app` owns App Router route segments, layouts, pages, and route-local styles.
- Shared UI components should be introduced under `src/components` when they have more than one caller.
- Feature-specific code should be introduced under `src/features/<feature-name>` when a route needs non-trivial state, adapters, or UI composition.
- Cross-cutting browser utilities should be introduced under `src/lib` only when they are independent of a single route or feature.

Keep Next.js configuration, package metadata, and environment examples at the `frontend/` root. Keep application route implementation inside `frontend/src/app`.
