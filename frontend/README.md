# Web App

This directory contains the Airpath Next.js / TypeScript web application.

Next.js-specific configuration, dependencies, and environment examples live at the `frontend/` root. Application route source code lives under `frontend/src/app`, and feature implementation lives under `frontend/src/features`.

## Structure

- `package.json`: Package metadata and scripts for the frontend app.
- `src/app`: Next.js App Router routes, layouts, and global Mapbox GL/sidebar CSS import.
- `components.json`: shadcn registry configuration using the `radix-nova` style.
- `src/components/app-sidebar.tsx`: Application sidebar composition following `docs/slidebar.md`, including the static Airpath Ops identity, backend-aligned navigation groups, and rail.
- `src/components/ui`: shadcn CLI generated components from `components.json` using the `radix-nova` style. Keep this directory reserved for generated shadcn files; Airpath-specific behavior belongs in callers, feature components, or feature CSS.
- `src/components/ui/sidebar.tsx`: generated Sidebar primitive used by `AppSidebar`, `MapWorkspace`, and `UsageWorkspace`. The Live Map overlay behavior is scoped from the `map-sidebar-layout` provider class instead of a custom Sidebar prop.
- `src/components/ui/button.tsx`, `alert.tsx`, `skeleton.tsx`, and `progress.tsx`: generated public APIs are used directly. Callers render SVG icons as direct children, use repeated `Skeleton` instances for placeholder rows, and use the single `Progress` export.
- `src/components/ui/command.tsx`, `popover.tsx`, `hover-card.tsx`, `tooltip.tsx`, `calendar.tsx`, `card.tsx`, `chart.tsx`, and related primitives: generated shadcn components used by Live Map controls, current-aircraft hover summaries, collapsed sidebar labels, board date picking, and Usage & Limits visualization.
- `src/features/map-workspace`: Live Map workspace shell that composes the icon-collapsible application sidebar, airport-board discovery, secondary flight-number search, selected-flight controls, user-facing data status, refresh controls, SidebarInset, fullscreen map, the hidden/sidebar-aligned Flight Search state, and its feature-local `map-workspace.module.css`.
- `src/features/map-workspace/lib/airport-board-controls.ts`: Airport selector options, airport code display labels, and date-only board helpers.
- `src/features/map-workspace/lib/flight-search-selection.ts`: Pure first-result selection rule for loading a flight-number search result on the map.
- `src/features/mapbox`: Fullscreen Mapbox UI surface with native Mapbox route line layers, route endpoint labels, current-aircraft marker rendering, and feature-local `fullscreen-map.module.css` presentation for the marker, hover card, and missing-token notice.
- `src/features/mapbox/lib/aircraft-hover-card-formatters.ts`: Pure display helpers for current-aircraft hover-card metrics.
- `src/features/mapbox/lib/geojson-coordinates.ts`: Reusable Mapbox-valid GeoJSON coordinate extraction for route rendering and camera bounds.
- `src/features/mapbox/lib/map-layer-availability.ts`: Shared display eligibility rules for available map-layer GeoJSON and exact FlightAware planned routes.
- `src/features/mapbox/lib/map-camera.ts`: Route-aware Mapbox bounds calculation for aircraft and route focus requests, including antimeridian-safe longitude wrapping.
- `src/features/mapbox/lib/map-route-layers.ts`: Native Mapbox route source and layer setup, deduplicated source data synchronization, and route paint repair after style redraws.
- `src/features/mapbox/lib/route-line-feature.ts`: Reusable finite-coordinate route line normalization, current-position coordinate appending, and geometry eligibility shared by Mapbox line sources and route-focus camera bounds.
- `src/features/mapbox/lib/route-layer-source-data.ts`: Pure transformation from selected flight map data into planned route, actual track, and endpoint FeatureCollections for Mapbox sources.
- `src/features/flights/lib/public-flight-status.ts`: Normalizes public English-only upstream flight statuses before they reach search results, selected-flight summaries, or aircraft hover cards, including ASCII slash-separated status parts and an `Unknown` fallback for unsupported localized or blank display text.
- `src/features/mapbox/lib/map-theme.ts`: Reusable Mapbox paint theme tokens and builders for route lines and endpoint labels.
- `src/features/flights`: Non-UI flight feature code, currently limited to typed API adapters, public-status formatting, and re-exported public types. Frontend source imports flight API shapes through `src/features/flights/types`; the API client validates public contract enum values and numeric ranges before feature state consumes responses, while browser-initiated refresh requests use the shared `RefreshTaskType` subset, not the wider internal fetch queue task type.
- `src/features/usage`: Usage & Limits UI that reads `/v1/usage/status` through the typed API client, keeps budget progress in the API cost card, shows availability only when no stop is active, reports paused fetching as a stop reason, shows API errors in an alert, shows the monthly usage chart by month without repeating the budget total in the chart header, and owns its feature-local `usage-workspace.module.css`.
- `src/hooks/use-mobile.ts`: Shared 640px mobile breakpoint constant, pure width predicate, and `useSyncExternalStore`-based viewport hook used by the Sidebar.
- `src/lib/display-date-time.ts`: Shared human-facing UTC date-time formatter for backend ISO timestamps.
- `tests`: Frontend test code. Tests mirror the source area they verify and are not stored under `src`.
- `tests/architecture/import-alias-boundary.test.ts`: Architecture guard that keeps imports targeting `src` on the `@` alias instead of relative paths.
- `next.config.ts`: Next.js configuration for the frontend app.
- `.env.example`: Documented environment variables for local frontend development.
- `tsconfig.json`: TypeScript settings for the frontend app, including `@/*` as the alias for `src/*`.

