import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

const readRepoFile = (path: string): string => readFileSync(path, "utf8");
const readTerraformFile = (path: string): string => readRepoFile(join("infra/terraform", path));

describe("F17 personal demo deployment contract", () => {
  const devMain = readTerraformFile("envs/dev/main.tf");
  const devVariables = readTerraformFile("envs/dev/variables.tf");
  const devTfvarsExample = readTerraformFile("envs/dev/terraform.tfvars.example");
  const makefile = readRepoFile("Makefile");
  const rootReadme = readRepoFile("README.md");
  const terraformReadme = readRepoFile("infra/terraform/README.md");

  it("keeps dev real FlightAware calls disabled unless an explicit opt-in flag is set", () => {
    expect(devVariables).toContain('variable "allow_real_flightaware_calls"');
    expect(devVariables).toContain("default     = false");
    expect(devMain).toMatch(
      /flightaware_real_calls_enabled\s*=\s*var\.allow_real_flightaware_calls/,
    );
    expect(devMain).toContain("FLIGHTAWARE_REAL_CALLS_ENABLED");
    expect(devMain).toContain("FLIGHTAWARE_FETCH_ENABLED");
    expect(devMain).toContain('var.allow_real_flightaware_calls ? "true" : "false"');
    expect(devMain).toContain("FLIGHTAWARE_FETCH_DISABLED_REASON");
    expect(devTfvarsExample).toMatch(/allow_real_flightaware_calls\s*=\s*false/);
  });

  it("marks dev as a personal non-commercial low-frequency demo in runtime config", () => {
    expect(devMain).toContain("AIRPATH_PERSONAL_DEMO_NOTICE");
    expect(devVariables).toContain("personal non-commercial low-frequency");
    expect(devMain).toMatch(
      /FETCHER_MODE\s*=\s*var\.allow_real_flightaware_calls \? "real-opt-in" : "mock"/,
    );
    expect(devVariables).toContain('variable "personal_demo_notice"');
  });

  it("documents artifact deployment and personal-use acceptance runs", () => {
    expect(rootReadme).toContain("Personal Demo Environment");
    expect(rootReadme).toContain("personal, non-commercial, low-frequency");
    expect(makefile).toContain("lambda-artifacts:");
    expect(makefile).toContain("api-lambda.zip");
    expect(rootReadme).toContain("make lambda-artifacts");
    expect(rootReadme).toContain("AIRPATH_FLIGHTAWARE_REAL_TESTS=true");
    expect(rootReadme).toContain("limited real-call smoke tests");
    expect(terraformReadme).toContain("allow_real_flightaware_calls");
    expect(terraformReadme).toContain("disabled by default");
  });
});
