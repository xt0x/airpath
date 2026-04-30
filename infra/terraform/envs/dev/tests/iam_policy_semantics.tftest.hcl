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
  target = module.api_lambda
  outputs = {
    function_name = "airpath-dev-api"
    function_arn  = "arn:aws:lambda:ap-northeast-1:123456789012:function:airpath-dev-api"
    invoke_arn    = "arn:aws:apigateway:ap-northeast-1:lambda:path/2015-03-31/functions/arn:aws:lambda:ap-northeast-1:123456789012:function:airpath-dev-api/invocations"
    role_arn      = "arn:aws:iam::123456789012:role/airpath-dev-api-role"
    role_name     = "airpath-dev-api-role"
  }
}

override_module {
  target = module.fetcher_lambda
  outputs = {
    function_name = "airpath-dev-fetcher"
    function_arn  = "arn:aws:lambda:ap-northeast-1:123456789012:function:airpath-dev-fetcher"
    invoke_arn    = "arn:aws:apigateway:ap-northeast-1:lambda:path/2015-03-31/functions/arn:aws:lambda:ap-northeast-1:123456789012:function:airpath-dev-fetcher/invocations"
    role_arn      = "arn:aws:iam::123456789012:role/airpath-dev-fetcher-role"
    role_name     = "airpath-dev-fetcher-role"
  }
}

override_module {
  target = module.dispatcher_lambda
  outputs = {
    function_name = "airpath-dev-dispatcher"
    function_arn  = "arn:aws:lambda:ap-northeast-1:123456789012:function:airpath-dev-dispatcher"
    invoke_arn    = "arn:aws:apigateway:ap-northeast-1:lambda:path/2015-03-31/functions/arn:aws:lambda:ap-northeast-1:123456789012:function:airpath-dev-dispatcher/invocations"
    role_arn      = "arn:aws:iam::123456789012:role/airpath-dev-dispatcher-role"
    role_name     = "airpath-dev-dispatcher-role"
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

run "keeps_secret_value_access_fetcher_only" {
  command = plan

  assert {
    condition = (
      !contains(flatten([
        for statement in jsondecode(data.aws_iam_policy_document.api_data_access.json).Statement :
        flatten([statement.Action])
      ]), "secretsmanager:GetSecretValue") &&
      contains(flatten([
        for statement in jsondecode(data.aws_iam_policy_document.fetcher_data_access.json).Statement :
        flatten([statement.Action])
      ]), "secretsmanager:GetSecretValue") &&
      !contains(flatten([
        for statement in jsondecode(data.aws_iam_policy_document.dispatcher_data_access.json).Statement :
        flatten([statement.Action])
      ]), "secretsmanager:GetSecretValue")
    )
    error_message = "Only the fetcher IAM policy may read the FlightAware secret value."
  }
}

run "keeps_dispatcher_out_of_s3_and_secrets" {
  command = plan

  assert {
    condition = length([
      for action in flatten([
        for statement in jsondecode(data.aws_iam_policy_document.dispatcher_data_access.json).Statement :
        flatten([statement.Action])
      ]) : action
      if startswith(action, "s3:") || startswith(action, "secretsmanager:")
    ]) == 0
    error_message = "Dispatcher IAM policy must not include S3 or Secrets Manager permissions."
  }
}

run "scopes_sqs_and_dynamodb_permissions_to_module_outputs" {
  command = plan

  assert {
    condition = alltrue([
      for resource in flatten([
        for statement in jsondecode(data.aws_iam_policy_document.api_data_access.json).Statement :
        flatten([statement.Resource])
        if length([
          for action in flatten([statement.Action]) : action
          if startswith(action, "dynamodb:")
        ]) > 0
        ]) : contains([
        "arn:aws:dynamodb:ap-northeast-1:123456789012:table/airpath-dev-flights",
        "arn:aws:dynamodb:ap-northeast-1:123456789012:table/airpath-dev-flight-lookup",
        "arn:aws:dynamodb:ap-northeast-1:123456789012:table/airpath-dev-flight-positions",
        "arn:aws:dynamodb:ap-northeast-1:123456789012:table/airpath-dev-usage-budget",
      ], resource)
    ])
    error_message = "API DynamoDB permissions must stay scoped to data_tables outputs."
  }

  assert {
    condition = alltrue([
      for resource in flatten([
        for statement in jsondecode(data.aws_iam_policy_document.api_data_access.json).Statement :
        flatten([statement.Resource])
        if length([
          for action in flatten([statement.Action]) : action
          if startswith(action, "sqs:")
        ]) > 0
        ]) : contains([
        "arn:aws:sqs:ap-northeast-1:123456789012:airpath-dev-fetch-task",
      ], resource)
    ])
    error_message = "API SQS permissions must stay scoped to the fetch_task_queue output."
  }

  assert {
    condition = alltrue([
      for resource in flatten([
        for statement in jsondecode(data.aws_iam_policy_document.dispatcher_data_access.json).Statement :
        flatten([statement.Resource])
        if length([
          for action in flatten([statement.Action]) : action
          if startswith(action, "dynamodb:") || startswith(action, "sqs:")
        ]) > 0
        ]) : contains([
        "arn:aws:dynamodb:ap-northeast-1:123456789012:table/airpath-dev-flights",
        "arn:aws:dynamodb:ap-northeast-1:123456789012:table/airpath-dev-flights/index/poll-due-index",
        "arn:aws:dynamodb:ap-northeast-1:123456789012:table/airpath-dev-flight-lookup",
        "arn:aws:dynamodb:ap-northeast-1:123456789012:table/airpath-dev-usage-budget",
        "arn:aws:sqs:ap-northeast-1:123456789012:airpath-dev-fetch-task",
      ], resource)
    ])
    error_message = "Dispatcher SQS and DynamoDB permissions must stay scoped to the expected data_tables and fetch_task_queue outputs."
  }

  assert {
    condition = alltrue([
      for resource in flatten([
        for statement in jsondecode(data.aws_iam_policy_document.fetcher_data_access.json).Statement :
        flatten([statement.Resource])
        if length([
          for action in flatten([statement.Action]) : action
          if startswith(action, "dynamodb:")
        ]) > 0
        ]) : contains([
        "arn:aws:dynamodb:ap-northeast-1:123456789012:table/airpath-dev-flights",
        "arn:aws:dynamodb:ap-northeast-1:123456789012:table/airpath-dev-flight-lookup",
        "arn:aws:dynamodb:ap-northeast-1:123456789012:table/airpath-dev-flight-positions",
        "arn:aws:dynamodb:ap-northeast-1:123456789012:table/airpath-dev-usage-budget",
      ], resource)
    ])
    error_message = "Fetcher DynamoDB permissions must stay scoped to data_tables outputs."
  }

  assert {
    condition = alltrue([
      for resource in flatten([
        for statement in jsondecode(data.aws_iam_policy_document.fetcher_data_access.json).Statement :
        flatten([statement.Resource])
        if length([
          for action in flatten([statement.Action]) : action
          if startswith(action, "sqs:")
        ]) > 0
        ]) : contains([
        "arn:aws:sqs:ap-northeast-1:123456789012:airpath-dev-fetch-task-dlq",
      ], resource)
    ])
    error_message = "Fetcher SQS permissions must stay scoped to the fetch_task_queue DLQ output."
  }
}