Source and test files must import frontend source through the `@` alias. For
example, use `@/features/mapbox/config` or `@/components/ui/button` instead of
walking through `../` segments. Relative imports are reserved for files outside
`src`, such as package-boundary tests that intentionally read frontend root
metadata.

`NEXT_PUBLIC_API_BASE_URL` is an optional API origin/base URL. For local browser
testing, run `pnpm dev:api` from the repository root and set this to
`http://localhost:8080`. Leave it empty only when a same-origin `/v1` proxy is
available, or set an origin such as `https://api.example.com` for a deployed API.
The shared API route builders already include the `/v1` prefix. When running the
standalone Next.js dev server without a same-origin API proxy, an empty value
causes `/v1/usage/status` to resolve to the frontend server and return HTML
instead of JSON. If the configured API origin cannot be reached, the typed API
client reports a retryable network error that tells the operator to start
`pnpm dev:api` and restart the frontend after changing
`NEXT_PUBLIC_API_BASE_URL`. The API client strips trailing slashes from the
configured base path before appending shared `/v1` route-builder paths.
Successful API responses are checked at the frontend adapter boundary against
the public flight facade, including cache freshness/source values, flight ID
types, map sources, position source/update values, numeric ranges, and fetch
task metadata. Legacy usage-status responses that omit `budget.dailyUsage` are
accepted and normalized to an empty chart series after validation.
Browser refresh requests are also checked at runtime against the request body
schema before the POST is sent: task lists must be non-empty, unique, and within
`RefreshTaskType`, so internal-only `summary` work cannot leave the frontend
even if a caller bypasses TypeScript.

The Live Map uses the same API base URL for
`/v1/airports/{airportCode}/departures`,
`/v1/airports/{airportCode}/arrivals`, `/v1/flights/search`,
`/v1/flights/{flightId}`, `/v1/flights/{flightId}/map-data`,
`/v1/flights/{flightId}/positions`, and `/v1/flights/{flightId}/refresh`.
Airport boards are the primary discovery path; flight-number search remains as a
fallback. Airport board results stay idle until the user presses `Show on map` on
a candidate. A flight-number search automatically selects its first result and
loads refresh/detail/map-data for that flight so the route can appear without a
second click. When a flight-number search succeeds with no matches, the panel
shows the result empty state and clears the selected-flight summary.
Pending flight-number searches, airport board loads, selected-flight loads, and
manual refreshes are ignored after the user changes board criteria, clears
selection, starts another discovery request, explicitly opens a board candidate,
or switches to Tracked Flights, so stale responses cannot restore cleared flight
details, stale board candidates, or map focus later. Track hydration retries also
stop once their selected-flight load or manual refresh has been invalidated.
Flight-number search results come from the API in `scheduledOut` ascending order.
Position history is exposed in the client for future history toggles, but the
default map renders only the selected flight's planned route, actual track, and
current position.

