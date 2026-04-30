mock_provider "aws" {
  override_during = plan
}

override_data {
  target = data.aws_iam_policy_document.assume_role
  values = {
    json = "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Action\":\"sts:AssumeRole\",\"Effect\":\"Allow\",\"Principal\":{\"Service\":\"lambda.amazonaws.com\"}}]}"
  }
}

run "plans_lambda_runtime_and_environment_contract" {
  command = plan

  variables {
    function_name = "airpath-test-api"
    description   = "test api"
    artifact_path = "./missing-bootstrap.zip"
    environment_variables = {
      APP_ENV     = "test"
      TABLE_NAME  = "airpath-test-flights"
      BUCKET_NAME = "airpath-test-geojson"
    }
    tags = {
      Environment = "test"
      Project     = "airpath"
    }
  }

  assert {
    condition = (
      aws_lambda_function.this.function_name == "airpath-test-api" &&
      aws_lambda_function.this.description == "test api" &&
      aws_lambda_function.this.runtime == "provided.al2023" &&
      aws_lambda_function.this.handler == "bootstrap" &&
      aws_lambda_function.this.memory_size == 128 &&
      aws_lambda_function.this.timeout == 10
    )
    error_message = "Lambda runtime, handler, memory, timeout, and identity must keep their defaults unless overridden."
  }

  assert {
    condition     = aws_lambda_function.this.environment[0].variables.APP_ENV == "test"
    error_message = "Lambda environment variables must be passed through unchanged."
  }

  assert {
    condition = (
      aws_iam_role.this.name == "airpath-test-api-role" &&
      aws_iam_role_policy_attachment.basic_execution.role == aws_iam_role.this.name &&
      aws_iam_role_policy_attachment.basic_execution.policy_arn == "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
    )
    error_message = "Lambda execution role and AWS basic logging policy attachment must be planned."
  }

  assert {
    condition     = length(aws_iam_role_policy.inline) == 0
    error_message = "The module must not attach an inline policy unless policy_json is provided."
  }
}

run "plans_overrides_and_optional_inline_policy" {
  command = plan

  variables {
    function_name   = "airpath-test-fetcher"
    artifact_path   = "./missing-bootstrap.zip"
    memory_size     = 256
    timeout_seconds = 30
    policy_json = jsonencode({
      Version = "2012-10-17"
      Statement = [
        {
          Effect   = "Allow"
          Action   = ["sqs:ReceiveMessage"]
          Resource = ["arn:aws:sqs:us-east-1:123456789012:airpath-test-fetch-task"]
        }
      ]
    })
  }

  assert {
    condition = (
      aws_lambda_function.this.memory_size == 256 &&
      aws_lambda_function.this.timeout == 30
    )
    error_message = "Lambda memory and timeout must follow module inputs."
  }

  assert {
    condition = (
      length(aws_iam_role_policy.inline) == 1 &&
      aws_iam_role_policy.inline[0].name == "airpath-test-fetcher-policy" &&
      jsondecode(aws_iam_role_policy.inline[0].policy).Statement[0].Action[0] == "sqs:ReceiveMessage"
    )
    error_message = "The module must attach the caller-provided inline IAM policy when policy_json is set."
  }
}
