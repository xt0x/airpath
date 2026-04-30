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
      aws_lambda_permission.allow_http_api.function_name == "airpath-test-api" &&
      aws_lambda_permission.allow_http_api.principal == "apigateway.amazonaws.com"
    )
    error_message = "HTTP API stage and Lambda invoke permission must match the API contract."
  }
}

run "wires_route_target_after_mock_apply" {
  command = apply

  variables {
    name                     = "airpath-test-api"
    api_lambda_invoke_arn    = "arn:aws:lambda:us-east-1:123456789012:function:airpath-test-api:live"
    api_lambda_function_name = "airpath-test-api"
  }

  assert {
    condition     = aws_apigatewayv2_route.v1_proxy.target == "integrations/${aws_apigatewayv2_integration.api_lambda.id}"
    error_message = "HTTP API must route the /v1 proxy to the Lambda integration."
  }

  assert {
    condition     = aws_lambda_permission.allow_http_api.source_arn == "${aws_apigatewayv2_api.this.execution_arn}/*/*/v1/*"
    error_message = "HTTP API Lambda permission must scope invocation to /v1 routes."
  }
}