`NEXT_PUBLIC_MAPBOX_ACCESS_TOKEN` is required for the browser map. Use a
publishable Mapbox token only; do not put secret Mapbox tokens or backend
credentials in `NEXT_PUBLIC_*` variables.

`NEXT_PUBLIC_MAPBOX_STYLE_URL` optionally overrides the Mapbox style. Leave it
empty to use `mapbox://styles/mapbox/streets-v12`.

The sidebar follows `docs/slidebar.md`: when collapsed it keeps the left icon
rail visible. On desktop the Live Map wraps `AppSidebar` in a `SidebarProvider`
with `map-sidebar-layout`, and global CSS scopes the generated sidebar gap to
zero for that provider. This keeps the fixed sidebar container over the map
without adding Airpath-specific props to the generated Sidebar primitive.
Opening or closing the menu does not resize the Mapbox surface. The map-side
trigger is translated by the open sidebar width so it remains reachable outside
the overlaid menu. When the sidebar is collapsed, the shared trigger alignment
variable only adjusts the y axis so the trigger center lines up with the
collapsed Airpath Ops identity icon center, while the x axis moves to the icon rail edge so
the trigger does not overlap the identity icon. The trigger keeps one composed
`transform` from separate x/y variables, so sliding the menu does not make the
button disappear while one axis changes. The desktop sidebar opens as a clipped
reveal: the container animates its width and hides overflow while the inner menu
keeps a stable `--sidebar-width` minimum width, so labels are revealed instead of
being squeezed during the opening motion. The sidebar width transition and the
map-side trigger use the same 200ms `cubic-bezier(0.2, 0.8, 0.2, 1)` easing.
Usage keeps its toolbar fixed through a scoped rule after the shared map toolbar
rule and omits `map-sidebar-layout`, so the generated Sidebar primitive keeps
its normal reserved gap. Its scoped CSS preserves the `SidebarProvider` flex
wrapper so opening the menu shifts the Usage content instead of covering the
`Usage & Limits` heading. The trigger toolbar passes pointer events through its
transparent area so collapsed sidebar icons, including the identity icon, remain
reachable; only the trigger button itself receives pointer events. `AppSidebar`
owns the application-specific collapsed rail classes: the Airpath Ops identity
uses a white `bg-sidebar-primary` 2rem square with a 4px radius, labels hide in
the icon rail, and menu buttons use explicit collapsed padding, overflow, and
transparent-background classes instead of global generated `data-slot`
overrides. The generated Sidebar primitive stays in
`src/components/ui/sidebar.tsx`, matching shadcn's direct `components/ui/<component>.tsx`
layout. FullscreenMap marker, hover-card, and missing-token notice styles live
in `src/features/mapbox/components/fullscreen-map.module.css`; Live Map
search-panel styles live in
`src/features/map-workspace/components/map-workspace.module.css`; and Usage
layout styles live in `src/features/usage/components/usage-workspace.module.css`.
`globals.css` must not contain `live-map-panel`, `usage-*`, or FullscreenMap
presentation selectors. Shared route loading styles stay in the App Router
route group.

The main sidebar navigation is grouped by backend-supported workflows:
`Workspace` contains `Live Map`, `Flight Search`, and `Tracked Flights`; `Data`
contains `Usage & Limits`; `System` contains `Settings`.
Fetch pipeline availability is covered by `Usage & Limits`, so the sidebar does
not include a separate `Fetch Status` item. These entries are direct top-level
sidebar menu buttons so the collapsed rail keeps one visible icon for each
workflow. `Flight Search` links
to the Live Map workspace action when rendered inside `MapWorkspace`; pressing
it does not navigate to another page and instead opens the existing search
panel in the current map view. Outside the Live Map workspace, the same menu
item links to `/?flightSearch=1`; `MapWorkspace` reads that query on mount and
opens the search panel after returning home. Action-backed and anchor-backed
menu items both rely on `SidebarMenuButton` for typography so `Flight Search`
does not render larger than the rest of the sidebar, and active state uses color
and background without increasing font weight. `Tracked Flights` follows the
same action pattern inside `MapWorkspace`: it closes the
search panel, marks the sidebar item active, and requests the Mapbox camera to
fly to the current tracked aircraft coordinates. Outside the Live Map
workspace, it links back to `/?trackedFlights=1` so the home route can perform
the same focus action after mounting.

