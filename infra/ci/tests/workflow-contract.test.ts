import { describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";
import { join } from "node:path";

const repositoryRoot = join(import.meta.dirname, "..", "..", "..");

const readWorkflow = (name: string): string =>
  readFileSync(join(repositoryRoot, ".github", "workflows", name), "utf8");

describe("GitHub Actions security workflow contract", () => {
  it("keeps Gitleaks default rules enabled", () => {
    const gitleaksConfig = readFileSync(join(repositoryRoot, ".gitleaks.toml"), "utf8");

    expect(gitleaksConfig).toContain("[extend]");
    expect(gitleaksConfig).toContain("useDefault = true");
  });

  it("is included in the top-level CI gate", () => {
    const ciWorkflow = readWorkflow("ci.yml");

    expect(ciWorkflow).toContain("security:");
    expect(ciWorkflow).toContain("uses: ./.github/workflows/ci-security.yml");
  });

  it("runs Gitleaks against full Git history with redacted output", () => {
    const securityWorkflow = readWorkflow("ci-security.yml");

    expect(securityWorkflow).toContain("fetch-depth: 0");
    expect(securityWorkflow).toMatch(/ghcr\.io\/gitleaks\/gitleaks:v\d+\.\d+\.\d+/);
    expect(securityWorkflow).toContain("git --redact");
    expect(securityWorkflow).toContain("--redact");
  });
});
