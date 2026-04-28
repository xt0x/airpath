import { describe, expect, it } from "vitest";

import usageResponse from "./account-usage.json";

describe("account usage fixture", () => {
  it("captures the usage-check contract used by the free-allowance guard", () => {
    expect(usageResponse.source.fixtureName).toBe("account-usage-response");
    expect(usageResponse.source.endpoint).toBe("GET /account/usage");
    expect(usageResponse.source).not.toHaveProperty("planItem");
    expect(usageResponse.response.currency).toBe("USD");
  });

  it("keeps endpoint-level usage details for reconciliation", () => {
    expect(usageResponse.response.endpoints).toEqual([
      {
        endpoint: "GET /flights/{ident}",
        result_sets: 12,
        estimated_cost_usd: 0.06,
      },
      {
        endpoint: "GET /flights/{id}/position",
        result_sets: 18,
        estimated_cost_usd: 0.18,
      },
    ]);
  });

  it("shows a soft-stop threshold before the Personal free allowance is exhausted", () => {
    expect(usageResponse.response.month_to_date.estimated_cost_usd).toBeLessThan(
      usageResponse.response.personal_free_allowance_usd,
    );
    expect(usageResponse.response.guard.soft_stop_threshold_usd).toBeLessThan(
      usageResponse.response.personal_free_allowance_usd,
    );
  });
});
