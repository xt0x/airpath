# Shared Types Source Design

This source directory owns TypeScript domain contracts that are shared by the web app, API contract work, and fixtures.

## L2-01 Domain Model Baseline

The initial domain model exports basic, dependency-free TypeScript types for flights, airports, positions, events, persistent user watches, and WebSocket session subscriptions. These types intentionally keep nullable FlightAware-derived fields as `null` instead of defaulting to empty strings or zero values.

ID generation, nullable display helpers, time normalization, unit conversion, duration calculation, and event dedupe key generation are left to later L2 work items.
