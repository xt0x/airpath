import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { UsageLimitsView } from "@/features/usage/components/usage-limits-view";
import type { UsageStatusResponse } from "@/features/flights/types";

describe("UsageLimitsView", () => {
  it("renders operator status, budget progress, monthly usage cost chart, and cache metadata", () => {
    const html = renderToStaticMarkup(
      <UsageLimitsView state={{ kind: "ready", status: usageStatus() }} />,
    );

    expect(html).toContain("Usage &amp; Limits");
    expect(html).toContain("Available");
    expect(html).toContain('aria-label="Overall fetch status is available"');

    expect(html).toContain("API cost");
    expect(html).toContain("$2.50 / $4.00");
    expect(html).toContain("63% used");
    expect(html).toContain("API usage cost");
    expect(html).toContain("April 2026");
    expect(html).toContain('aria-label="API usage cost by month for April 2026"');
    expect(html).toContain("Last checked");
    expect(html).toContain("Apr 29, 2026, 12:00 AM UTC");

    expect(html).not.toContain("Fetch status");
    expect(html).not.toContain("Current usage status payload");
    expect(html).not.toContain("local_accounting");
  });

  it("does not repeat the budget summary beside the monthly usage chart month label", () => {
    const html = renderToStaticMarkup(
      <UsageLimitsView
        state={{
          kind: "ready",
          status: usageStatus({
            budget: {
              month: "2026-05",
              estimatedMonthToDateCost: 0.13,
              softStopThreshold: 4,
            },
          }),
        }}
      />,
    );

    expect(html).toContain("May 2026");
    expect(html).toContain("$0.13 / $4.00");
    expect(html.match(/\$0\.13 \/ \$4\.00/g)).toHaveLength(1);
  });

  it("renders stop reasons and rate limit reset time when fetching is paused", () => {
    const html = renderToStaticMarkup(
      <UsageLimitsView
        state={{
          kind: "ready",
          status: usageStatus({
            fetchingEnabled: false,
            budget: { estimatedMonthToDateCost: 4.5, softStopThreshold: 4, stopped: true },
            rateLimit: { limited: true, resetAt: "2026-04-29T01:00:00Z" },
          }),
        }}
      />,
    );

    expect(html).toContain("Usage &amp; Limits");
    expect(html).toContain("Fetching paused");
    expect(html).not.toContain("Available");
    expect(html).toContain("Budget stop");
    expect(html).toContain("Rate limit");
    expect(html).toContain("Rate limit reset");
    expect(html).toContain("2026");
    expect(html).toContain("$4.50 / $4.00");
  });

  it("does not report availability while an active rate limit is present", () => {
    const html = renderToStaticMarkup(
      <UsageLimitsView
        state={{
          kind: "ready",
          status: usageStatus({
            fetchingEnabled: true,
            rateLimit: { limited: true, resetAt: null },
          }),
        }}
      />,
    );

    expect(html).not.toContain("Available");
    expect(html).not.toContain("Rate limit reset");
    expect(html).toContain("Rate limit");
  });

  it("renders paused fetching as a stop reason without a budget or rate-limit stop", () => {
    const html = renderToStaticMarkup(
      <UsageLimitsView
        state={{
          kind: "ready",
          status: usageStatus({
            fetchingEnabled: false,
            budget: { stopped: false },
            rateLimit: { limited: false, resetAt: null },
          }),
        }}
      />,
    );

    expect(html).not.toContain("Available");
    expect(html).toContain("Fetching paused");
    expect(html).not.toContain("Budget stop");
    expect(html).not.toContain("Rate limit");
  });

  it("keeps loading empty and renders usage status errors without usage details", () => {
    const loading = renderToStaticMarkup(<UsageLimitsView state={{ kind: "loading" }} />);
    const error = renderToStaticMarkup(
      <UsageLimitsView state={{ kind: "error", message: "Request failed" }} />,
    );

    expect(loading).toContain("Usage &amp; Limits");
    expect(loading).not.toContain("Available");
    expect(loading).not.toContain("API cost");
    expect(loading).not.toContain("Last checked");

    expect(error).toContain("Usage &amp; Limits");
    expect(error).toContain("Usage status unavailable");
    expect(error).toContain("Request failed");
    expect(error).not.toContain("API cost");
    expect(error).not.toContain("Last checked");
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
