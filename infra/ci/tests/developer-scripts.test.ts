import { describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";
import { join } from "node:path";

const repositoryRoot = join(import.meta.dirname, "..", "..", "..");

const readJson = <T>(path: string): T => JSON.parse(readFileSync(path, "utf8")) as T;

describe("developer script contract", () => {
  it("exposes a root dev script for the frontend workspace app", () => {
    const rootPackageJson = readJson<{ scripts: Record<string, string> }>(
      join(repositoryRoot, "package.json"),
    );

    expect(rootPackageJson.scripts.dev).toBe("pnpm --filter @airpath/web dev");
  });

  it("documents the root frontend dev command", () => {
    const readme = readFileSync(join(repositoryRoot, "README.md"), "utf8");

    expect(readme).toContain("pnpm dev");
    expect(readme).toMatch(/Do not run `npx next dev` from the\s+repository root/);
  });
});
