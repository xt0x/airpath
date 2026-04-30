import { describe, expect, it } from "vitest";

import { bodyIncludes, readTerraformFile, resourceBlock } from "../../test-support/hcl";

describe("compute-lambda module HCL contract", () => {
  it("creates Lambda functions after log group and IAM policy attachments", () => {
    const computeLambda = readTerraformFile("modules/compute-lambda/main.tf");
    const lambdaFunction = resourceBlock(computeLambda, "aws_lambda_function", "this");

    expect(bodyIncludes(lambdaFunction, "aws_cloudwatch_log_group.this")).toBe(true);
    expect(bodyIncludes(lambdaFunction, "aws_iam_role_policy_attachment.basic_execution")).toBe(
      true,
    );
    expect(bodyIncludes(lambdaFunction, "aws_iam_role_policy.inline")).toBe(true);
  });
});
