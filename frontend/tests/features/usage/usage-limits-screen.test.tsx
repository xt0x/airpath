// @vitest-environment happy-dom

import * as React from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import {
  UsageLimitsScreen,
  type UsageStatusClient,
} from "@/features/usage/components/usage-limits-screen";
import type { UsageStatusResponse } from "@/features/flights/types";

let mountedRoot: Root | null = null;
let mountedContainer: HTMLDivElement | null = null;

describe("UsageLimitsScreen", () => {
  beforeAll(() => {
    (globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;
  });

  afterEach(async () => {
    if (mountedRoot !== null) {
      await React.act(async () => {
        mountedRoot?.unmount();
      });
      mountedContainer?.remove();
      mountedRoot = null;
      mountedContainer = null;
    }
  });

  it("loads usage status through the injected client and renders the ready view", async () => {
    const request = deferred<UsageStatusResponse>();
    const client: UsageStatusClient = {
      getUsageStatus: vi.fn(() => request.promise),
    };

    const container = await renderUsageScreen(client);

    expect(client.getUsageStatus).toHaveBeenCalledOnce();
    expect(container.textContent).toContain("Usage & Limits");
    expect(container.textContent).not.toContain("API cost");

    await React.act(async () => {
      request.resolve(usageStatus());
      await request.promise;
    });

    expect(container.textContent).toContain("Available");
    expect(container.textContent).toContain("API cost");
    expect(container.textContent).toContain("$2.50 / $4.00");
    expect(container.textContent).toContain("Last checked");
  });

  it("renders the client error message when usage status loading fails", async () => {
    const request = deferred<UsageStatusResponse>();
    const client: UsageStatusClient = {
      getUsageStatus: vi.fn(() => request.promise),
    };

    const container = await renderUsageScreen(client);

    await React.act(async () => {
      request.reject(new Error("Usage API unavailable"));
      await request.promise.catch(() => undefined);
    });

    expect(container.textContent).toContain("Usage status unavailable");
    expect(container.textContent).toContain("Usage API unavailable");
    expect(container.textContent).not.toContain("API cost");
  });
});

async function renderUsageScreen(client: UsageStatusClient): Promise<HTMLDivElement> {
  mountedContainer = document.createElement("div");
  document.body.appendChild(mountedContainer);
  mountedRoot = createRoot(mountedContainer);

  await React.act(async () => {
    mountedRoot?.render(<UsageLimitsScreen client={client} />);
  });

  return mountedContainer;
}

function deferred<TValue>() {
  let resolve!: (value: TValue) => void;
  let reject!: (reason: unknown) => void;
  const promise = new Promise<TValue>((innerResolve, innerReject) => {
    resolve = innerResolve;
    reject = innerReject;
  });

  return { promise, resolve, reject };
}

function usageStatus(): UsageStatusResponse {
  return {
    budget: {
      environment: "dev",
      month: "2026-04",
      currency: "USD",
      estimatedMonthToDateCost: 2.5,
      softStopThreshold: 4,
      stopped: false,
      dailyUsage: [],
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
}
