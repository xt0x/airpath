# Shared Fixtures

This directory contains fixtures shared by API contracts, Go API tests, and Next.js UI tests.

L1 only creates the fixture location. Before committing real FlightAware responses, anonymize API keys, flight numbers, aircraft registrations, and user information.

## FlightAware Personal Endpoint Table

`flightaware-personal-endpoints.json` records the endpoint set used by the free-allowance MVP:

- `GET /flights/{ident}`
- `GET /flights/{id}/position`
- `GET /flights/{id}/route`
- `GET /flights/{id}/track`
- `GET /schedules/{date_start}/{date_end}`
- `GET /account/usage`

It also records the guardrails for Personal usage: backend-only calls, `max_pages = 1`, opt-in real-call tests, local accounting, and budget stop behavior.

## Flight Ident Search Fixture

`search/flight-ident-search.json` captures the `/flights/{ident}` response shape needed by the MVP search and summary flow. It includes one current flight with a FlightAware flight ID and one future scheduled flight without a FlightAware flight ID so provisional ID handling remains testable.

The local environment did not provide a FlightAware AeroAPI key during this implementation, so the fixture is OpenAPI-shaped and fully anonymized. Replace it with a sanitized real `/flights/{ident}` response when Personal endpoint verification access is available.

## Future Schedule Fixture Without FlightAware Flight ID

`schedules/future-without-fa-flight-id.json` captures the `/schedules` response shape for future scheduled flights where `fa_flight_id` can be `null` while still carrying the fields needed to create a provisional flight leg ID (`ident`, origin, destination, and `scheduled_out`).

The local environment did not provide a FlightAware AeroAPI key during this implementation, so the fixture is OpenAPI-shaped and fully anonymized. Replace it with a sanitized real `/schedules` response when Personal endpoint verification access is available.

## Flight Route Response Fixtures

`routes/flight-route-cases.json` captures the `/flights/{id}/route` cases needed by the free-allowance MVP:

- decoded route with coordinate-bearing fixes
- route string fallback where decoded fix coordinates are unavailable
- route unavailable response

The local environment did not provide a FlightAware AeroAPI key during this implementation, so these fixtures are OpenAPI-shaped and fully anonymized. Replace them with sanitized real `/route` responses when Personal endpoint verification access is available.

## Current Position Response Fixtures

`positions/current-position-cases.json` captures the `/flights/{id}/position` cases needed by the free-allowance MVP:

- current position with displayable coordinates
- current position unavailable

The local environment did not provide a FlightAware AeroAPI key during this implementation, so these fixtures are OpenAPI-shaped and fully anonymized. Replace them with sanitized real `/position` responses when Personal endpoint verification access is available.

## Flight Track Response Fixtures

`tracks/flight-track-cases.json` captures the `/flights/{id}/track` cases needed by the free-allowance MVP:

- coordinate-bearing ordered track positions
- track unavailable response

The local environment did not provide a FlightAware AeroAPI key during this implementation, so these fixtures are OpenAPI-shaped and fully anonymized. Replace them with sanitized real `/track` responses when Personal endpoint verification access is available.

## Account Usage Fixture

`usage/account-usage.json` captures the internal usage-check contract used by the free-allowance guard. The exact FlightAware `/account/usage` response shape must be verified when Personal endpoint access is available; until then, this fixture pins the budget fields Airpath needs for local accounting and stop behavior.
