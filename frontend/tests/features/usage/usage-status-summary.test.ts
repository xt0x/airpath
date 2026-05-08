import { describe, expect, it } from "vitest";

import {
  buildMonthlyCostChartData,
  buildUsageStatusSummary,
  formatUsageCurrency,
} from "@/features/usage/lib/usage-status-summary";
import type { UsageStatusResponse } from "@/features/flights/types";

describe("usage status summary", () => {
  it("summarizes budget usage, stop reasons, and last checked time", () => {
    const summary = buildUsageStatusSummary(usageStatus());

    expect(summary.budgetSummaryLabel).toBe("$2.50 / $4.00");
    expect(summary.budgetPercent).toBe(63);
    expect(summary.stopReasons).toEqual([]);
    expect(summary.lastCheckedLabel).toBe("Apr 29, 2026, 12:00 AM UTC");
  });

  it("reports paused fetching with budget and rate-limit stop reasons", () => {
    const summary = buildUsageStatusSummary(
      usageStatus({
        fetchingEnabled: false,
        budget: { estimatedMonthToDateCost: 4.5, softStopThreshold: 4, stopped: true },
        rateLimit: { limited: true, resetAt: "2026-04-29T01:00:00Z" },
        cache: { freshness: "stale", stale: true },
      }),
    );

    expect(summary.budgetSummaryLabel).toBe("$4.50 / $4.00");
    expect(summary.budgetPercent).toBe(100);
    expect(summary.stopReasons).toEqual(["Budget stop", "Rate limit"]);
    expect(summary.rateLimitResetLabel).toContain("2026");
    expect(summary.rateLimitResetLabel).not.toContain("T01:00:00Z");
  });

  it("builds monthly API usage cost chart data for the usage month", () => {
    const data = buildMonthlyCostChartData(usageStatus());

    expect(data).toEqual([
      {
        month: "2026-04",
        monthLabel: "April 2026",
        estimatedMonthToDateCost: 2.5,
        estimatedMonthToDateCostLabel: "$2.50",
      },
    ]);
  });

  it("formats usage values for compact operator display", () => {
    expect(formatUsageCurrency(2.5)).toBe("$2.50");
  });
});

type UsageStatusOverrides = Partial<Omit<UsageStatusResponse, "budget" | "rateLimit" | "cache">> & {
  budget?: Partial<UsageStatusResponse["budget"]>;
  rateLimit?: Partial<UsageStatusResponse["rateLimit"]>;
  cache?: Partial<UsageStatusResponse["cache"]>;
};

function usageStatus(overrides: UsageStatusOverrides = {}): UsageStatusResponse {
  const base: UsageStatusResponse = {
    budget: {
      environment: "dev",
      month: "2026-04",
      currency: "USD",
      estimatedMonthToDateCost: 2.5,
      softStopThreshold: 4,
      stopped: false,
      dailyUsage: [
        { date: "2026-04-02", estimatedCostUSD: 0.12, estimatedCallCount: 12 },
        { date: "2026-04-29", estimatedCostUSD: 0.34, estimatedCallCount: 34 },
      ],
    },
    rateLimit: {
      limited: false,
      resetAt: null,
    },
    fetchingEnabled: true,
    cache: {
      freshness: "fresh",
      source: "local_accounting",
      stale: false,
      checkedAt: "2026-04-29T00:00:00Z",
      fetchedAt: "2026-04-29T00:00:00Z",
      staleAt: "2026-04-29T00:15:00Z",
      expiresAt: "2026-04-29T00:30:00Z",
    },
  };

  return {
    ...base,
    ...overrides,
    budget: {
      ...base.budget,
      ...overrides.budget,
    },
    rateLimit: {
      ...base.rateLimit,
      ...overrides.rateLimit,
    },
    cache: {
      ...base.cache,
      ...overrides.cache,
    },
  };
}
