import { describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";
import { join } from "node:path";

const bootstrapDir = "infra/terraform/bootstrap";

const readBootstrapFile = (fileName: string): string =>
  readFileSync(join(bootstrapDir, fileName), "utf8");

describe("F6 Terraform bootstrap contract", () => {
  const main = readBootstrapFile("main.tf");
  const variables = readBootstrapFile("variables.tf");
  const outputs = readBootstrapFile("outputs.tf");

  it("creates an encrypted versioned S3 backend bucket with native lockfile access", () => {
    expect(main).toContain('resource "aws_s3_bucket" "terraform_state"');
    expect(main).toContain('resource "aws_s3_bucket_versioning" "terraform_state"');
    expect(main).toContain(
      'resource "aws_s3_bucket_server_side_encryption_configuration" "terraform_state"',
    );
    expect(main).toContain('resource "aws_s3_bucket_public_access_block" "terraform_state"');
    expect(main).toContain(".tflock");
  });

  it("defines a KMS key and least-privilege state access policy", () => {
    expect(main).toContain('resource "aws_kms_key" "terraform_state"');
    expect(main).toContain("kms:Decrypt");
    expect(main).toContain("kms:GenerateDataKey");
    expect(main).toContain("s3:GetObject");
    expect(main).toContain("s3:PutObject");
    expect(main).toContain("s3:DeleteObject");
  });

  it("defines a GitHub OIDC provider and CI plan role without static AWS keys", () => {
    expect(main).toContain('resource "aws_iam_openid_connect_provider" "github"');
    expect(main).toContain('resource "aws_iam_role" "github_ci_plan"');
    expect(main).toContain('resource "aws_iam_policy" "terraform_plan_read"');
    expect(main).toContain("token.actions.githubusercontent.com");
    expect(main).toContain("sts:AssumeRoleWithWebIdentity");
    expect(main).toContain("sts:GetCallerIdentity");
    expect(main).not.toMatch(/aws_access_key_id|aws_secret_access_key/i);
  });

  it("preserves environment naming for local, dev, stg, and prod", () => {
    expect(variables).toContain(
      'contains(["local", "dev", "stg", "prod"], var.bootstrap_environment)',
    );
    expect(variables).toContain('regex("^[^/]+/[^/]+$", var.github_repository)');
    expect(outputs).toContain("backend_config");
  });
});
