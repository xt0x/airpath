# Map Rendering Source Design

This source directory owns shared browser map rendering adapters for the FlightAware free-allowance MVP.

The package converts normalized API map-data responses into deck.gl `GeoJsonLayer` and `ScatterplotLayer` instances for the Mapbox light style. It imports map-data DTOs from the `@airpath/shared-types` workspace package instead of redefining API contract shapes locally; local verification builds that package's declarations before typechecking this package. It does not fetch data, own React state, or render fallback markup. Feature UI can import these adapters while keeping route-specific dashboard state inside the frontend package.
