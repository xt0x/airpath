import { describe, expect, it } from "vitest";

import { bodyIncludes, hasAttribute, resourceBlock } from "./hcl";

describe("HCL test support", () => {
  it("keeps heredoc braces inside the owning block body", () => {
    const source = String.raw`
resource "aws_iam_policy" "example" {
  policy = <<JSON
{
  "Version": "2012-10-17",
  "Statement": [{ "Action": "s3:GetObject", "Resource": "*" }]
}
JSON
  name = "example-policy"
}
`;

    const block = resourceBlock(source, "aws_iam_policy", "example");

    expect(hasAttribute(block, "name", /"example-policy"/)).toBe(true);
    expect(bodyIncludes(block, '"Action": "s3:GetObject"')).toBe(true);
  });

  it("matches multiline attribute values without relying on line-local regexes", () => {
    const source = `
resource "aws_iam_role_policy" "example" {
  actions = [
    "sqs:SendMessage",
    "sqs:GetQueueAttributes",
  ]
}
`;

    const block = resourceBlock(source, "aws_iam_role_policy", "example");

    expect(hasAttribute(block, "actions", /sqs:GetQueueAttributes/)).toBe(true);
  });
});
