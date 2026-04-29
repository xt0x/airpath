data "aws_caller_identity" "current" {}
data "aws_partition" "current" {}

locals {
  name_prefix = "airpath-${var.environment}"

  common_tags = {
    Project     = "airpath"
    Environment = var.environment
    ManagedBy   = "terraform"
  }

  fetch_task_queue_name = "${local.name_prefix}-fetch-task"
  fetch_task_queue_arn  = "arn:${data.aws_partition.current.partition}:sqs:${var.aws_region}:${data.aws_caller_identity.current.account_id}:${local.fetch_task_queue_name}"
  fetch_task_queue_url  = "https://sqs.${var.aws_region}.amazonaws.com/${data.aws_caller_identity.current.account_id}/${local.fetch_task_queue_name}"
  geojson_bucket_name   = "${local.name_prefix}-geojson-${data.aws_caller_identity.current.account_id}-${var.aws_region}"

  flightaware_real_calls_enabled    = var.allow_real_flightaware_calls ? "true" : "false"
  flightaware_fetch_enabled         = var.allow_real_flightaware_calls ? "true" : "false"
  flightaware_fetch_disabled_reason = var.allow_real_flightaware_calls ? "" : var.flightaware_fetch_disabled_reason
}

data "aws_iam_policy_document" "dispatcher_enqueue" {
  statement {
    actions   = ["sqs:SendMessage"]
    resources = [local.fetch_task_queue_arn]
  }
}

data "aws_iam_policy_document" "api_data_access" {
  statement {
    actions = [
      "dynamodb:GetItem",
      "dynamodb:PutItem",
      "dynamodb:Query",
      "dynamodb:UpdateItem",
    ]

    resources = values(module.data_tables.table_arns)
  }

  statement {
    actions = [
      "s3:GetObject",
      "s3:PutObject",
    ]

    resources = [
      "${module.geojson_storage.bucket_arn}/routes/*",
      "${module.geojson_storage.bucket_arn}/tracks/*",
    ]
  }

  statement {
    actions   = ["sqs:SendMessage"]
    resources = [module.fetch_task_queue.fetch_task_queue_arn]
  }

  statement {
    actions   = ["secretsmanager:DescribeSecret"]
    resources = [module.secret_references.flightaware_api_key_secret_arn]
  }
}

data "aws_iam_policy_document" "fetcher_data_access" {
  statement {
    actions = [
      "dynamodb:GetItem",
      "dynamodb:PutItem",
      "dynamodb:Query",
      "dynamodb:UpdateItem",
    ]

    resources = values(module.data_tables.table_arns)
  }

  statement {
    actions = [
      "s3:GetObject",
      "s3:PutObject",
    ]

    resources = [
      "${module.geojson_storage.bucket_arn}/routes/*",
      "${module.geojson_storage.bucket_arn}/tracks/*",
    ]
  }

  statement {
    actions = [
      "secretsmanager:DescribeSecret",
      "secretsmanager:GetSecretValue",
    ]

    resources = [module.secret_references.flightaware_api_key_secret_arn]
  }
}

module "data_tables" {
  source = "../../modules/data-dynamodb"

  name_prefix                    = local.name_prefix
  point_in_time_recovery_enabled = false
  tags                           = local.common_tags
}

module "geojson_storage" {
  source = "../../modules/storage-s3"

  bucket_name             = local.geojson_bucket_name
  artifact_retention_days = var.geojson_artifact_retention_days
  tags                    = local.common_tags
}

module "secret_references" {
  source = "../../modules/secrets"

  flightaware_api_key_secret_name = var.flightaware_api_key_secret_name
  tags                            = local.common_tags
}

module "api_lambda" {
  source = "../../modules/compute-lambda"

  function_name   = "${local.name_prefix}-api"
  description     = "Airpath dev Go API Lambda."
  artifact_path   = var.api_lambda_artifact_path
  memory_size     = 128
  timeout_seconds = 10
  policy_json     = data.aws_iam_policy_document.api_data_access.json

