mock_provider "aws" {
  override_during = plan
}

override_data {
  target = data.aws_iam_policy_document.assume_role
  values = {
    json = "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Action\":\"sts:AssumeRole\",\"Effect\":\"Allow\",\"Principal\":{\"Service\":\"lambda.amazonaws.com\"}}]}"
  }
}

run "rejects_memory_size_below_lambda_minimum" {
  command = plan

  variables {
    function_name = "airpath-test-api"
    artifact_path = "./missing-bootstrap.zip"
    memory_size   = 127
  }

  expect_failures = [
    var.memory_size,
  ]
}

run "rejects_timeout_seconds_below_lambda_minimum" {
  command = plan

  variables {
    function_name   = "airpath-test-api"
    artifact_path   = "./missing-bootstrap.zip"
    timeout_seconds = 0
  }

  expect_failures = [
    var.timeout_seconds,
  ]
}

run "rejects_unsupported_log_retention_days" {
  command = plan

  variables {
    function_name      = "airpath-test-api"
    artifact_path      = "./missing-bootstrap.zip"
    log_retention_days = 2
  }

  expect_failures = [
    var.log_retention_days,
  ]
}
