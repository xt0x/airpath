mock_provider "aws" {
  override_during = plan
}

run "plans_flightaware_alarms_lambda_alarms_and_dashboard" {
  command = plan

  variables {
    name_prefix = "airpath-test"
    environment = "test"
    aws_region  = "ap-northeast-1"
    lambda_timeout_seconds_by_function = {
      airpath-test-api     = 10
      airpath-test-fetcher = 30
    }
    fetch_task_dlq_name       = "airpath-test-fetch-task-dlq"
    fetch_task_queue_name     = "airpath-test-fetch-task"
    budget_soft_threshold_usd = 3.5
    alarm_actions             = ["arn:aws:sns:us-east-1:123456789012:airpath-test-alerts"]
    tags = {
      Environment = "test"
      Project     = "airpath"
    }
  }

  assert {
    condition = (
      aws_cloudwatch_metric_alarm.flightaware_rate_limited.metric_name == "RateLimitedCount" &&
      aws_cloudwatch_metric_alarm.flightaware_rate_limited.threshold == 0 &&
      aws_cloudwatch_metric_alarm.flightaware_rate_limited.dimensions.Environment == "test"
    )
    error_message = "The FlightAware rate-limit alarm must keep its metric, threshold, and environment dimension."
  }

  assert {
    condition = (
      aws_cloudwatch_metric_alarm.flightaware_budget_soft_threshold.metric_name == "EstimatedSpendUSD" &&
      aws_cloudwatch_metric_alarm.flightaware_budget_soft_threshold.threshold == 3.5 &&
      aws_cloudwatch_metric_alarm.flightaware_budget_soft_threshold.comparison_operator == "GreaterThanOrEqualToThreshold"
    )
    error_message = "The budget soft threshold alarm must follow the configured threshold."
  }

  assert {
    condition = (
      aws_cloudwatch_metric_alarm.lambda_errors["airpath-test-api"].metric_name == "Errors" &&
      aws_cloudwatch_metric_alarm.lambda_errors["airpath-test-api"].alarm_name == "airpath-test-api-errors" &&
      aws_cloudwatch_metric_alarm.lambda_errors["airpath-test-api"].dimensions.FunctionName == "airpath-test-api" &&
      aws_cloudwatch_metric_alarm.lambda_timeouts["airpath-test-api"].metric_name == "Duration" &&
      aws_cloudwatch_metric_alarm.lambda_timeouts["airpath-test-api"].alarm_name == "airpath-test-api-timeouts" &&
      aws_cloudwatch_metric_alarm.lambda_timeouts["airpath-test-api"].threshold == 8000 &&
      aws_cloudwatch_metric_alarm.lambda_timeouts["airpath-test-fetcher"].metric_name == "Duration" &&
      aws_cloudwatch_metric_alarm.lambda_timeouts["airpath-test-fetcher"].threshold == 28000
    )
    error_message = "Lambda error and timeout alarms must avoid duplicated prefixes and derive timeout thresholds from each function timeout."
  }

  assert {
    condition = (
      aws_cloudwatch_metric_alarm.fetch_task_dlq_depth.dimensions.QueueName == "airpath-test-fetch-task-dlq" &&
      aws_cloudwatch_metric_alarm.fetch_task_queue_age.dimensions.QueueName == "airpath-test-fetch-task" &&
      aws_cloudwatch_metric_alarm.fetch_task_queue_age.threshold == 900
    )
    error_message = "SQS alarms must use the configured queue names and queue-age threshold."
  }

  assert {
    condition = (
      aws_cloudwatch_dashboard.flightaware.dashboard_name == "airpath-test-flightaware-free-allowance" &&
      length(jsondecode(aws_cloudwatch_dashboard.flightaware.dashboard_body).widgets) == 4 &&
      jsondecode(aws_cloudwatch_dashboard.flightaware.dashboard_body).widgets[0].properties.title == "FlightAware Calls And 429s" &&
      jsondecode(aws_cloudwatch_dashboard.flightaware.dashboard_body).widgets[0].properties.region == "ap-northeast-1"
    )
    error_message = "The dashboard must keep the expected FlightAware operations widgets and concrete AWS region."
  }
}
