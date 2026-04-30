import { readFileSync } from "node:fs";

import { describe, expect, it } from "vitest";

import {
  generateInternalFlightLegId,
  generateProvisionalFlightLegId,
  normalizeFlightPositionMetrics,
  normalizeLocalDateTimeToUtcIso,
  normalizeUtcIsoDateTime,
  nullableProgressDisplayText,
  toNullableDisplayValue,
} from "../../src/index.js";

interface DomainHelperGolden {
  idGeneration: {
    provisional: {
      input: Parameters<typeof generateProvisionalFlightLegId>[0];
      expected: string;
    };
    internal: {
      input: Parameters<typeof generateInternalFlightLegId>[0];
      expected: string;
    };
  };
  timeNormalization: {
    utcInput: string;
    localInput: Parameters<typeof normalizeLocalDateTimeToUtcIso>[0];
    expectedUtc: string;
  };
  nullableDisplay: {
    missingReason: "not_acquired" | "not_announced" | "not_applicable" | "unavailable";
    expectedMissingLabel: string;
    progress: number;
    expectedProgress: string;
  };
  positionMetrics: {
    input: Parameters<typeof normalizeFlightPositionMetrics>[0];
    expected: ReturnType<typeof normalizeFlightPositionMetrics>;
  };
}

const golden = JSON.parse(
  readFileSync(new URL("./domain-helper-golden.json", import.meta.url), "utf8"),
) as DomainHelperGolden;

describe("domain helper golden fixture", () => {
  it("pins helpers that are implemented in TypeScript and Go", () => {
    expect(generateProvisionalFlightLegId(golden.idGeneration.provisional.input)).toBe(
      golden.idGeneration.provisional.expected,
    );
    expect(generateInternalFlightLegId(golden.idGeneration.internal.input)).toBe(
      golden.idGeneration.internal.expected,
    );
    expect(normalizeUtcIsoDateTime(golden.timeNormalization.utcInput)).toBe(
      golden.timeNormalization.expectedUtc,
    );
    expect(normalizeLocalDateTimeToUtcIso(golden.timeNormalization.localInput)).toBe(
      golden.timeNormalization.expectedUtc,
    );
    expect(toNullableDisplayValue(null, golden.nullableDisplay.missingReason)).toMatchObject({
      label: golden.nullableDisplay.expectedMissingLabel,
    });
    expect(
      nullableProgressDisplayText(
        golden.nullableDisplay.progress,
        golden.nullableDisplay.missingReason,
      ),
    ).toBe(golden.nullableDisplay.expectedProgress);
    expect(normalizeFlightPositionMetrics(golden.positionMetrics.input)).toEqual(
      golden.positionMetrics.expected,
    );
  });
});
