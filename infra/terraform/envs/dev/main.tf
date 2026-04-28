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
}

data "aws_iam_policy_document" "dispatcher_noop_enqueue" {
  statement {
    actions   = ["sqs:SendMessage"]
    resources = [local.fetch_task_queue_arn]
  }
}

module "api_lambda" {
  source = "../../modules/compute-lambda"

  function_name   = "${local.name_prefix}-api"
  description     = "Airpath dev Go API Lambda shell without direct FlightAware calls."
  artifact_path   = var.api_lambda_artifact_path
  memory_size     = 128
  timeout_seconds = 10

  environment_variables = {
    AIRPATH_ENVIRONMENT       = var.environment
    FLIGHTAWARE_FETCH_ENABLED = "false"
  }

  tags = local.common_tags
}

module "fetcher_lambda" {
  source = "../../modules/compute-lambda"

  function_name   = "${local.name_prefix}-fetcher"
  description     = "Airpath dev SQS-triggered fetcher Lambda shell using mock upstream behavior."
  artifact_path   = var.fetcher_lambda_artifact_path
  memory_size     = 128
  timeout_seconds = 30

  environment_variables = {
    AIRPATH_ENVIRONMENT = var.environment
    FETCHER_MODE        = "mock"
  }

  tags = local.common_tags
}

module "dispatcher_lambda" {
  source = "../../modules/compute-lambda"

  function_name   = "${local.name_prefix}-dispatcher"
  description     = "Airpath dev no-op due-flight dispatcher Lambda shell."
  artifact_path   = var.dispatcher_lambda_artifact_path
  memory_size     = 128
  timeout_seconds = 10
  policy_json     = data.aws_iam_policy_document.dispatcher_noop_enqueue.json

  environment_variables = {
    AIRPATH_ENVIRONMENT  = var.environment
    FETCH_TASK_QUEUE_URL = local.fetch_task_queue_url
    NOOP_FETCH_ENABLED   = "true"
    DISPATCHER_MODE      = "noop"
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
