provider "aws" {
  region                      = "ap-northeast-1"
  access_key                  = "test"
  secret_key                  = "test"
  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_requesting_account_id  = true
  skip_region_validation      = true
}

override_data {
  target = data.aws_caller_identity.current
  values = {
    account_id = "123456789012"
  }
}

override_data {
  target = data.aws_iam_policy_document.dispatcher_data_access
  values = {
    json = "{\"Version\":\"2012-10-17\",\"Statement\":[]}"
  }
}

override_data {
  target = data.aws_iam_policy_document.api_data_access
  values = {
    json = "{\"Version\":\"2012-10-17\",\"Statement\":[]}"
  }
}

override_data {
  target = data.aws_iam_policy_document.fetcher_data_access
  values = {
    json = "{\"Version\":\"2012-10-17\",\"Statement\":[]}"
  }
}

override_module {
  target = module.data_tables
  outputs = {
    table_names = {
      flights          = "airpath-dev-flights"
      flight_lookup    = "airpath-dev-flight-lookup"
      flight_positions = "airpath-dev-flight-positions"
      usage_budget     = "airpath-dev-usage-budget"
    }
    table_arns = {
      flights          = "arn:aws:dynamodb:ap-northeast-1:123456789012:table/airpath-dev-flights"
      flight_lookup    = "arn:aws:dynamodb:ap-northeast-1:123456789012:table/airpath-dev-flight-lookup"
      flight_positions = "arn:aws:dynamodb:ap-northeast-1:123456789012:table/airpath-dev-flight-positions"
      usage_budget     = "arn:aws:dynamodb:ap-northeast-1:123456789012:table/airpath-dev-usage-budget"
    }
  }
}

override_module {
  target = module.geojson_storage
  outputs = {
    bucket_name = "airpath-dev-geojson-123456789012-ap-northeast-1"
    bucket_arn  = "arn:aws:s3:::airpath-dev-geojson-123456789012-ap-northeast-1"
  }
}

override_module {
  target = module.secret_references
  outputs = {
    flightaware_api_key_secret_name = "airpath/dev/flightaware-api-key"
    flightaware_api_key_secret_arn  = "arn:aws:secretsmanager:ap-northeast-1:123456789012:secret:airpath/dev/flightaware-api-key"
  }
}

override_module {
  target = module.fetch_task_queue
  outputs = {
    fetch_task_queue_arn  = "arn:aws:sqs:ap-northeast-1:123456789012:airpath-dev-fetch-task"
    fetch_task_queue_url  = "https://sqs.ap-northeast-1.amazonaws.com/123456789012/airpath-dev-fetch-task"
    fetch_task_queue_name = "airpath-dev-fetch-task"
    fetch_task_dlq_arn    = "arn:aws:sqs:ap-northeast-1:123456789012:airpath-dev-fetch-task-dlq"
    fetch_task_dlq_name   = "airpath-dev-fetch-task-dlq"
    fetch_task_dlq_url    = "https://sqs.ap-northeast-1.amazonaws.com/123456789012/airpath-dev-fetch-task-dlq"
    dispatcher_rule_arn   = "arn:aws:events:ap-northeast-1:123456789012:rule/airpath-dev-dispatcher"
  }
}

override_module {
  target = module.http_api
  outputs = {
    api_id        = "api-id"
    api_endpoint  = "https://api-id.execute-api.ap-northeast-1.amazonaws.com"
    execution_arn = "arn:aws:execute-api:ap-northeast-1:123456789012:api-id"
  }
}

override_module {
  target = module.observability
  outputs = {
    alarm_names    = {}
    dashboard_name = "airpath-dev-flightaware"
  }
}

run "default_plan_keeps_real_flightaware_calls_disabled" {
  command = plan

  assert {
    condition = (
      module.api_lambda.environment_variables.FLIGHTAWARE_REAL_CALLS_ENABLED == "false" &&
      module.api_lambda.environment_variables.FLIGHTAWARE_FETCH_ENABLED == "false" &&
      module.fetcher_lambda.environment_variables.FETCHER_MODE == "mock" &&
      module.fetcher_lambda.environment_variables.FLIGHTAWARE_REAL_CALLS_ENABLED == "false" &&
      module.fetcher_lambda.environment_variables.FLIGHTAWARE_FETCH_ENABLED == "false" &&
      module.dispatcher_lambda.environment_variables.FLIGHTAWARE_REAL_CALLS_ENABLED == "false" &&
      module.dispatcher_lambda.environment_variables.FLIGHTAWARE_FETCH_ENABLED == "false"
    )
    error_message = "Default dev plans must keep real FlightAware calls and fetch execution disabled."
  }
}

run "opt_in_plan_enables_real_flightaware_fetch_flags" {
  command = plan

  variables {
    allow_real_flightaware_calls = true
  }

  assert {
    condition = (
      module.api_lambda.environment_variables.FLIGHTAWARE_REAL_CALLS_ENABLED == "true" &&
      module.api_lambda.environment_variables.FLIGHTAWARE_FETCH_ENABLED == "true" &&
      module.fetcher_lambda.environment_variables.FETCHER_MODE == "real-opt-in" &&
      module.fetcher_lambda.environment_variables.FLIGHTAWARE_REAL_CALLS_ENABLED == "true" &&
      module.fetcher_lambda.environment_variables.FLIGHTAWARE_FETCH_ENABLED == "true" &&
      module.dispatcher_lambda.environment_variables.FLIGHTAWARE_REAL_CALLS_ENABLED == "true" &&
      module.dispatcher_lambda.environment_variables.FLIGHTAWARE_FETCH_ENABLED == "true"
    )
    error_message = "Opting into real FlightAware calls must switch the dev runtime fetch flags and fetcher mode."
  }
}

run "outputs_expose_secret_reference_metadata_only" {
  command = plan

  assert {
    condition = (
      output.flightaware_api_key_secret_arn == "arn:aws:secretsmanager:ap-northeast-1:123456789012:secret:airpath/dev/flightaware-api-key" &&
      startswith(output.flightaware_api_key_secret_arn, "arn:aws:secretsmanager:") &&
      !strcontains(lower(output.flightaware_api_key_secret_arn), "x-apikey") &&
      !strcontains(lower(output.flightaware_api_key_secret_arn), "secret_string")
    )
    error_message = "Dev outputs must expose only the FlightAware secret ARN metadata, not any secret value."
  }
}

run "planned_lambda_names_use_dev_prefix" {
  command = plan

  assert {
    condition = alltrue([
      startswith(module.api_lambda.function_name, "airpath-dev-"),
      startswith(module.fetcher_lambda.function_name, "airpath-dev-"),
      startswith(module.dispatcher_lambda.function_name, "airpath-dev-"),
    ])
    error_message = "Dev Lambda names must use the airpath-dev-* naming policy."
  }
}
