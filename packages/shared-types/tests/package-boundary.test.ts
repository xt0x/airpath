import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

describe("shared-types package boundaries", () => {
  it("keeps the public index as a barrel instead of the domain type implementation", () => {
    const indexSource = readFileSync("src/index.ts", "utf8");
    const apiTypesSource = readFileSync("src/api-types.ts", "utf8");

    expect(indexSource).not.toMatch(/^export\s+(interface|type)\s+(Flight|Airport|FlightTimes)\b/m);
    expect(indexSource).toContain('from "./domain-types.js"');
    expect(apiTypesSource).not.toContain('from "./index.js"');
    expect(apiTypesSource).toContain('from "./domain-types.js"');
  });

  it("keeps shared contract fixtures inside the package boundary and type checks them", () => {
    const packageJson = JSON.parse(readFileSync("package.json", "utf8")) as {
      files: string[];
      scripts: Record<string, string>;
    };
    const fixtureTsconfig = readFileSync("tsconfig.fixtures.json", "utf8");
    const testTsconfig = readFileSync("tsconfig.test.json", "utf8");

    expect(packageJson.files).toContain("fixtures");
    expect(packageJson.scripts.typecheck).toContain("tsconfig.test.json");
    expect(packageJson.scripts.typecheck).toContain("tsconfig.fixtures.json");
    expect(testTsconfig).toContain('"test/**/*.ts"');
    expect(fixtureTsconfig).toContain('"fixtures/**/*.ts"');
  });

  it("keeps FNV hashing in one TypeScript helper", () => {
    const idGenerationSource = readFileSync("src/id-generation.ts", "utf8");
    const eventDedupeSource = readFileSync("src/event-dedupe.ts", "utf8");

    expect(idGenerationSource).toContain('from "./stable-hash.js"');
    expect(eventDedupeSource).toContain('from "./stable-hash.js"');
    expect(idGenerationSource + eventDedupeSource).not.toContain("0xcbf29ce484222325n");
  });
});
