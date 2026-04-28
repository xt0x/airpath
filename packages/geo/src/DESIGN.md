# Geo Source Design

This source directory owns dependency-free GeoJSON conversion helpers for the free-allowance MVP map data flow.

## F5 GeoJSON And Map Data Processing

The package converts normalized route, track, and current-position data into GeoJSON features that can be consumed by API responses and map-rendering code without depending on deck.gl or Mapbox.

Planned routes prefer decoded FlightAware route points. When decoded points are unavailable, the package can derive a great-circle fallback from origin and destination airport coordinates. Route and track lines are emitted as `MultiLineString` geometries after antimeridian splitting so a path crossing longitude 180 degrees does not draw across the entire map.

Track helpers produce both an actual-track line feature and display point features. Current-position markers are created only when latitude, longitude, and timestamp are valid. Track simplification keeps departure, arrival, latest, and explicitly important points before filling any remaining display budget.