The sidebar mobile breakpoint is centralized in `src/hooks/use-mobile.ts`.
`isMobileViewport` covers the exact 640px boundary in unit tests, and the
sidebar layout tests keep Tailwind's `md` breakpoint aligned with the same
exported constant. `useIsMobile` subscribes through `useSyncExternalStore`,
returns desktop during server rendering, and does not call React state setters
synchronously from an effect. `SidebarProvider` resets the temporary mobile
Sheet open state when the viewport returns to desktop, so resize cycles do not
preserve a stale mobile drawer.

The app uses shadcn's default dark color tokens at the document root. The root
layout applies the `dark` class and sets shadcn's dark `--background`
(`oklch(0.145 0 0)`) inline on `html` and `body`, so reloads do not show the
browser's default white background while CSS is still loading.
`--map-surface-background` resolves to `var(--background)`, and the html
element, body, Tailwind base body rule, sidebar wrapper, map inset, Mapbox
wrapper, Mapbox-generated map container, canvas container, and canvas all use
that backing color. The base body rule does not reapply the generic white
`bg-background`, so reloads do not briefly paint a white app background before
Mapbox redraws. The Mapbox component still observes its container for real
viewport changes, but sidebar open and close no longer change the map container
width because the menu is overlaid on top of the map. The Mapbox component
delegates native route source and layer orchestration to `map-route-layers.ts`,
pure route FeatureCollection selection to `route-layer-source-data.ts`, camera
bounds calculation to `map-camera.ts`, display eligibility to
`map-layer-availability.ts`, finite-coordinate route line normalization and
geometry eligibility to `route-line-feature.ts`, and reusable Mapbox-valid GeoJSON coordinate extraction to
`geojson-coordinates.ts`. Planned route, actual track, and
endpoint label GeoJSON attach directly to native Mapbox sources and layers so
route geometry is pinned to the same renderer as the base map during drag, zoom,
pitch, and style redraws. Exact FlightAware planned route geometry is
rendered as a dashed scheduled-route line, while actual track geometry remains
solid; both route lines use `map-theme.ts` paint builders with the same
`#facc15` yellow as the aircraft marker and `line-emissive-strength`, so Mapbox
Standard lighting cannot darken them in dark presets. The actual track display
discards non-finite or out-of-range track coordinates before appending the available current
aircraft position as the final coordinate, and appended current-position coordinates must
also pass Mapbox longitude/latitude validation. Route-focus bounds unwrap longitudes across
the antimeridian so transpacific route focus uses the shorter visible span without mutating cached
backend data.
Airport-to-airport fallback geometry, unavailable layer GeoJSON, and non-line
route features are not rendered as planned or actual route lines and are not
used for route-focus camera bounds. Route line source data is normalized to
finite longitude/latitude pairs within Mapbox's accepted ranges before it reaches Mapbox. Native Mapbox symbol
layers render origin and destination airport code labels only when exact planned
route geometry, at least two valid planned-route coordinates, and selected
flight detail are available; no start/end endpoint circle layer is rendered.
When the selected flight has a current heading,
the lucide Plane aircraft marker rotates by that source heading; missing heading
values render without inventing a direction. The aircraft marker renders as a
filled `--airpath-map-aircraft-marker` CSS-token plane and without a circular marker
background, and marker React root unmounts are deferred outside React cleanup so
route changes do not trigger nested-root unmount warnings. The mobile sidebar
sheet also forces the sidebar background so its slide-in panel cannot briefly
paint the generic popover background over the map.

