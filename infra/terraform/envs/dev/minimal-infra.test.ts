import { describe, expect, it } from "vitest";

import {
  bodyIncludes,
  hasAttribute,
  moduleBlock,
  outputBlock,
  readTerraformFile,
  resourceBlock,
  variableBlock,
} from "../../test-support/hcl";

describe("dev infrastructure contract", () => {
  const devMain = readTerraformFile("envs/dev/main.tf");
  const devVariables = readTerraformFile("envs/dev/variables.tf");
  const devOutputs = readTerraformFile("envs/dev/outputs.tf");
  const httpApiMain = readTerraformFile("modules/api-http/main.tf");
  const lambdaMain = readTerraformFile("modules/compute-lambda/main.tf");
  const eventingMain = readTerraformFile("modules/eventing/main.tf");
  const dynamodbMain = readTerraformFile("modules/data-dynamodb/main.tf");
  const storageMain = readTerraformFile("modules/storage-s3/main.tf");
  const secretsMain = readTerraformFile("modules/secrets/main.tf");
  const observabilityMain = readTerraformFile("modules/observability/main.tf");

  it("routes /v1/* HTTP API traffic to the Go API Lambda integration", () => {
    expect(moduleBlock(devMain, "http_api")).toBeDefined();
    expect(resourceBlock(httpApiMain, "aws_apigatewayv2_api", "this")).toBeDefined();
    expect(
      hasAttribute(
        resourceBlock(httpApiMain, "aws_apigatewayv2_route", "v1_proxy"),
        "route_key",
        /"ANY \/v1\/\{proxy\+\}"/,
      ),
    ).toBe(true);
    expect(
      hasAttribute(
        resourceBlock(httpApiMain, "aws_apigatewayv2_integration", "api_lambda"),
        "integration_type",
        /"AWS_PROXY"/,
      ),
    ).toBe(true);
    expect(resourceBlock(httpApiMain, "aws_lambda_permission", "allow_http_api")).toBeDefined();
    expect(outputBlock(devOutputs, "http_api_endpoint")).toBeDefined();
  });

  it("defines deployable API, fetcher, and dispatcher Lambda artifacts from artifact references", () => {
    expect(moduleBlock(devMain, "api_lambda")).toBeDefined();
    expect(moduleBlock(devMain, "fetcher_lambda")).toBeDefined();
    expect(moduleBlock(devMain, "dispatcher_lambda")).toBeDefined();
    expect(variableBlock(devVariables, "api_lambda_artifact_path")).toBeDefined();
    expect(variableBlock(devVariables, "fetcher_lambda_artifact_path")).toBeDefined();
    expect(variableBlock(devVariables, "dispatcher_lambda_artifact_path")).toBeDefined();
    expect(
      hasAttribute(
        resourceBlock(lambdaMain, "aws_lambda_function", "this"),
        "filename",
        /var\.artifact_path/,
      ),
    ).toBe(true);
  });

  it("wires SQS-triggered fetcher execution for mock fetch tasks", () => {
    expect(moduleBlock(devMain, "fetch_task_queue")).toBeDefined();
    expect(resourceBlock(eventingMain, "aws_sqs_queue", "fetch_task")).toBeDefined();
    const mapping = resourceBlock(eventingMain, "aws_lambda_event_source_mapping", "fetcher");
    expect(hasAttribute(mapping, "event_source_arn", /aws_sqs_queue\.fetch_task\.arn/)).toBe(true);
    expect(hasAttribute(mapping, "function_name", /var\.fetcher_lambda_arn/)).toBe(true);
    expect(bodyIncludes(mapping, 'function_response_types = ["ReportBatchItemFailures"]')).toBe(
      true,
    );
    expect(devMain).toContain('actions   = ["sqs:SendMessage"]');
    expect(bodyIncludes(moduleBlock(devMain, "api_lambda"), "FETCH_TASK_QUEUE_URL")).toBe(true);
    expect(
      bodyIncludes(moduleBlock(devMain, "fetcher_lambda"), "FETCH_TASK_DIAGNOSTIC_QUEUE_URL"),
    ).toBe(true);
    expect(devMain).toContain("module.fetch_task_queue.fetch_task_dlq_arn");
  });

  it("wires an EventBridge dispatcher schedule that can enqueue due-flight tasks", () => {
    expect(
      hasAttribute(
        resourceBlock(eventingMain, "aws_cloudwatch_event_rule", "dispatcher"),
        "schedule_expression",
        /var\.dispatcher_schedule_expression/,
      ),
    ).toBe(true);
    expect(resourceBlock(eventingMain, "aws_cloudwatch_event_target", "dispatcher")).toBeDefined();
    const dispatcher = moduleBlock(devMain, "dispatcher_lambda");
    expect(bodyIncludes(dispatcher, 'NOOP_FETCH_ENABLED        = "false"')).toBe(true);
    expect(bodyIncludes(dispatcher, 'DISPATCHER_MODE           = "active"')).toBe(true);
    expect(bodyIncludes(dispatcher, "FETCH_TASK_QUEUE_URL")).toBe(true);
    expect(bodyIncludes(dispatcher, "FLIGHTAWARE_FETCH_ENABLED")).toBe(true);
    expect(bodyIncludes(dispatcher, "FLIGHTAWARE_REAL_CALLS_ENABLED")).toBe(true);
  });

  it("creates the minimal dev DynamoDB tables for cached flight data and usage budget", () => {
    expect(moduleBlock(devMain, "data_tables")).toBeDefined();
    expect(resourceBlock(dynamodbMain, "aws_dynamodb_table", "flights")).toBeDefined();
    expect(resourceBlock(dynamodbMain, "aws_dynamodb_table", "flight_lookup")).toBeDefined();
    expect(resourceBlock(dynamodbMain, "aws_dynamodb_table", "flight_positions")).toBeDefined();
    expect(resourceBlock(dynamodbMain, "aws_dynamodb_table", "usage_budget")).toBeDefined();
    expect(outputBlock(devOutputs, "dynamodb_table_names")).toBeDefined();
  });

  it("keys DynamoDB access paths for lookup and due polling without table scans", () => {
    const lookup = resourceBlock(dynamodbMain, "aws_dynamodb_table", "flight_lookup");
    expect(hasAttribute(lookup, "hash_key", /"lookupType"/)).toBe(true);
    expect(hasAttribute(lookup, "range_key", /"lookupKey"/)).toBe(true);
    expect(bodyIncludes(lookup, 'name = "lookupType"')).toBe(true);
    expect(bodyIncludes(lookup, 'name = "lookupKey"')).toBe(true);

    const flights = resourceBlock(dynamodbMain, "aws_dynamodb_table", "flights");
    expect(bodyIncludes(flights, 'name            = "poll-due-index"')).toBe(true);
    expect(bodyIncludes(flights, 'hash_key        = "pollShard"')).toBe(true);
    expect(bodyIncludes(flights, 'range_key       = "nextPollAt"')).toBe(true);
  });

  it("creates a lifecycle-managed S3 bucket for route and track GeoJSON artifacts", () => {
    expect(moduleBlock(devMain, "geojson_storage")).toBeDefined();
    expect(resourceBlock(storageMain, "aws_s3_bucket", "geojson")).toBeDefined();
    const lifecycle = resourceBlock(
      storageMain,
      "aws_s3_bucket_lifecycle_configuration",
      "geojson",
    );
    expect(bodyIncludes(lifecycle, 'prefix = "routes/"')).toBe(true);
    expect(bodyIncludes(lifecycle, 'prefix = "tracks/"')).toBe(true);
    expect(outputBlock(devOutputs, "geojson_bucket_name")).toBeDefined();
  });

  it("separates fetch task processing from failed task storage with an SQS DLQ", () => {
    expect(resourceBlock(eventingMain, "aws_sqs_queue", "fetch_task_dlq")).toBeDefined();
    expect(
      bodyIncludes(resourceBlock(eventingMain, "aws_sqs_queue", "fetch_task"), "redrive_policy"),
    ).toBe(true);
    expect(outputBlock(devOutputs, "fetch_task_dlq_url")).toBeDefined();
  });

  it("uses Secrets Manager metadata references without raw FlightAware keys in Terraform", () => {
    expect(moduleBlock(devMain, "secret_references")).toBeDefined();
    expect(
      resourceBlock(secretsMain, "aws_secretsmanager_secret", "flightaware_api_key"),
    ).toBeDefined();
    expect(secretsMain).not.toMatch(/secret_string|secret_binary|x-apikey|API_KEY_VALUE/i);
    expect(variableBlock(devVariables, "flightaware_api_key_secret_name")).toBeDefined();
    expect(devVariables).not.toMatch(/flightaware_api_key\\s*=|api_key_value/i);
    expect(outputBlock(devOutputs, "flightaware_api_key_secret_arn")).toBeDefined();
  });

  it("creates CloudWatch alarms for 429, budget stop, Lambda errors, and DLQ depth", () => {
    expect(moduleBlock(devMain, "observability")).toBeDefined();
    expect(
      hasAttribute(
        resourceBlock(observabilityMain, "aws_cloudwatch_metric_alarm", "flightaware_rate_limited"),
        "metric_name",
        /"RateLimitedCount"/,
      ),
    ).toBe(true);
    expect(
      hasAttribute(
        resourceBlock(observabilityMain, "aws_cloudwatch_metric_alarm", "flightaware_budget_stop"),
        "metric_name",
        /"BudgetStopCount"/,
      ),
    ).toBe(true);
    expect(
      resourceBlock(observabilityMain, "aws_cloudwatch_metric_alarm", "lambda_errors"),
    ).toBeDefined();
    expect(
      hasAttribute(
        resourceBlock(observabilityMain, "aws_cloudwatch_metric_alarm", "fetch_task_dlq_depth"),
        "metric_name",
        /"ApproximateNumberOfMessagesVisible"/,
      ),
    ).toBe(true);
    expect(outputBlock(devOutputs, "cloudwatch_alarm_names")).toBeDefined();
  });
});
