import { describe, expect, expectTypeOf, it } from "vitest";
import { existsSync, readdirSync, readFileSync } from "node:fs";
import { join } from "node:path";

import { AirpathApiClient } from "@/features/flights/api/api-client";
import type { FlightDetail, RefreshTaskType } from "@/features/flights/types";
// @ts-expect-error internal fetch queue task payloads must stay out of the frontend facade.
import type { FetchTask, FetchTaskType } from "@/features/flights/types"; // eslint-disable-line @typescript-eslint/no-unused-vars

describe("flight feature type boundary", () => {
  it("exports flight detail through the frontend flight type facade", () => {
    expectTypeOf<FlightDetail>().toHaveProperty("flightId").toEqualTypeOf<string>();
  });

  it("exposes refresh-only task types for browser refresh requests", () => {
    type RefreshTaskArgument = Parameters<AirpathApiClient["requestFlightRefresh"]>[1];

    expectTypeOf<RefreshTaskArgument>().toEqualTypeOf<readonly RefreshTaskType[]>();
  });

  it("keeps summary fetch tasks out of the browser refresh contract", () => {
    const refreshTasks = [
      "position",
      "route",
      "track",
      "final_track",
    ] as const satisfies readonly RefreshTaskType[];

    // @ts-expect-error summary tasks seed fetch work internally but are not valid manual refresh inputs.
    const invalidRefreshTasks = ["summary"] as const satisfies readonly RefreshTaskType[];

    expect(refreshTasks).toEqual(["position", "route", "track", "final_track"]);
    void invalidRefreshTasks;
  });

  it("keeps shared flight API type imports behind the frontend facade", () => {
    const directSharedTypeImports = sourceFilesUnder("src")
      .map((path) => ({
        path,
        source: readFileSync(path, "utf8"),
      }))
      .filter(({ path, source }) => {
        if (path === "src/features/flights/types/index.ts") {
          return false;
        }
        return /import\s+type\s+\{[^}]+}\s+from\s+["']@airpath\/shared-types["']/.test(source);
      })
      .map(({ path }) => path);

    expect(directSharedTypeImports).toEqual([]);
  });
});

function sourceFilesUnder(directory: string): string[] {
  if (!existsSync(directory)) {
    return [];
  }

  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = join(directory, entry.name);
    if (entry.isDirectory()) {
      return sourceFilesUnder(path);
    }
    return /\.tsx?$/.test(entry.name) ? [path] : [];
  });
}
