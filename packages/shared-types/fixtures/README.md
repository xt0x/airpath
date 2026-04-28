# Shared Fixtures

This directory contains fixtures shared by API contracts, Go API tests, and Next.js UI tests.

L1 only creates the fixture location. Before committing real FlightAware responses, anonymize API keys, flight numbers, aircraft registrations, and user information.

## L3-02 Schedules Fixture

`schedules/future-without-fa-flight-id.json` captures the `/schedules` response shape needed by L3-02: future scheduled flights can have `fa_flight_id: null` while still carrying the fields needed to create a provisional flight leg ID (`ident`, origin, destination, and `scheduled_out`).

The local environment did not provide a FlightAware AeroAPI key during this implementation, so the fixture is OpenAPI-shaped and fully anonymized. Replace it with a sanitized real `/schedules` response when L3 API access is available.
