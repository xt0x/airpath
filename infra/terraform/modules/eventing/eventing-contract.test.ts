import { describe, expect, it } from "vitest";

import { bodyIncludes, readTerraformFile, resourceBlock } from "../../test-support/hcl";

describe("eventing module HCL contract", () => {
  it("creates the fetcher event source mapping after SQS consume permissions", () => {
    const eventing = readTerraformFile("modules/eventing/main.tf");
    const mapping = resourceBlock(eventing, "aws_lambda_event_source_mapping", "fetcher");

    expect(bodyIncludes(mapping, "depends_on = [aws_iam_role_policy.fetcher_queue_consume]")).toBe(
      true,
    );
  });
});