The Live Map panel is scoped to one selected flight. It uses shadcn `Command`
inside `Popover` for airport selection, shadcn `Calendar` inside `Popover` for
the board date, shadcn `Button` for direction, Load, Search, and Refresh
actions, shadcn `Input` for the flight-number fallback, shadcn `Badge` for
compact flight-data status labels, and shadcn `Spinner` for pending
`Show on map` feedback. The normal Live Map state does not render the
search panel. Pressing `Flight Search` in the sidebar switches the same
workspace to the search state, marks that sidebar action active while the panel
is visible, clears that active state after close, and horizontally centers the
search box within the visible map area. Its top edge
aligns with the Live Map menu hover row: `6.5rem` from the top when the sidebar
is expanded and `4.5rem` when the sidebar is collapsed. The panel opens and
closes with matching 420ms animations; closing keeps the panel mounted in a
`closing` state until the exit animation finishes. The position uses the shadcn
sidebar `data-state` and shared sidebar width variables, so the search box moves
right while the menu is expanded and recenters when the menu collapses to the
icon rail. While visible, a document-level pointer handler closes the search
view when the user clicks outside the search panel. Airport and calendar
popover content plus the sidebar Flight Search trigger are considered part of
the search interaction surface, so those clicks do not race the outside-click
handler. Only the search state blurs the Mapbox background; during `closing`,
the blur class is removed while the panel is still mounted, allowing the
background filter/scale transition to release smoothly in parallel with the
panel exit. The sidebar, menu trigger, and search panel are never blurred. When
open, the panel loads an airport departure or arrival board and exposes a
`Show on map` action for each board or flight-number result. A flight-number
Search shows non-ANA/JAL FlightAware-style ident examples such as `UAL130`,
`DAL276`, `AAL176`, `CPA509`, and `SIA12`. It also selects the first returned
result and starts the same
route/final-track/position refresh plus detail/map-data loading path while keeping the
search panel open. The explicit `Show on map` action selects the candidate,
increments an explicit flight-data request counter so pressing it again for the
same flight retries, requests route/final-track/position refresh work, shows a shadcn
Spinner on the pending candidate while candidate buttons are disabled, briefly
retries detail/map-data reads when an actual track refresh was accepted but
track GeoJSON is not ready yet, then closes the search panel and asks Mapbox to focus the aircraft
when current position data is available. If no current position is available,
Mapbox fits the viewport to the selected planned route, actual track, and
current-position GeoJSON bounds. Focus requests remain pending until either
current position coordinates or route bounds are usable, so an early request is
not dropped while selected-flight map data is still loading. The panel displays normalized flight summary
facts, shows current position metrics when available, and summarizes data
freshness as user-facing updated/outdated copy instead of exposing cache source,
checked, stale, and expiry metadata.
The current aircraft marker uses shadcn `HoverCard` to show the selected flight
ident, route, status, aircraft type, registration, altitude, speed, and
human-readable last update timestamp on hover without turning the map marker
into a persistent side panel. Internal data source and heading diagnostics stay
out of the hover card.
No duplicate fixed aircraft heading indicator is rendered in a viewport corner.
Pressing `Tracked Flights` in the sidebar flies the map camera to that current
aircraft marker when a current position is available.
Map layer diagnostics are intentionally omitted from the search panel so the airport-board
flow stays focused on finding and opening a flight. Load and flight-number
Search share one fixed action-column width, so the lower flight-number input
absorbs the remaining row width while both submit buttons stay aligned. The
search panel box, shadcn-backed buttons, input, airport/date popover layers,
results, alerts, and compact status labels use a 4px radius so the search
surface has one consistent corner treatment. Manual refresh only requests
`position`, `route`, and `final_track` tasks, uses the same short actual-track
hydration retry as `Show on map`, and is disabled while loading or when usage
status reports fetching disabled, a budget stop, or an active rate limit.

The App Router loading state uses `src/app/(app)/loading.tsx` and
`src/app/(app)/loading.module.css` with the documented shadcn Progress usage.
The loading surface is shared by every route in the `(app)` group and uses the
root shadcn background token instead of map-only surface tokens, so route
reloads and transitions show a compact progress bar without introducing a white
flash.

