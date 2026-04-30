import { readdirSync } from "node:fs";
import { join } from "node:path";

import { describe, expect, it } from "vitest";

import {
  bodyIncludes,
  hasAttribute,
  hclBlocks,
  outputBlock,
  readTerraformFile,
  variableBlock,
} from "../test-support/hcl";

interface EnvironmentRootContract {
  readonly name: "stg" | "prod";
  readonly description: string;
}

const roots: EnvironmentRootContract[] = [
  { name: "stg", description: "staging" },
  { name: "prod", description: "production" },
];

describe.each(roots)("$description Terraform root contract", ({ name }) => {
  const rootPath = `envs/${name}`;
  const variables = readTerraformFile(`${rootPath}/variables.tf`);
  const providers = readTerraformFile(`${rootPath}/providers.tf`);
  const backend = readTerraformFile(`${rootPath}/backend.tf`);
  const outputs = readTerraformFile(`${rootPath}/outputs.tf`);

  it(`defaults environment to ${name} and rejects other environment names`, () => {
    const environment = variableBlock(variables, "environment");

    expect(hasAttribute(environment, "default", new RegExp(`"${name}"`))).toBe(true);
    expect(bodyIncludes(environment, `contains(["${name}"], var.environment)`)).toBe(true);
    expect(bodyIncludes(environment, `This directory only manages the ${name} environment.`)).toBe(
      true,
    );
  });

  it("keeps the provider shape scoped to the root region variable", () => {
    const awsRegion = variableBlock(variables, "aws_region");

    expect(hasAttribute(awsRegion, "default", /"ap-northeast-1"/)).toBe(true);
    expect(providers).toContain('required_version = ">= 1.10.0, < 2.0.0"');
    expect(providers).toContain('source  = "hashicorp/aws"');
    expect(providers).toContain('version = "~> 5.0"');
    expect(providers).toContain("region = var.aws_region");
  });

  it("keeps backend wiring as an explicit bootstrap placeholder", () => {
    expect(backend).toContain("Backend is wired after L6 bootstrap creates state resources.");
    expect(backend).not.toMatch(/\bbackend\s+"/);
  });

  it("exports only the managed environment name", () => {
    const outputsSource = outputs.trim();
    const outputNames = hclBlocks(outputs, "output").map((block) => block.labels[0]);

    expect(outputNames).toEqual(["environment"]);
    expect(
      bodyIncludes(outputBlock(outputsSource, "environment"), "value       = var.environment"),
    ).toBe(true);
  });

  it("does not declare deployable infrastructure yet", () => {
    const source = readdirSync(join("infra/terraform", rootPath))
      .filter((file) => file.endsWith(".tf"))
      .map((file) => readTerraformFile(`${rootPath}/${file}`))
      .join("\n");

    expect(hclBlocks(source, "resource")).toEqual([]);
    expect(hclBlocks(source, "module")).toEqual([]);
    expect(hclBlocks(source, "data")).toEqual([]);
  });
});
