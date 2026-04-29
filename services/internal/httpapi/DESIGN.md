# HTTP API Adapter Design

The `httpapi` package maps API Gateway HTTP API v2 events to application use cases and typed JSON responses. It keeps Lambda event parsing separate from application logic so handlers can be tested without AWS.

The adapter surface covers the MVP flight search, flight detail, map-data, refresh, and usage-status routes. It translates request paths, methods, query values, and JSON bodies into application inputs, then maps application errors into the shared typed API error response.