The frontend also includes shadcn `Card`, `Chart`, `Badge`, and `Alert`
primitives for reusable surfaces, plus shadcn `Command`, `Popover`, and
`Calendar` for airport-board discovery. Command keeps cmdk generated-heading and
item-icon Tailwind in named class constants and routes the input, group, and
item surfaces through explicit internal subcomponents. The Usage & Limits screen composes
Card for readable grouping, Chart for the active month's API usage cost, Progress
for month-to-date cost against the soft stop threshold, Badge for compact usage
labels, and Alert for active stop reasons.

The `/usage` route renders the Usage & Limits workflow inside the same sidebar
shell as the map. It fetches usage status in the browser through
`AirpathApiClient.getUsageStatus()`. The visible screen keeps only operator
decision data: month-to-date cost against soft stop, stop reason, rate-limit
reset time when the API provides one, availability when fetching is enabled with
no active fetching pause, budget stop, or rate-limit stop, user-facing API
errors, last checked timestamp in the shared UTC display format, and monthly API
usage cost.
`estimatedMonthToDateCost` is visualized as a single monthly bar with the UTC
last checked timestamp below it; budget consumption is shown as numbers and a
progress indicator.
Backend secrets, API keys, queue payloads, projections, cache source, and
operational stop/resume controls stay out of the frontend. The page content
starts below the fixed sidebar trigger toolbar with deliberate top spacing, and
the Usage inset remains the scroll container because the app body is fixed for
the fullscreen map shell. The Usage sidebar shell keeps the provider as the flex
layout wrapper, allowing the reserved sidebar gap to keep the heading and cards
outside the opening desktop menu. Usage spacing, toolbar positioning, scroll
behavior, and sidebar-state overrides belong to `usage-workspace.module.css`.

The sidebar menu background is centralized as
`--sidebar-menu-background: var(--background)`, matching shadcn's default dark
background token. `--sidebar` references that token, while `--sidebar-border`
references `var(--border)` so the menu panel and its vertical border line follow
the default shadcn dark palette. The menu background token is exposed to Tailwind
as `--color-sidebar-menu-background`. The map-side sidebar trigger uses
`var(--foreground)` for the icon and `var(--accent)` for hover, and does not keep
a separate collapsed-state or focus background. The trigger button radius is
forced to 4px, so its hover background box matches the Mapbox control corner
radius instead of shadcn's default icon-button radius. When the sidebar is
collapsed, the trigger uses `--sidebar-width-icon` for its x offset and
`--map-sidebar-collapsed-trigger-y`, derived from the collapsed identity icon
center and the toolbar center, for its y offset. This keeps all sidebar shells
vertically aligned without covering the identity icon. The trigger transform is
composed from `--map-sidebar-trigger-x` and `--map-sidebar-trigger-y`, which
prevents open/close animation from temporarily dropping the x or y offset.
Desktop sidebar side borders also use `border-sidebar-border` so menu open/close
does not reveal a mismatched vertical border. Global CSS must not set generated
sidebar gap or icon-menu dimensions through broad `data-slot` selectors; those
app-specific layout choices belong to `AppSidebar` classes.

The frontend imports workspace packages through their package exports. Its
verification and build commands run `build:deps` first, which generates the
`@airpath/shared-types` `dist` artifacts needed by TypeScript on clean CI
workers.

## Commands

Run the frontend dev server from the repository root with:

```sh
pnpm dev
```

Do not run `npx next dev` from the repository root. The root directory is a
pnpm workspace; the Next.js app root is `frontend`.

Run these commands from `frontend`:

```sh
pnpm dev
pnpm build
pnpm start
pnpm typecheck
```

Run these commands from the repository root:

```sh
pnpm dev
pnpm --filter @airpath/web dev
pnpm --filter @airpath/web build
```

The frontend app root is `frontend`. App Router files belong under `frontend/src/app`; the root page renders the map workspace shell with the documented shadcn Radix Sidebar composition and fullscreen Mapbox surface. Sidebar layout code belongs in `frontend/src/components/app-sidebar.tsx`; the map workspace feature only composes that sidebar with the map inset.
