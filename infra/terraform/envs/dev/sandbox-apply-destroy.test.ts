import { execFileSync } from "node:child_process";
import { existsSync, mkdtempSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";

import { describe, expect, it } from "vitest";

import { readTerraformFile } from "../../test-support/hcl";

const terraform = process.env.TERRAFORM ?? "terraform";
const sandboxEnabled = process.env.AIRPATH_TERRAFORM_SANDBOX_APPLY_DESTROY === "1";
const devRoot = resolve("infra/terraform/envs/dev");
const terraformTest = sandboxEnabled ? it : it.skip;

type TerraformOutput<T> = {
  sensitive: boolean;
  type: unknown;
  value: T;
};

type DevOutputs = {
  api_lambda_function_name: TerraformOutput<string>;
  cloudwatch_alarm_names: TerraformOutput<Record<string, string>>;
  cloudwatch_dashboard_name: TerraformOutput<string>;
  dispatcher_lambda_function_name: TerraformOutput<string>;
  dynamodb_table_names: TerraformOutput<Record<string, string>>;
  environment: TerraformOutput<string>;
  fetch_task_queue_url: TerraformOutput<string>;
  fetcher_lambda_function_name: TerraformOutput<string>;
  flightaware_api_key_secret_arn: TerraformOutput<string>;
  geojson_bucket_name: TerraformOutput<string>;
  http_api_endpoint: TerraformOutput<string>;
};

describe("sandbox AWS apply/destroy integration", () => {
  const terraformReadme = readFileSync("infra/terraform/README.md", "utf8");
  const terraformDesign = readFileSync("infra/terraform/DESIGN.md", "utf8");
  const devVariables = readTerraformFile("envs/dev/variables.tf");

  it("documents the manual sandbox workflow and keeps it opt-in", () => {
    expect(terraformReadme).toContain("Sandbox Apply/Destroy Integration Test");
    expect(terraformReadme).toContain("AIRPATH_TERRAFORM_SANDBOX_APPLY_DESTROY=1");
    expect(terraformReadme).toContain("make lambda-artifacts");
    expect(terraformReadme).toContain("terraform apply");
    expect(terraformReadme).toContain("terraform state list");
    expect(terraformReadme).toContain("terraform destroy");
    expect(terraformDesign).toContain("sandbox apply/destroy integration test");
    expect(terraformDesign).toContain("manual, nightly, or release-before-deploy");
  });

  it("keeps the sandbox workflow on Terraform metadata and secret references only", () => {
    expect(devVariables).toContain("flightaware_api_key_secret_name");
    expect(terraformReadme).toContain("Do not provide raw API keys through Terraform variables");
    expect(terraformReadme).not.toMatch(/flightaware_api_key_value|api_key_value|secret_string/i);
  });

  terraformTest(
    "applies dev to sandbox AWS, checks safe outputs, and destroys it",
    () => {
      const workDir = mkdtempSync(join(tmpdir(), "airpath-terraform-sandbox-"));
      const statePath = join(workDir, "terraform.tfstate");
      const dataDir = join(workDir, ".terraform");
      const tfvarsPath = process.env.AIRPATH_TERRAFORM_SANDBOX_TFVARS;
      const terraformEnv = {
        ...process.env,
        TF_DATA_DIR: dataDir,
      };

      const terraformArgs = (args: string[]): string[] => [
        "-chdir=infra/terraform/envs/dev",
        ...args,
      ];
      const terraformApplyArgs = (args: string[]): string[] => {
        const withVarFile =
          tfvarsPath === undefined || tfvarsPath.trim() === ""
            ? args
            : [...args, `-var-file=${resolve(tfvarsPath)}`];
        return ["-chdir=infra/terraform/envs/dev", ...withVarFile];
      };

      try {
        for (const artifactPath of [
          "../../../artifacts/dev/api-lambda.zip",
          "../../../artifacts/dev/fetcher-lambda.zip",
          "../../../artifacts/dev/dispatcher-lambda.zip",
        ]) {
          expect(existsSync(resolve(devRoot, artifactPath)), `${artifactPath} exists`).toBe(true);
        }

        execFileSync(terraform, terraformArgs(["init", "-backend=false", "-input=false"]), {
          env: terraformEnv,
          stdio: "inherit",
        });
        execFileSync(
          terraform,
          terraformApplyArgs(["apply", "-auto-approve", "-input=false", `-state=${statePath}`]),
          {
            env: terraformEnv,
            stdio: "inherit",
          },
        );

        const outputs = JSON.parse(
          execFileSync(terraform, terraformArgs(["output", "-json", `-state=${statePath}`]), {
            encoding: "utf8",
            env: terraformEnv,
          }),
        ) as DevOutputs;

        expect(outputs.environment.value).toBe("dev");
        expect(outputs.http_api_endpoint.value).toMatch(/^https:\/\//);
        expect(outputs.api_lambda_function_name.value).toMatch(/^airpath-dev-/);
        expect(outputs.fetcher_lambda_function_name.value).toMatch(/^airpath-dev-/);
        expect(outputs.dispatcher_lambda_function_name.value).toMatch(/^airpath-dev-/);
        expect(outputs.fetch_task_queue_url.value).toContain("airpath-dev");
        expect(outputs.dynamodb_table_names.value.flights).toMatch(/^airpath-dev-/);
        expect(outputs.geojson_bucket_name.value).toMatch(/^airpath-dev-/);
        expect(outputs.cloudwatch_dashboard_name.value).toMatch(/^airpath-dev-/);
        expect(Object.keys(outputs.cloudwatch_alarm_names.value).length).toBeGreaterThan(0);
        expect(outputs.flightaware_api_key_secret_arn.value).toMatch(/^arn:aws:secretsmanager:/);

        const stateAddresses = execFileSync(
          terraform,
          terraformArgs(["state", "list", `-state=${statePath}`]),
          {
            encoding: "utf8",
            env: terraformEnv,
          },
        );

        for (const resourceType of [
          "aws_lambda_function",
          "aws_apigatewayv2_api",
          "aws_sqs_queue",
          "aws_dynamodb_table",
          "aws_s3_bucket",
          "aws_cloudwatch_dashboard",
          "aws_cloudwatch_metric_alarm",
          "aws_secretsmanager_secret",
        ]) {
          expect(stateAddresses, `${resourceType} exists in Terraform state`).toContain(
            resourceType,
          );
        }
      } finally {
        if (existsSync(statePath)) {
          execFileSync(
            terraform,
            terraformApplyArgs(["destroy", "-auto-approve", "-input=false", `-state=${statePath}`]),
            {
              env: terraformEnv,
              stdio: "inherit",
            },
          );
        }
        rmSync(workDir, { force: true, recursive: true });
      }
    },
    45 * 60 * 1000,
  );
});
