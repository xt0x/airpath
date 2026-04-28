import { describe, expect, it } from "vitest";

import type { Airport, Flight, FlightTimes, FlightEvent, FlightPosition } from "./index.js";
import { generateInternalFlightLegId, generateProvisionalFlightLegId } from "./index.js";
import {
  calculateFlightDuration,
  generateFlightEventDedupeKey,
  normalizeAltitudeFeet,
  normalizeFlightPositionMetrics,
  normalizeGroundspeedKnots,
  normalizeHeadingDegrees,
  nullableAirportDisplayText,
  nullableDateTimeDisplayText,
  nullableProgressDisplayText,
  nullableTextDisplayText,
  normalizeLocalDateTimeToUtcIso,
  normalizeUtcIsoDateTime,
  toNullableDisplayValue,
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
});

describe("flight leg ID generation", () => {
  it("generates a stable provisionalFlightLegId from schedule fields", () => {
    const input = {
      ident: "ANA110",
      originCode: "RJTT",
      destinationCode: "KJFK",
      scheduledOut: "2026-08-01T01:00:00Z",
    };

    expect(generateProvisionalFlightLegId(input)).toBe("sched_c9df8a3ee088");
    expect(generateProvisionalFlightLegId(input)).toBe(generateProvisionalFlightLegId(input));
    expect(
      generateProvisionalFlightLegId({
        ...input,
        destinationCode: "KLAX",
      }),
    ).toBe("sched_b76cfb4f33db");
  });

  it("generates a stable internalFlightLegId from FlightAware and leg fields", () => {
    const input = {
      faFlightId: "UAL1234-1234567890-airline-0123",
      originCode: "KSFO",
      destinationCode: "RJTT",
      scheduledOut: "2026-04-25T10:00:00Z",
      legIndex: 0,
    };

    expect(generateInternalFlightLegId(input)).toBe("iflg_de338117b7ac");
    expect(generateInternalFlightLegId(input)).toBe(generateInternalFlightLegId(input));
    expect(
      generateInternalFlightLegId({
        ...input,
        legIndex: 1,
      }),
    ).toBe("iflg_de338217b7ac");
  });
});

describe("nullable display conversion", () => {
  it("keeps missing time, aircraft, and registration values explicit", () => {
    expect(nullableDateTimeDisplayText(null, "not_announced")).toBe("未発表");
    expect(nullableTextDisplayText(null, "not_acquired")).toBe("未取得");
    expect(nullableTextDisplayText(undefined, "unavailable")).toBe("取得不可");
    expect(nullableTextDisplayText("B789", "not_acquired")).toBe("B789");
    expect(nullableTextDisplayText("N12345", "not_acquired")).toBe("N12345");
  });

  it("does not treat zero progress as missing", () => {
    expect(nullableProgressDisplayText(null, "not_acquired")).toBe("未取得");
    expect(nullableProgressDisplayText(0, "not_acquired")).toBe("0%");
    expect(nullableProgressDisplayText(62, "not_acquired")).toBe("62%");
  });

  it("keeps missing airport details explicit without hiding known airport codes", () => {
    const airportWithMissingDetails: Airport = {
      code: "RJTT",
      name: null,
      timezone: null,
    };

    expect(nullableAirportDisplayText(null, "not_acquired")).toBe("未取得");
    expect(nullableAirportDisplayText(airportWithMissingDetails, "not_acquired")).toBe("RJTT");
    expect(
      nullableAirportDisplayText(
        { ...airportWithMissingDetails, name: "Tokyo Haneda" },
        "not_acquired",
      ),
    ).toBe("RJTT - Tokyo Haneda");
  });

  it("returns structured missing metadata for API callers", () => {
    expect(toNullableDisplayValue(null, "not_applicable")).toEqual({
      kind: "missing",
      reason: "not_applicable",
      label: "対象外",
    });
    expect(toNullableDisplayValue("2026-04-25T10:00:00Z", "not_announced")).toEqual({
      kind: "available",
      value: "2026-04-25T10:00:00Z",
    });
  });
});

describe("time normalization", () => {
  it("normalizes FlightAware ISO timestamps to UTC ISO 8601 seconds", () => {
    expect(normalizeUtcIsoDateTime("2026-04-25T10:00:00Z")).toBe("2026-04-25T10:00:00Z");
    expect(normalizeUtcIsoDateTime("2026-04-25T03:00:00-07:00")).toBe("2026-04-25T10:00:00Z");
    expect(normalizeUtcIsoDateTime("2026-04-25T10:00:00.123Z")).toBe("2026-04-25T10:00:00Z");
  });

  it("normalizes screen local date-time input with an IANA timezone", () => {
    expect(
      normalizeLocalDateTimeToUtcIso({
        localDateTime: "2026-04-25T19:00:00",
        timeZone: "Asia/Tokyo",
      }),
    ).toBe("2026-04-25T10:00:00Z");
    expect(
      normalizeLocalDateTimeToUtcIso({
        localDateTime: "2026-04-25T03:00:00",
        timeZone: "America/Los_Angeles",
      }),
    ).toBe("2026-04-25T10:00:00Z");
  });

  it("rejects invalid time inputs instead of guessing", () => {
    expect(() => normalizeUtcIsoDateTime("2026-04-25 10:00:00")).toThrow("Invalid ISO 8601");
    expect(() =>
      normalizeLocalDateTimeToUtcIso({
        localDateTime: "2026-04-25T19:00:00",
        timeZone: "Not/AZone",
      }),
    ).toThrow("Invalid IANA timezone");
  });
});

