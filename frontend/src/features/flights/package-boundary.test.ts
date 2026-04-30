import { describe, expect, it } from "vitest";
import { existsSync, readFileSync } from "node:fs";

import frontendPackage from "../../../package.json";
import frontendTsconfig from "../../../tsconfig.json";

describe("frontend package boundaries", () => {
  it("declares workspace packages that feature code imports directly", () => {
    expect(frontendPackage.dependencies).toMatchObject({
      "@airpath/map-rendering": "workspace:*",
      "@airpath/shared-types": "workspace:*",
    });
  });

  it("resolves workspace packages through package exports for production builds", () => {
    expect(frontendTsconfig.compilerOptions.paths).not.toHaveProperty("@airpath/shared-types");
    expect(frontendTsconfig.compilerOptions.paths).not.toHaveProperty("@airpath/map-rendering");
  });

  it("builds imported workspace package artifacts before frontend verification", () => {
    expect(frontendPackage.scripts["build:deps"]).toContain(
      "pnpm --filter @airpath/map-rendering build",
    );
  });

  it("keeps flight feature styles scoped and dashboard internals out of global entrypoints", () => {
    const globals = readFileSync("src/app/globals.css", "utf8");
    const featureStyles = readFileSync("src/features/flights/flight-dashboard.css", "utf8");
    const dashboardEntrypoint = readFileSync("src/features/flights/flight-dashboard.tsx", "utf8");
    const dashboardView = readFileSync("src/features/flights/flight-dashboard-view.tsx", "utf8");

    expect(globals).not.toContain(".app-shell");
    expect(featureStyles).toContain(".flight-dashboard");
    expect(featureStyles).not.toMatch(/(^|})\s*h[1-6]\s*[{,]/);
    expect(featureStyles).not.toMatch(/(^|})\s*\.(app-shell|topbar|panel|workspace)\b/);
    expect(dashboardEntrypoint).not.toMatch(/export\s*\{[^}]*SearchPanel/s);
    expect(dashboardEntrypoint).not.toMatch(/export\s*\{[^}]*SummaryPanel/s);
    expect(dashboardView).not.toMatch(
      /export\s+function\s+(SearchPanel|SummaryPanel|RefreshControls|StaleDataNotice|UsageStatusBanner)\b/,
    );
  });

  it("keeps test-only sample dashboard data outside production source", () => {
    expect(existsSync("src/features/flights/sample-data.ts")).toBe(false);
  });
});
