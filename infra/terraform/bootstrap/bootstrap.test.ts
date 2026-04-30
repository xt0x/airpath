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

describe("Terraform bootstrap contract", () => {
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
    expect(bodyIncludes(statePolicy, 'sid = "ReadTerraformStateObjects"')).toBe(true);
    expect(bodyIncludes(statePolicy, 'sid = "ReadWriteTerraformLockObjects"')).toBe(true);
    expect(bodyIncludes(statePolicy, "for key in local.state_object_keys")).toBe(true);
    expect(bodyIncludes(statePolicy, "for key in local.lock_object_keys")).toBe(true);
  });

  it("defines a GitHub OIDC provider and CI plan role without static AWS keys", () => {
    const oidc = resourceBlock(main, "aws_iam_openid_connect_provider", "github");
    const assumeRolePolicy = dataBlock(
      main,
      "aws_iam_policy_document",
      "github_ci_plan_assume_role",
    );
    expect(resourceBlock(main, "aws_iam_role", "github_ci_plan")).toBeDefined();
    const policy = dataBlock(main, "aws_iam_policy_document", "terraform_plan_read");
    expect(bodyIncludes(oidc, "token.actions.githubusercontent.com")).toBe(true);
    expect(bodyIncludes(assumeRolePolicy, "local.github_oidc_subjects")).toBe(true);
    expect(bodyIncludes(assumeRolePolicy, 'test     = "StringEquals"')).toBe(true);
    expect(bodyIncludes(assumeRolePolicy, "repo:${var.github_repository}:*")).toBe(false);
    expect(bodyIncludes(policy, "sts:GetCallerIdentity")).toBe(true);
    expect(bodyIncludes(policy, "s3:GetBucketPolicy")).toBe(true);
    expect(bodyIncludes(policy, "s3:GetEncryptionConfiguration")).toBe(true);
    expect(bodyIncludes(policy, "s3:GetObject")).toBe(false);
    expect(bodyIncludes(policy, "s3:Get*")).toBe(false);
    expect(bodyIncludes(policy, "s3:ListBucket")).toBe(false);
    expect(bodyIncludes(policy, "s3:ListBucketVersions")).toBe(false);
    expect(bodyIncludes(policy, "s3:ListAllMyBuckets")).toBe(true);
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
    expect(bodyIncludes(variableBlock(variables, "github_oidc_subjects"), "pull_request")).toBe(
      true,
    );
    expect(bodyIncludes(localsBlock(main), "refs/heads/main")).toBe(true);
    expect(outputBlock(outputs, "backend_config")).toBeDefined();
  });
});