describe("flight position metric conversion", () => {
  it("converts FlightAware altitude hundreds of feet to feet", () => {
    expect(normalizeAltitudeFeet(370)).toBe(37000);
    expect(normalizeAltitudeFeet(0)).toBe(0);
    expect(normalizeAltitudeFeet(null)).toBeNull();
    expect(normalizeAltitudeFeet(undefined)).toBeNull();
  });

  it("keeps missing speed and heading values as null", () => {
    expect(normalizeGroundspeedKnots(488)).toBe(488);
    expect(normalizeGroundspeedKnots(null)).toBeNull();
    expect(normalizeHeadingDegrees(275)).toBe(275);
    expect(normalizeHeadingDegrees(null)).toBeNull();
  });

  it("normalizes heading 360 degrees to 0 degrees for display rotation", () => {
    expect(normalizeHeadingDegrees(0)).toBe(0);
    expect(normalizeHeadingDegrees(360)).toBe(0);
  });

  it("converts a position metric set without inventing missing values", () => {
    expect(
      normalizeFlightPositionMetrics({
        altitudeHundredsFeet: 370,
        groundspeedKnots: 488,
        headingDegrees: 360,
      }),
    ).toEqual({
      altitudeHundredsFeet: 370,
      altitudeFeet: 37000,
      groundspeedKnots: 488,
      headingDegrees: 0,
    });
    expect(
      normalizeFlightPositionMetrics({
        altitudeHundredsFeet: null,
        groundspeedKnots: null,
        headingDegrees: null,
      }),
    ).toEqual({
      altitudeHundredsFeet: null,
      altitudeFeet: null,
      groundspeedKnots: null,
      headingDegrees: null,
    });
  });
});

describe("flight duration calculation", () => {
  const baseTimes: FlightTimes = {
    scheduledOut: "2026-04-25T09:30:00Z",
    estimatedOut: null,
    actualOut: "2026-04-25T10:02:00Z",
    scheduledOff: "2026-04-25T10:00:00Z",
    estimatedOff: "2026-04-25T10:05:00Z",
    actualOff: "2026-04-25T10:08:00Z",
    scheduledOn: "2026-04-25T21:40:00Z",
    estimatedOn: "2026-04-25T21:35:00Z",
    actualOn: "2026-04-25T21:30:00Z",
    scheduledIn: null,
    estimatedIn: null,
    actualIn: "2026-04-25T21:50:00Z",
  };

  it("uses actual runway times before lower-priority duration sources", () => {
    expect(calculateFlightDuration({ times: baseTimes, filedEteSeconds: 42000 })).toEqual({
      kind: "actual",
      seconds: 40920,
      startAt: "2026-04-25T10:08:00Z",
      endAt: "2026-04-25T21:30:00Z",
    });
  });

  it("falls back through estimated, scheduled, filed, and actual gate duration", () => {
    expect(
      calculateFlightDuration({
        times: {
          ...baseTimes,
          actualOff: null,
          actualOn: null,
        },
        filedEteSeconds: 42000,
      }),
    ).toEqual({
      kind: "estimated",
      seconds: 41400,
      startAt: "2026-04-25T10:05:00Z",
      endAt: "2026-04-25T21:35:00Z",
    });
    expect(
      calculateFlightDuration({
        times: {
          ...baseTimes,
          actualOff: null,
          actualOn: null,
          estimatedOff: null,
          estimatedOn: null,
        },
        filedEteSeconds: 42000,
      }),
    ).toEqual({
      kind: "scheduled",
      seconds: 42000,
      startAt: "2026-04-25T10:00:00Z",
      endAt: "2026-04-25T21:40:00Z",
    });
    expect(
      calculateFlightDuration({
        times: {
          ...baseTimes,
          actualOff: null,
          actualOn: null,
          estimatedOff: null,
          estimatedOn: null,
          scheduledOff: null,
          scheduledOn: null,
        },
        filedEteSeconds: 42000,
      }),
    ).toEqual({
      kind: "filed",
      seconds: 42000,
      startAt: null,
      endAt: null,
    });
    expect(
      calculateFlightDuration({
        times: {
          ...baseTimes,
          actualOff: null,
          actualOn: null,
          estimatedOff: null,
          estimatedOn: null,
          scheduledOff: null,
          scheduledOn: null,
        },
        filedEteSeconds: null,
      }),
    ).toEqual({
      kind: "gate_actual",
      seconds: 42480,
      startAt: "2026-04-25T10:02:00Z",
      endAt: "2026-04-25T21:50:00Z",
    });
  });

  it("returns null when no complete duration source is available", () => {
    expect(
      calculateFlightDuration({
        times: {
          scheduledOut: null,
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
      }),
    ).toBeNull();
  });
});

describe("flight event dedupe key generation", () => {
  it("uses provisionalFlightLegId when faFlightId is not available", () => {
    expect(
      generateFlightEventDedupeKey({
        eventType: "departure",
        faFlightId: null,
        provisionalFlightLegId: "sched_3f5a7c8d91ab",
        eventTimestamp: "2026-04-25T10:08:00Z",
      }),
    ).toBe("13b84e62b4d0");
  });

  it("generates a stable polling event dedupe key without an external notification id", () => {
    expect(
      generateFlightEventDedupeKey({
        eventType: "arrival",
        faFlightId: "iflg_8d2a2a3a6c4f",
        provisionalFlightLegId: null,
        eventTimestamp: "2026-04-25T21:50:00Z",
      }),
    ).toBe("b22238bbb5ee");
  });

  it("requires a FlightAware or provisional flight key", () => {
    expect(() =>
      generateFlightEventDedupeKey({
        eventType: "status_updated",
        faFlightId: null,
        provisionalFlightLegId: null,
        eventTimestamp: "2026-04-25T21:50:00Z",
      }),
    ).toThrow("faFlightId or provisionalFlightLegId is required for event dedupe keys");
  });
});
