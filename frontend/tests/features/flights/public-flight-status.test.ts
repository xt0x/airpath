import { describe, expect, expectTypeOf, it } from "vitest";

import { formatPublicFlightStatus } from "@/features/flights/lib/public-flight-status";

describe("public flight status formatting", () => {
  it("preserves English upstream status text before UI display", () => {
    expect(formatPublicFlightStatus("Scheduled")).toBe("Scheduled");
    expect(formatPublicFlightStatus("Arrived / Gate Arrival")).toBe("Arrived / Gate Arrival");
    expect(formatPublicFlightStatus("In Flight / Delayed")).toBe("In Flight / Delayed");
    expect(formatPublicFlightStatus("Taxiing / Gate Departure")).toBe("Taxiing / Gate Departure");
    expect(formatPublicFlightStatus("Departed / Delayed")).toBe("Departed / Delayed");
    expect(formatPublicFlightStatus("Diverted")).toBe("Diverted");
  });

  it("normalizes English status parts separated by ASCII slash punctuation", () => {
    expect(formatPublicFlightStatus("Arrived/Gate Arrival")).toBe("Arrived / Gate Arrival");
    expect(formatPublicFlightStatus("Taxiing / Airline Confirmation")).toBe(
      "Taxiing / Airline Confirmation",
    );
    expect(formatPublicFlightStatus("Taxiing\t/\tGate Departure")).toBe("Taxiing / Gate Departure");
  });

  it("ignores empty status parts created by upstream separator noise", () => {
    expect(formatPublicFlightStatus("Arrived / ")).toBe("Arrived");
    expect(formatPublicFlightStatus(" / Delayed")).toBe("Delayed");
    expect(formatPublicFlightStatus(" / ")).toBe("Unknown");
  });

  it("does not leak non-English status text into public UI", () => {
    expect(formatPublicFlightStatus("Авиакомпания уточняет")).toBe("Unknown");
    expect(formatPublicFlightStatus("Taxiing / Авиакомпания уточняет")).toBe("Taxiing / Unknown");
    expect(formatPublicFlightStatus("Arrived／Gate Arrival")).toBe("Unknown");
    expect(formatPublicFlightStatus("\u00A0Scheduled\u00A0")).toBe("Unknown");
    expect(formatPublicFlightStatus("Arrived\u00A0/\u00A0Gate Arrival")).toBe("Unknown / Unknown");
  });

  it("preserves already-English status values and hides blank display text", () => {
    expect(formatPublicFlightStatus(" En Route ")).toBe("En Route");
    expect(formatPublicFlightStatus("\tEn Route\t")).toBe("En Route");
    expect(formatPublicFlightStatus("")).toBe("Unknown");
    expect(formatPublicFlightStatus(" \t ")).toBe("Unknown");
    expect(formatPublicFlightStatus(null)).toBe(null);
    expect(formatPublicFlightStatus(undefined)).toBe(undefined);
  });

  it("exposes honest public return types for transformed display strings", () => {
    expectTypeOf(formatPublicFlightStatus("Scheduled")).toEqualTypeOf<string>();
    expectTypeOf(formatPublicFlightStatus("En Route")).toEqualTypeOf<string>();
    expectTypeOf(formatPublicFlightStatus(null)).toEqualTypeOf<null>();
    expectTypeOf(formatPublicFlightStatus(undefined)).toEqualTypeOf<undefined>();
  });
});
