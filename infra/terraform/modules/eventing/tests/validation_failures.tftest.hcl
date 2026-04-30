mock_provider "aws" {
  override_during = plan
}

run "rejects_zero_fetch_task_max_receive_count" {
  command = plan

  variables {
    name_prefix                     = "airpath-test"
    fetcher_lambda_arn              = "arn:aws:lambda:us-east-1:123456789012:function:airpath-test-fetcher"
    fetcher_lambda_role_name        = "airpath-test-fetcher-role"
    dispatcher_lambda_arn           = "arn:aws:lambda:us-east-1:123456789012:function:airpath-test-dispatcher"
    dispatcher_lambda_function_name = "airpath-test-dispatcher"
    dispatcher_schedule_expression  = "rate(5 minutes)"
    fetch_task_max_receive_count    = 0
  }

  expect_failures = [
    var.fetch_task_max_receive_count,
  ]
}
