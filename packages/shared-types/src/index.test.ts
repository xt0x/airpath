import { describe, expect, it } from "vitest";

import type {
  Airport,
  Flight,
  FlightEvent,
  FlightPosition,
  FlightSubscription,
  UserWatch,
} from "./index.js";

describe("shared domain models", () => {
  it("represents a flight with nullable FlightAware fields", () => {
    const origin: Airport = {
      code: "RJTT",
      name: "Tokyo Haneda",
      timezone: "Asia/Tokyo",
    };
    const destination: Airport = {
      code: "KJFK",
      name: "John F. Kennedy International Airport",
      timezone: "America/New_York",
    };
    const flight: Flight = {
      flightId: "sched_3f5a7c8d91ab",
      flightIdType: "provisional",
      internalFlightLegId: null,
      provisionalFlightLegId: "sched_3f5a7c8d91ab",
      faFlightId: null,
      ident: "ANA110",
      identIata: "NH110",
      operator: "ANA",
      aircraftType: null,
      registration: null,
      origin,
      destination,
      originalDestination: null,
      diverted: false,
      legIndex: null,
      status: "Scheduled",
      progressPercent: null,
      times: {
        scheduledOut: "2026-08-01T01:00:00Z",
        estimatedOut: null,
        actualOut: null,
        scheduledOff: null,
        estimatedOff: null,
        actualOff: null,
        scheduledOn: null,
        estimatedOn: null,
        actualOn: null,
        scheduledIn: null,
        estimatedIn: null,
        actualIn: null,
      },
      filedEteSeconds: null,
      plannedRouteS3Key: null,
      actualTrackS3Key: null,
      latestPositionTimestamp: null,
      latestPositionSource: null,
      pollState: "scheduled",
      watcherCount: 0,
      sessionSubscriberCount: 0,
      persistentWatcherCount: 0,
      fetchLeaseUntil: null,
      fetchOwner: null,
      nextSummaryPollAt: null,
      nextPositionPollAt: null,
      nextTrackPollAt: null,
      nextRoutePollAt: null,
      idleSince: null,
      updatedAt: "2026-04-25T15:00:00Z",
      ttl: null,
    };

    expect(flight.faFlightId).toBeNull();
    expect(flight.aircraftType).toBeNull();
    expect(flight.registration).toBeNull();
    expect(flight.times.estimatedOut).toBeNull();
  });

  it("represents position and event records without deriving later L2 helper behavior", () => {
    const position: FlightPosition = {
      flightId: "iflg_8d2a2a3a6c4f",
      internalFlightLegId: "iflg_8d2a2a3a6c4f",
      provisionalFlightLegId: null,
      faFlightId: "UAL1234-1234567890-airline-0123",
      latitude: 45.123,
      longitude: 160.456,
      altitudeHundredsFeet: 370,
      altitudeFeet: 37000,
      altitudeChange: "level",
      groundspeedKnots: 488,
      headingDegrees: 275,
      timestamp: "2026-04-25T15:05:00Z",
      updateType: "estimated",
      source: "flightaware_position",
      ttl: null,
    };
    const event: FlightEvent = {
      flightId: "iflg_8d2a2a3a6c4f",
      internalFlightLegId: "iflg_8d2a2a3a6c4f",
      provisionalFlightLegId: null,
      faFlightId: "UAL1234-1234567890-airline-0123",
      dedupeKey: "dedupe_123",
      eventType: "departure",
      eventTimestamp: "2026-04-25T10:08:00Z",
      payload: { sourceStatus: "En Route" },
      source: "polling",
      appliedToFlightState: true,
      createdAt: "2026-04-25T15:05:01Z",
    };

    expect(position.altitudeFeet).toBe(37000);
    expect(position.source).toBe("flightaware_position");
    expect(event.payload["sourceStatus"]).toBe("En Route");
  });

  it("keeps persistent watches separate from session subscriptions", () => {
    const watch: UserWatch = {
      userId: "user_123",
      flightId: "iflg_8d2a2a3a6c4f",
      faFlightId: "UAL1234-1234567890-airline-0123",
      flightIdType: "internal",
      watchState: "active",
      createdAt: "2026-04-25T15:00:00Z",
      updatedAt: "2026-04-25T15:00:00Z",
      ttl: null,
    };
    const subscription: FlightSubscription = {
      flightId: "iflg_8d2a2a3a6c4f",
      connectionId: "abc123",
      userId: "user_123",
      subscriptionType: "session",
      createdAt: "2026-04-25T15:01:00Z",
      lastSeenAt: "2026-04-25T15:02:00Z",
      ttl: 1770000000,
    };

    expect(watch.watchState).toBe("active");
    expect(subscription.subscriptionType).toBe("session");
  });
});
