import { describe, expect, it } from "vitest";
import { existsSync, readdirSync } from "node:fs";
import { join } from "node:path";

import frontendPackage from "../../../package.json";
import frontendTsconfig from "../../../tsconfig.json";

describe("flight feature package boundaries", () => {
  it("declares the shared type contract package used by the flight type facade", () => {
    expect(frontendPackage.dependencies).toMatchObject({
      "@airpath/shared-types": "workspace:*",
    });
    expect(frontendPackage.dependencies).not.toHaveProperty("@airpath/map-rendering");
  });

  it("resolves workspace packages through package exports for production builds", () => {
    expect(frontendTsconfig.compilerOptions.paths).not.toHaveProperty("@airpath/shared-types");
    expect(frontendTsconfig.compilerOptions.paths).not.toHaveProperty("@airpath/map-rendering");
    expect(frontendTsconfig.compilerOptions.paths).toHaveProperty("@/*");
  });

  it("builds imported workspace package artifacts before frontend verification", () => {
    expect(frontendPackage.scripts["build:deps"]).toBe("pnpm --filter @airpath/shared-types build");
  });

  it("keeps retired flight source entrypoints out of the public feature boundary", () => {
    expect(existsSync("src/features/flights/api-client.ts")).toBe(false);
    expect(existsSync("src/features/flights/flight-dashboard.tsx")).toBe(false);
    expect(existsSync("src/features/flights/types.ts")).toBe(false);
    expect(cssFilesUnder("src/features/flights")).toEqual([]);
  });

  it("keeps flight source inside documented API, lib, and type facade areas", () => {
    const unexpectedSourceFiles = sourceFilesUnder("src/features/flights").filter((path) => {
      return !/^src\/features\/flights\/(api|lib|types)\//.test(path);
    });

    expect(unexpectedSourceFiles).toEqual([]);
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

function cssFilesUnder(directory: string): string[] {
  if (!existsSync(directory)) {
    return [];
  }

  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = join(directory, entry.name);
    if (entry.isDirectory()) {
      return cssFilesUnder(path);
    }
    return entry.name.endsWith(".css") ? [path] : [];
  });
}
