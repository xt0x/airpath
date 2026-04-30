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

run "planned_storage_data_and_eventing_names_use_dev_prefix" {
  command = plan

  assert {
    condition = alltrue([
      startswith(module.data_tables.table_names.flights, "airpath-dev-"),
      startswith(module.data_tables.table_names.flight_lookup, "airpath-dev-"),
      startswith(module.data_tables.table_names.flight_positions, "airpath-dev-"),
      startswith(module.data_tables.table_names.usage_budget, "airpath-dev-"),
      startswith(module.fetch_task_queue.fetch_task_queue_name, "airpath-dev-"),
      startswith(module.fetch_task_queue.fetch_task_dlq_name, "airpath-dev-"),
      startswith(module.geojson_storage.bucket_name, "airpath-dev-"),
    ])
    error_message = "Dev storage, data, and eventing resource names must use the airpath-dev-* naming policy."
  }
}