  environment_variables = {
    AIRPATH_PERSONAL_DEMO_NOTICE      = var.personal_demo_notice
    AIRPATH_ENVIRONMENT               = var.environment
    FLIGHTAWARE_API_KEY_SECRET_ARN    = module.secret_references.flightaware_api_key_secret_arn
    FLIGHTAWARE_FETCH_DISABLED_REASON = local.flightaware_fetch_disabled_reason
    FLIGHTAWARE_FETCH_ENABLED         = local.flightaware_fetch_enabled
    FLIGHTAWARE_REAL_CALLS_ENABLED    = local.flightaware_real_calls_enabled
    FETCH_TASK_QUEUE_URL              = module.fetch_task_queue.fetch_task_queue_url
    FLIGHTS_TABLE_NAME                = module.data_tables.table_names.flights
    FLIGHT_LOOKUP_TABLE_NAME          = module.data_tables.table_names.flight_lookup
    FLIGHT_POSITIONS_TABLE_NAME       = module.data_tables.table_names.flight_positions
    GEOJSON_BUCKET_NAME               = module.geojson_storage.bucket_name
    USAGE_BUDGET_TABLE_NAME           = module.data_tables.table_names.usage_budget
  }

  tags = local.common_tags
}

module "fetcher_lambda" {
  source = "../../modules/compute-lambda"

  function_name   = "${local.name_prefix}-fetcher"
  description     = "Airpath dev SQS-triggered fetcher Lambda."
  artifact_path   = var.fetcher_lambda_artifact_path
  memory_size     = 128
  timeout_seconds = 30
  policy_json     = data.aws_iam_policy_document.fetcher_data_access.json

  environment_variables = {
    AIRPATH_PERSONAL_DEMO_NOTICE      = var.personal_demo_notice
    AIRPATH_ENVIRONMENT               = var.environment
    FETCHER_MODE                      = var.allow_real_flightaware_calls ? "real-opt-in" : "mock"
    FLIGHTAWARE_API_KEY_SECRET_ARN    = module.secret_references.flightaware_api_key_secret_arn
    FLIGHTAWARE_FETCH_DISABLED_REASON = local.flightaware_fetch_disabled_reason
    FLIGHTAWARE_FETCH_ENABLED         = local.flightaware_fetch_enabled
    FLIGHTAWARE_REAL_CALLS_ENABLED    = local.flightaware_real_calls_enabled
    FLIGHTS_TABLE_NAME                = module.data_tables.table_names.flights
    FLIGHT_LOOKUP_TABLE_NAME          = module.data_tables.table_names.flight_lookup
    FLIGHT_POSITIONS_TABLE_NAME       = module.data_tables.table_names.flight_positions
    GEOJSON_BUCKET_NAME               = module.geojson_storage.bucket_name
    USAGE_BUDGET_TABLE_NAME           = module.data_tables.table_names.usage_budget
  }

  tags = local.common_tags
}

module "dispatcher_lambda" {
  source = "../../modules/compute-lambda"

  function_name   = "${local.name_prefix}-dispatcher"
  description     = "Airpath dev due-flight dispatcher Lambda."
  artifact_path   = var.dispatcher_lambda_artifact_path
  memory_size     = 128
  timeout_seconds = 10
  policy_json     = data.aws_iam_policy_document.dispatcher_enqueue.json

  environment_variables = {
    AIRPATH_ENVIRONMENT  = var.environment
    FETCH_TASK_QUEUE_URL = local.fetch_task_queue_url
    NOOP_FETCH_ENABLED   = "false"
    DISPATCHER_MODE      = "active"
  }

  tags = local.common_tags
}

module "fetch_task_queue" {
  source = "../../modules/eventing"

  name_prefix                     = local.name_prefix
  fetcher_lambda_arn              = module.fetcher_lambda.function_arn
  fetcher_lambda_role_name        = module.fetcher_lambda.role_name
  dispatcher_lambda_arn           = module.dispatcher_lambda.function_arn
  dispatcher_lambda_function_name = module.dispatcher_lambda.function_name
  dispatcher_schedule_expression  = var.dispatcher_schedule_expression
  fetch_task_max_receive_count    = var.fetch_task_max_receive_count
  tags                            = local.common_tags
}

module "http_api" {
  source = "../../modules/api-http"

  name                     = "${local.name_prefix}-http-api"
  api_lambda_invoke_arn    = module.api_lambda.invoke_arn
  api_lambda_function_name = module.api_lambda.function_name
  stage_name               = "$default"
  tags                     = local.common_tags
}

module "observability" {
  source = "../../modules/observability"

  name_prefix = local.name_prefix
  environment = var.environment

  lambda_function_names = [
    module.api_lambda.function_name,
    module.fetcher_lambda.function_name,
    module.dispatcher_lambda.function_name,
  ]

  fetch_task_queue_name = module.fetch_task_queue.fetch_task_queue_name
  fetch_task_dlq_name   = module.fetch_task_queue.fetch_task_dlq_name
  tags                  = local.common_tags
}
