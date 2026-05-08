import { describe, expect, it } from "vitest";

import { formatDisplayDateTime } from "@/lib/display-date-time";

describe("display date-time formatting", () => {
  it("formats ISO timestamps in UTC for stable server and browser rendering", () => {
    const formatted = formatDisplayDateTime("2026-04-29T00:05:00Z");

    expect(formatted).toContain("Apr 29, 2026");
    expect(formatted).toContain("12:05 AM");
    expect(formatted).toContain("UTC");
    expect(formatted).toContain("2026");
    expect(formatted).not.toBe("2026-04-29T00:05:00Z");
    expect(formatted).not.toContain("T00:05:00Z");
  });

  it("keeps the UTC calendar date instead of shifting to the host time zone", () => {
    expect(formatDisplayDateTime("2026-01-01T00:30:00Z")).toContain("Jan 1, 2026");
  });

  it("returns a configurable fallback for missing or invalid timestamps", () => {
    expect(formatDisplayDateTime(null)).toBe("Unavailable");
    expect(formatDisplayDateTime(undefined, "Not set")).toBe("Not set");
    expect(formatDisplayDateTime("not-a-date")).toBe("Unavailable");
    expect(formatDisplayDateTime("   ", "Missing")).toBe("Missing");
  });
});
