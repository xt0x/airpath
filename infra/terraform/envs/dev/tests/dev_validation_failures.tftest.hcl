mock_provider "aws" {
  override_during = plan
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

run "rejects_prod_environment_name" {
  command = plan

  variables {
    environment = "prod"
  }

  expect_failures = [
    var.environment,
  ]
}

run "rejects_zero_geojson_artifact_retention_days" {
  command = plan

  variables {
    geojson_artifact_retention_days = 0
  }

  expect_failures = [
    var.geojson_artifact_retention_days,
  ]
}

run "rejects_zero_fetch_task_max_receive_count" {
  command = plan

  variables {
    fetch_task_max_receive_count = 0
  }

  expect_failures = [
    var.fetch_task_max_receive_count,
  ]
}
