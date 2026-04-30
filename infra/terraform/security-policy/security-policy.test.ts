import { readdirSync, statSync } from "node:fs";
import { join } from "node:path";

import { describe, expect, it } from "vitest";

import {
  bodyIncludes,
  hclBlocks,
  readRepoFile,
  readTerraformFile,
  resourceBlock,
} from "../test-support/hcl";

const terraformRoot = "infra/terraform";

function terraformFilesUnder(path: string): string[] {
  const entries = readdirSync(path).sort();
  return entries.flatMap((entry) => {
    const fullPath = join(path, entry);
    const stat = statSync(fullPath);

    if (stat.isDirectory()) {
      if (entry === ".terraform") {
        return [];
      }
      return terraformFilesUnder(fullPath);
    }

    return entry.endsWith(".tf") ? [fullPath] : [];
  });
}

describe("Terraform security policy scanner contract", () => {
  it("wires a Checkov policy scan into the Terraform check path", () => {
    const makefile = readRepoFile("Makefile");
    const workflow = readRepoFile(".github/workflows/ci-terraform.yml");
    const checkovConfig = readRepoFile(".checkov.yml");

    expect(makefile).toContain("terraform-policy:");
    expect(makefile).toContain("bash scripts/ci/terraform-policy.sh");
    expect(makefile).toMatch(/terraform-check:[\s\S]*\$\(MAKE\) terraform-policy/);
    expect(workflow).toContain("astral-sh/setup-uv");
    expect(workflow).toContain("make terraform-policy");
    expect(checkovConfig).toContain("CKV_AWS_19");
    expect(checkovConfig).toContain("CKV_AWS_27");
    expect(checkovConfig).toContain("CKV2_AWS_6");
  });

  it("encrypts S3 buckets and SQS queues that hold Terraform-managed application data", () => {
    const storage = readTerraformFile("modules/storage-s3/main.tf");
    const eventing = readTerraformFile("modules/eventing/main.tf");

    expect(
      resourceBlock(storage, "aws_s3_bucket_server_side_encryption_configuration", "geojson"),
    ).toBeDefined();
    expect(
      bodyIncludes(
        resourceBlock(eventing, "aws_sqs_queue", "fetch_task"),
        "sqs_managed_sse_enabled = true",
      ),
    ).toBe(true);
    expect(
      bodyIncludes(
        resourceBlock(eventing, "aws_sqs_queue", "fetch_task_dlq"),
        "sqs_managed_sse_enabled = true",
      ),
    ).toBe(true);
  });

  it("keeps Terraform from managing raw Secrets Manager values", () => {
    const allTerraform = terraformFilesUnder(terraformRoot)
      .map((path) => readRepoFile(path))
      .join("\n");

    expect(
      hclBlocks(allTerraform, "resource").filter(
        (block) => block.labels[0] === "aws_secretsmanager_secret_version",
      ),
    ).toEqual([]);
  });

  it("limits wildcard IAM resources to bootstrap administration and read-only plan discovery", () => {
    const wildcardPolicyFiles = terraformFilesUnder(terraformRoot).flatMap((path) => {
      const source = readRepoFile(path);
      return hclBlocks(source, "data")
        .filter((block) => block.labels[0] === "aws_iam_policy_document")
        .filter((block) => bodyIncludes(block, 'resources = ["*"]'))
        .map((block) => `${path}:${block.labels.join(".")}`);
    });

    expect(wildcardPolicyFiles).toEqual([
      "infra/terraform/bootstrap/main.tf:aws_iam_policy_document.terraform_state_key",
      "infra/terraform/bootstrap/main.tf:aws_iam_policy_document.terraform_plan_read",
    ]);
  });
});
