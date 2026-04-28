import { describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";
import { join } from "node:path";

const readTerraformFile = (path: string): string =>
  readFileSync(join("infra/terraform", path), "utf8");

describe("F7-01..04 dev infrastructure contract", () => {
  const devMain = readTerraformFile("envs/dev/main.tf");
  const devVariables = readTerraformFile("envs/dev/variables.tf");
  const devOutputs = readTerraformFile("envs/dev/outputs.tf");
  const httpApiMain = readTerraformFile("modules/api-http/main.tf");
  const lambdaMain = readTerraformFile("modules/compute-lambda/main.tf");
  const eventingMain = readTerraformFile("modules/eventing/main.tf");

  it("routes /v1/* HTTP API traffic to the Go API Lambda integration", () => {
    expect(devMain).toContain('module "http_api"');
    expect(httpApiMain).toContain('resource "aws_apigatewayv2_api" "this"');
    expect(httpApiMain).toContain('route_key = "ANY /v1/{proxy+}"');
    expect(httpApiMain).toContain("AWS_PROXY");
    expect(httpApiMain).toContain("aws_lambda_permission");
    expect(devOutputs).toContain("http_api_endpoint");
  });

  it("defines deployable API, fetcher, and dispatcher Lambda shells from artifact references", () => {
    expect(devMain).toContain('module "api_lambda"');
    expect(devMain).toContain('module "fetcher_lambda"');
    expect(devMain).toContain('module "dispatcher_lambda"');
    expect(devVariables).toContain("api_lambda_artifact_path");
    expect(devVariables).toContain("fetcher_lambda_artifact_path");
    expect(devVariables).toContain("dispatcher_lambda_artifact_path");
    expect(lambdaMain).toContain('resource "aws_lambda_function" "this"');
    expect(lambdaMain).toContain("filename         = var.artifact_path");
  });

  it("wires SQS-triggered fetcher execution for mock fetch tasks", () => {
    expect(devMain).toContain('module "fetch_task_queue"');
    expect(eventingMain).toContain('resource "aws_sqs_queue" "fetch_task"');
    expect(eventingMain).toContain('resource "aws_lambda_event_source_mapping" "fetcher"');
    expect(eventingMain).toContain("event_source_arn = aws_sqs_queue.fetch_task.arn");
    expect(eventingMain).toContain("function_name    = var.fetcher_lambda_arn");
  });

  it("wires an EventBridge dispatcher schedule that can enqueue no-op tasks", () => {
    expect(eventingMain).toContain('resource "aws_cloudwatch_event_rule" "dispatcher"');
    expect(eventingMain).toContain('resource "aws_cloudwatch_event_target" "dispatcher"');
    expect(eventingMain).toContain("var.dispatcher_schedule_expression");
    expect(devMain).toContain("NOOP_FETCH_ENABLED");
    expect(devMain).toContain("FETCH_TASK_QUEUE_URL");
  });
});
