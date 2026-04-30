import { describe, expect, it } from "vitest";

import {
  bodyIncludes,
  dataBlock,
  moduleBlock,
  readTerraformFile,
  resourceBlock,
} from "../../test-support/hcl";

describe("services runtime contract", () => {
  const devMain = readTerraformFile("envs/dev/main.tf");
  const devVariables = readTerraformFile("envs/dev/variables.tf");
  const devOutputs = readTerraformFile("envs/dev/outputs.tf");
  const eventingMain = readTerraformFile("modules/eventing/main.tf");
  const secretsMain = readTerraformFile("modules/secrets/main.tf");

  it("passes the runtimewiring environment contract to each deployed Lambda", () => {
    const api = moduleBlock(devMain, "api_lambda");
    for (const name of [
      "AIRPATH_PERSONAL_DEMO_NOTICE",
      "AIRPATH_ENVIRONMENT",
      "FLIGHTAWARE_API_KEY_SECRET_ARN",
      "FLIGHTAWARE_FETCH_DISABLED_REASON",
      "FLIGHTAWARE_FETCH_ENABLED",
      "FLIGHTAWARE_REAL_CALLS_ENABLED",
      "FETCH_TASK_QUEUE_URL",
      "FLIGHTS_TABLE_NAME",
      "FLIGHT_LOOKUP_TABLE_NAME",
      "FLIGHT_POSITIONS_TABLE_NAME",
      "GEOJSON_BUCKET_NAME",
      "USAGE_BUDGET_TABLE_NAME",
    ]) {
      expect(bodyIncludes(api, name), `api env ${name}`).toBe(true);
    }

    const fetcher = moduleBlock(devMain, "fetcher_lambda");
    for (const name of [
      "AIRPATH_PERSONAL_DEMO_NOTICE",
      "AIRPATH_ENVIRONMENT",
      "FETCHER_MODE",
      "FLIGHTAWARE_API_KEY_SECRET_ARN",
      "FLIGHTAWARE_FETCH_DISABLED_REASON",
      "FLIGHTAWARE_FETCH_ENABLED",
      "FLIGHTAWARE_REAL_CALLS_ENABLED",
      "FETCH_TASK_DIAGNOSTIC_QUEUE_URL",
      "FLIGHTS_TABLE_NAME",
      "FLIGHT_LOOKUP_TABLE_NAME",
      "FLIGHT_POSITIONS_TABLE_NAME",
      "GEOJSON_BUCKET_NAME",
      "USAGE_BUDGET_TABLE_NAME",
    ]) {
      expect(bodyIncludes(fetcher, name), `fetcher env ${name}`).toBe(true);
    }

    const dispatcher = moduleBlock(devMain, "dispatcher_lambda");
    for (const name of [
      "AIRPATH_ENVIRONMENT",
      "DISPATCHER_ASSUME_ACTIVE_VIEWER",
      "DISPATCHER_MODE",
      "FETCH_TASK_QUEUE_URL",
      "FLIGHTS_TABLE_NAME",
      "FLIGHT_LOOKUP_TABLE_NAME",
      "FLIGHTAWARE_FETCH_ENABLED",
      "FLIGHTAWARE_REAL_CALLS_ENABLED",
      "NOOP_FETCH_ENABLED",
      "USAGE_BUDGET_TABLE_NAME",
    ]) {
      expect(bodyIncludes(dispatcher, name), `dispatcher env ${name}`).toBe(true);
    }
    expect(bodyIncludes(dispatcher, "module.fetch_task_queue.fetch_task_queue_url")).toBe(true);
  });

  it("keeps fetcher SQS batches on partial batch response semantics", () => {
    const mapping = resourceBlock(eventingMain, "aws_lambda_event_source_mapping", "fetcher");
    expect(bodyIncludes(mapping, 'function_response_types = ["ReportBatchItemFailures"]')).toBe(
      true,
    );
  });

  it("allows dispatcher to query due poll state, reserve fetch tasks, and enqueue to the fetch task queue", () => {
    const dispatcherPolicy = dataBlock(devMain, "aws_iam_policy_document", "dispatcher_data_access");
    for (const action of [
      "dynamodb:DeleteItem",
      "dynamodb:GetItem",
      "dynamodb:PutItem",
      "dynamodb:Query",
      "dynamodb:UpdateItem",
      "sqs:SendMessage",
    ]) {
      expect(bodyIncludes(dispatcherPolicy, action), `dispatcher action ${action}`).toBe(true);
    }
    expect(bodyIncludes(dispatcherPolicy, "module.data_tables.table_arns.flights")).toBe(true);
    expect(bodyIncludes(dispatcherPolicy, "/index/poll-due-index")).toBe(true);
    expect(bodyIncludes(dispatcherPolicy, "module.data_tables.table_arns.flight_lookup")).toBe(
      true,
    );
    expect(bodyIncludes(dispatcherPolicy, "module.data_tables.table_arns.usage_budget")).toBe(
      true,
    );
    expect(bodyIncludes(dispatcherPolicy, "module.fetch_task_queue.fetch_task_queue_arn")).toBe(
      true,
    );
    expect(
      bodyIncludes(
        moduleBlock(devMain, "dispatcher_lambda"),
        "policy_json     = data.aws_iam_policy_document.dispatcher_data_access.json",
      ),
    ).toBe(true);
  });

  it("allows fetcher access to DynamoDB, S3 artifacts, FlightAware secret metadata/value, and diagnostics SQS", () => {
    const fetcherPolicy = dataBlock(devMain, "aws_iam_policy_document", "fetcher_data_access");
    for (const action of [
      "dynamodb:DeleteItem",
      "dynamodb:GetItem",
      "dynamodb:PutItem",
      "dynamodb:Query",
      "dynamodb:TransactWriteItems",
      "dynamodb:UpdateItem",
      "s3:GetObject",
      "s3:PutObject",
      "secretsmanager:DescribeSecret",
      "secretsmanager:GetSecretValue",
      "sqs:SendMessage",
    ]) {
      expect(bodyIncludes(fetcherPolicy, action), `fetcher action ${action}`).toBe(true);
    }
    expect(bodyIncludes(fetcherPolicy, "values(module.data_tables.table_arns)")).toBe(true);
    expect(bodyIncludes(fetcherPolicy, "${module.geojson_storage.bucket_arn}/routes/*")).toBe(
      true,
    );
    expect(bodyIncludes(fetcherPolicy, "${module.geojson_storage.bucket_arn}/tracks/*")).toBe(
      true,
    );
    expect(
      bodyIncludes(fetcherPolicy, "module.secret_references.flightaware_api_key_secret_arn"),
    ).toBe(true);
    expect(bodyIncludes(fetcherPolicy, "module.fetch_task_queue.fetch_task_dlq_arn")).toBe(true);

    const queueConsume = dataBlock(eventingMain, "aws_iam_policy_document", "fetch_task_consume");
    for (const action of [
      "sqs:ChangeMessageVisibility",
      "sqs:DeleteMessage",
      "sqs:GetQueueAttributes",
      "sqs:ReceiveMessage",
    ]) {
      expect(bodyIncludes(queueConsume, action), `fetcher queue action ${action}`).toBe(true);
    }
  });

  it("keeps raw API key values out of Terraform variables and outputs", () => {
    expect(devVariables).toContain("flightaware_api_key_secret_name");
    expect(devOutputs).toContain("flightaware_api_key_secret_arn");
    expect(secretsMain).not.toMatch(/secret_string|secret_binary/i);
    expect(`${devVariables}\n${devOutputs}`).not.toMatch(
      /FLIGHTAWARE_API_KEY\s*=|flightaware_api_key_value|api_key_value|x-apikey/i,
    );
  });
});
