# Shared Fixtures

This directory contains fixtures shared by API contracts, Go API tests, and Next.js UI tests.

L1 only creates the fixture location. Before committing real FlightAware responses, anonymize API keys, flight numbers, aircraft registrations, and user information.

## Future Schedule Fixture Without FlightAware Flight ID

`schedules/future-without-fa-flight-id.json` captures the `/schedules` response shape for future scheduled flights where `fa_flight_id` can be `null` while still carrying the fields needed to create a provisional flight leg ID (`ident`, origin, destination, and `scheduled_out`).

The local environment did not provide a FlightAware AeroAPI key during this implementation, so the fixture is OpenAPI-shaped and fully anonymized. Replace it with a sanitized real `/schedules` response when Personal endpoint verification access is available.

## Flight Route Response Fixtures

`routes/flight-route-cases.json` captures the `/flights/{id}/route` cases needed by the free-allowance MVP:

- decoded route with coordinate-bearing fixes
- route string fallback where decoded fix coordinates are unavailable
- route unavailable response

The local environment did not provide a FlightAware AeroAPI key during this implementation, so these fixtures are OpenAPI-shaped and fully anonymized. Replace them with sanitized real `/route` responses when Personal endpoint verification access is available.
