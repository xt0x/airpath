mock_provider "aws" {
  override_during = plan
}

override_resource {
  target = aws_apigatewayv2_api.this
  values = {
    id            = "api-test-id"
    execution_arn = "arn:aws:execute-api:us-east-1:123456789012:api-test-id"
  }
}

override_resource {
  target = aws_apigatewayv2_integration.api_lambda
  values = {
    id = "integration-test-id"
  }
}

override_resource {
  target = aws_cloudwatch_log_group.access
  values = {
    arn = "arn:aws:logs:us-east-1:123456789012:log-group:/aws/apigateway/airpath-test-api"
  }
}

run "plans_http_api_proxy_contract" {
  command = plan

  variables {
    name                     = "airpath-test-api"
    api_lambda_invoke_arn    = "arn:aws:lambda:us-east-1:123456789012:function:airpath-test-api:live"
    api_lambda_function_name = "airpath-test-api"
    stage_name               = "$default"
    tags = {
      Environment = "test"
      Project     = "airpath"
    }
  }

  assert {
    condition = (
      aws_apigatewayv2_api.this.name == "airpath-test-api" &&
      aws_apigatewayv2_api.this.protocol_type == "HTTP"
    )
    error_message = "HTTP API must use the configured name and HTTP protocol."
  }

  assert {
    condition = (
      aws_apigatewayv2_integration.api_lambda.integration_type == "AWS_PROXY" &&
      aws_apigatewayv2_integration.api_lambda.integration_uri == "arn:aws:lambda:us-east-1:123456789012:function:airpath-test-api:live" &&
      aws_apigatewayv2_integration.api_lambda.payload_format_version == "2.0"
    )
    error_message = "HTTP API Lambda integration must use AWS_PROXY payload format 2.0."
  }

  assert {
    condition     = aws_apigatewayv2_route.v1_proxy.route_key == "ANY /v1/{proxy+}"
    error_message = "HTTP API must expose the /v1 proxy route."
  }

  assert {
    condition = (
      aws_apigatewayv2_stage.this.name == "$default" &&
      aws_apigatewayv2_stage.this.auto_deploy == true &&
      length(aws_apigatewayv2_stage.this.access_log_settings) == 1 &&
      jsondecode(aws_apigatewayv2_stage.this.access_log_settings[0].format).requestId == "$context.requestId" &&
      aws_apigatewayv2_stage.this.default_route_settings[0].throttling_burst_limit == 20 &&
      aws_apigatewayv2_stage.this.default_route_settings[0].throttling_rate_limit == 10 &&
      aws_lambda_permission.allow_http_api.function_name == "airpath-test-api" &&
      aws_lambda_permission.allow_http_api.principal == "apigateway.amazonaws.com"
    )
    error_message = "HTTP API stage, logging, throttling, and Lambda invoke permission must match the API contract."
  }

  assert {
    condition = (
      aws_cloudwatch_log_group.access.name == "/aws/apigateway/airpath-test-api" &&
      aws_cloudwatch_log_group.access.retention_in_days == 30
    )
    error_message = "HTTP API access logs must use a bounded default retention period."
  }
}

run "wires_route_target_after_mock_apply" {
  command = apply

  variables {
    name                     = "airpath-test-api"
    api_lambda_invoke_arn    = "arn:aws:lambda:us-east-1:123456789012:function:airpath-test-api:live"
    api_lambda_function_name = "airpath-test-api"
    throttling_burst_limit   = 7
    throttling_rate_limit    = 3
  }

  assert {
    condition     = aws_apigatewayv2_route.v1_proxy.target == "integrations/${aws_apigatewayv2_integration.api_lambda.id}"
    error_message = "HTTP API must route the /v1 proxy to the Lambda integration."
  }

  assert {
    condition     = aws_lambda_permission.allow_http_api.source_arn == "${aws_apigatewayv2_api.this.execution_arn}/*/*/v1/*"
    error_message = "HTTP API Lambda permission must scope invocation to /v1 routes."
  }

  assert {
    condition = (
      aws_apigatewayv2_stage.this.default_route_settings[0].throttling_burst_limit == 7 &&
      aws_apigatewayv2_stage.this.default_route_settings[0].throttling_rate_limit == 3
    )
    error_message = "HTTP API throttling must follow module inputs."
  }
}
