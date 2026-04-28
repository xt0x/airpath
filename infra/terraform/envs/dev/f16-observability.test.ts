import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

const readRepoFile = (path: string): string => readFileSync(path, "utf8");
const readTerraformFile = (path: string): string => readRepoFile(join("infra/terraform", path));

describe("F16 monitoring and operations contract", () => {
  const devMain = readTerraformFile("envs/dev/main.tf");
  const devOutputs = readTerraformFile("envs/dev/outputs.tf");
  const eventingOutputs = readTerraformFile("modules/eventing/outputs.tf");
  const observabilityMain = readTerraformFile("modules/observability/main.tf");
  const observabilityVariables = readTerraformFile("modules/observability/variables.tf");
  const observabilityOutputs = readTerraformFile("modules/observability/outputs.tf");
  const runbook = readRepoFile("services/RUNBOOK.md");

  it("creates a CloudWatch dashboard for FlightAware calls, spend, 429s, and stop state", () => {
    expect(observabilityMain).toContain('resource "aws_cloudwatch_dashboard" "flightaware"');
    expect(observabilityMain).toContain("FlightAwareCallCount");
    expect(observabilityMain).toContain("EstimatedSpendUSD");
    expect(observabilityMain).toContain("RateLimitedCount");
    expect(observabilityMain).toContain("BudgetStopCount");
    expect(observabilityOutputs).toContain("dashboard_name");
    expect(devOutputs).toContain("cloudwatch_dashboard_name");
  });

  it("alarms on Lambda errors, Lambda timeouts, DLQ depth, and queue age", () => {
    expect(observabilityMain).toContain('resource "aws_cloudwatch_metric_alarm" "lambda_errors"');
    expect(observabilityMain).toContain('resource "aws_cloudwatch_metric_alarm" "lambda_timeouts"');
    expect(observabilityMain).toContain('metric_name         = "Duration"');
    expect(observabilityMain).toContain(
      'resource "aws_cloudwatch_metric_alarm" "fetch_task_dlq_depth"',
    );
    expect(observabilityMain).toContain(
      'resource "aws_cloudwatch_metric_alarm" "fetch_task_queue_age"',
    );
    expect(observabilityMain).toContain('metric_name         = "ApproximateAgeOfOldestMessage"');
    expect(eventingOutputs).toContain("fetch_task_queue_name");
    expect(devMain).toContain(
      "fetch_task_queue_name = module.fetch_task_queue.fetch_task_queue_name",
    );
  });

  it("separates soft-threshold and hard-stop budget alarms", () => {
    expect(observabilityVariables).toContain("budget_soft_threshold_usd");
    expect(observabilityMain).toContain(
      'resource "aws_cloudwatch_metric_alarm" "flightaware_budget_soft_threshold"',
    );
    expect(observabilityMain).toContain('metric_name         = "EstimatedSpendUSD"');
    expect(observabilityMain).toContain(
      'resource "aws_cloudwatch_metric_alarm" "flightaware_budget_hard_stop"',
    );
    expect(observabilityMain).toContain('metric_name         = "BudgetHardStopCount"');
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
