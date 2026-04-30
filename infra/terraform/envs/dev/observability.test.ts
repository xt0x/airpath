import { describe, expect, it } from "vitest";

import {
  bodyIncludes,
  hasAttribute,
  outputBlock,
  readRepoFile,
  readTerraformFile,
  resourceBlock,
  variableBlock,
} from "../../test-support/hcl";

describe("monitoring and operations contract", () => {
  const devMain = readTerraformFile("envs/dev/main.tf");
  const devOutputs = readTerraformFile("envs/dev/outputs.tf");
  const eventingOutputs = readTerraformFile("modules/eventing/outputs.tf");
  const observabilityMain = readTerraformFile("modules/observability/main.tf");
  const observabilityVariables = readTerraformFile("modules/observability/variables.tf");
  const observabilityOutputs = readTerraformFile("modules/observability/outputs.tf");
  const runbook = readRepoFile("services/RUNBOOK.md");

  it("creates a CloudWatch dashboard for FlightAware calls, spend, 429s, and stop state", () => {
    const dashboard = resourceBlock(observabilityMain, "aws_cloudwatch_dashboard", "flightaware");
    expect(bodyIncludes(dashboard, "FlightAwareCallCount")).toBe(true);
    expect(bodyIncludes(dashboard, "EstimatedSpendUSD")).toBe(true);
    expect(bodyIncludes(dashboard, "RateLimitedCount")).toBe(true);
    expect(bodyIncludes(dashboard, "BudgetStopCount")).toBe(true);
    expect(outputBlock(observabilityOutputs, "dashboard_name")).toBeDefined();
    expect(outputBlock(devOutputs, "cloudwatch_dashboard_name")).toBeDefined();
  });

  it("alarms on Lambda errors, Lambda timeouts, DLQ depth, and queue age", () => {
    expect(
      resourceBlock(observabilityMain, "aws_cloudwatch_metric_alarm", "lambda_errors"),
    ).toBeDefined();
    expect(
      hasAttribute(
        resourceBlock(observabilityMain, "aws_cloudwatch_metric_alarm", "lambda_timeouts"),
        "metric_name",
        /"Duration"/,
      ),
    ).toBe(true);
    expect(
      resourceBlock(observabilityMain, "aws_cloudwatch_metric_alarm", "fetch_task_dlq_depth"),
    ).toBeDefined();
    expect(
      hasAttribute(
        resourceBlock(observabilityMain, "aws_cloudwatch_metric_alarm", "fetch_task_queue_age"),
        "metric_name",
        /"ApproximateAgeOfOldestMessage"/,
      ),
    ).toBe(true);
    expect(outputBlock(eventingOutputs, "fetch_task_queue_name")).toBeDefined();
    expect(devMain).toContain(
      "fetch_task_queue_name = module.fetch_task_queue.fetch_task_queue_name",
    );
  });

  it("separates soft-threshold and hard-stop budget alarms", () => {
    expect(variableBlock(observabilityVariables, "budget_soft_threshold_usd")).toBeDefined();
    expect(
      hasAttribute(
        resourceBlock(
          observabilityMain,
          "aws_cloudwatch_metric_alarm",
          "flightaware_budget_soft_threshold",
        ),
        "metric_name",
        /"EstimatedSpendUSD"/,
      ),
    ).toBe(true);
    expect(
      hasAttribute(
        resourceBlock(
          observabilityMain,
          "aws_cloudwatch_metric_alarm",
          "flightaware_budget_hard_stop",
        ),
        "metric_name",
        /"BudgetHardStopCount"/,
      ),
    ).toBe(true);
  });

  it("documents manual stop/resume and operational runbook actions", () => {
    expect(runbook).toContain("Manual Stop And Resume");
    expect(runbook).toContain("FLIGHTAWARE_FETCH_ENABLED=false");
    expect(runbook).toContain("FLIGHTAWARE_FETCH_ENABLED=true");
    expect(runbook).toContain("Budget Exhaustion");
    expect(runbook).toContain("429 Rate Limiting");
    expect(runbook).toContain("Key Rotation");
    expect(runbook).toContain("Stale Cache Behavior");
  });
});
