# HTTP API Adapter Design

The `httpapi` package maps API Gateway HTTP API v2 events to application use cases and typed JSON responses. It keeps Lambda event parsing separate from application logic so handlers can be tested without AWS.

The first adapter surface covers flight search and shared typed error responses. Additional routes should be added here as their application use cases are wired to runtime repositories.
