mock_provider "aws" {
  override_during = plan
}

run "rejects_unsupported_access_log_retention_days" {
  command = plan

  variables {
    name                      = "airpath-test-api"
    api_lambda_invoke_arn     = "arn:aws:lambda:us-east-1:123456789012:function:airpath-test-api:live"
    api_lambda_function_name  = "airpath-test-api"
    access_log_retention_days = 2
  }

  expect_failures = [
    var.access_log_retention_days,
  ]
}

run "rejects_zero_throttling_burst_limit" {
  command = plan

  variables {
    name                     = "airpath-test-api"
    api_lambda_invoke_arn    = "arn:aws:lambda:us-east-1:123456789012:function:airpath-test-api:live"
    api_lambda_function_name = "airpath-test-api"
    throttling_burst_limit   = 0
  }

  expect_failures = [
    var.throttling_burst_limit,
  ]
}

run "rejects_zero_throttling_rate_limit" {
  command = plan

  variables {
    name                     = "airpath-test-api"
    api_lambda_invoke_arn    = "arn:aws:lambda:us-east-1:123456789012:function:airpath-test-api:live"
    api_lambda_function_name = "airpath-test-api"
    throttling_rate_limit    = 0
  }

  expect_failures = [
    var.throttling_rate_limit,
  ]
}
