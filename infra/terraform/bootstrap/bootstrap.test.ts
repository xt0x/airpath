import { describe, expect, it } from "vitest";

import {
  bodyIncludes,
  dataBlock,
  localsBlock,
  outputBlock,
  readTerraformFile,
  resourceBlock,
  variableBlock,
} from "../test-support/hcl";

describe("F6 Terraform bootstrap contract", () => {
  const main = readTerraformFile("bootstrap/main.tf");
  const variables = readTerraformFile("bootstrap/variables.tf");
  const outputs = readTerraformFile("bootstrap/outputs.tf");

  it("creates an encrypted versioned S3 backend bucket with native lockfile access", () => {
    expect(resourceBlock(main, "aws_s3_bucket", "terraform_state")).toBeDefined();
    expect(resourceBlock(main, "aws_s3_bucket_versioning", "terraform_state")).toBeDefined();
    expect(
      resourceBlock(main, "aws_s3_bucket_server_side_encryption_configuration", "terraform_state"),
    ).toBeDefined();
    expect(
      resourceBlock(main, "aws_s3_bucket_public_access_block", "terraform_state"),
    ).toBeDefined();
    expect(bodyIncludes(localsBlock(main), ".tflock")).toBe(true);
    expect(
      bodyIncludes(
        dataBlock(main, "aws_iam_policy_document", "terraform_state_access"),
        "local.lock_object_keys",
      ),
    ).toBe(true);
  });

  it("defines a KMS key and least-privilege state access policy", () => {
    expect(resourceBlock(main, "aws_kms_key", "terraform_state")).toBeDefined();
    const statePolicy = dataBlock(main, "aws_iam_policy_document", "terraform_state_access");
    const keyPolicy = dataBlock(main, "aws_iam_policy_document", "terraform_state_key");
    expect(bodyIncludes(keyPolicy, "kms:Decrypt")).toBe(true);
    expect(bodyIncludes(keyPolicy, "kms:GenerateDataKey")).toBe(true);
    expect(bodyIncludes(statePolicy, "s3:GetObject")).toBe(true);
    expect(bodyIncludes(statePolicy, "s3:PutObject")).toBe(true);
    expect(bodyIncludes(statePolicy, "s3:DeleteObject")).toBe(true);
  });

  it("defines a GitHub OIDC provider and CI plan role without static AWS keys", () => {
    const oidc = resourceBlock(main, "aws_iam_openid_connect_provider", "github");
    expect(resourceBlock(main, "aws_iam_role", "github_ci_plan")).toBeDefined();
    const policy = dataBlock(main, "aws_iam_policy_document", "terraform_plan_read");
    expect(bodyIncludes(oidc, "token.actions.githubusercontent.com")).toBe(true);
    expect(bodyIncludes(policy, "sts:GetCallerIdentity")).toBe(true);
    expect(main).toContain("sts:AssumeRoleWithWebIdentity");
    expect(main).not.toMatch(/aws_access_key_id|aws_secret_access_key/i);
  });

  it("preserves environment naming for local, dev, stg, and prod", () => {
    expect(
      bodyIncludes(
        variableBlock(variables, "bootstrap_environment"),
        '["local", "dev", "stg", "prod"]',
      ),
    ).toBe(true);
    expect(
      bodyIncludes(variableBlock(variables, "github_repository"), 'regex("^[^/]+/[^/]+$"'),
    ).toBe(true);
    expect(outputBlock(outputs, "backend_config")).toBeDefined();
  });
});
