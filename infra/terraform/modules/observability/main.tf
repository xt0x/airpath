locals {
  lambda_alarm_name_suffixes = {
    for function_name in keys(var.lambda_timeout_seconds_by_function) :
    function_name => startswith(function_name, "${var.name_prefix}-") ? trimprefix(function_name, "${var.name_prefix}-") : function_name
  }
}

resource "aws_cloudwatch_metric_alarm" "flightaware_rate_limited" {
  alarm_name          = "${var.name_prefix}-flightaware-429"
  alarm_description   = "FlightAware rate limit guard emitted at least one 429 event."
  namespace           = var.metric_namespace
  metric_name         = "RateLimitedCount"
  statistic           = "Sum"
  period              = 300
  evaluation_periods  = 1
  threshold           = 0
  comparison_operator = "GreaterThanThreshold"
  treat_missing_data  = "notBreaching"
  alarm_actions       = var.alarm_actions

  dimensions = {
    Environment = var.environment
  }

  tags = var.tags
}

resource "aws_cloudwatch_metric_alarm" "flightaware_budget_stop" {
  alarm_name          = "${var.name_prefix}-flightaware-budget-stop"
  alarm_description   = "FlightAware budget stop guard emitted at least one stop event."
  namespace           = var.metric_namespace
  metric_name         = "BudgetStopCount"
  statistic           = "Sum"
  period              = 300
  evaluation_periods  = 1
  threshold           = 0
  comparison_operator = "GreaterThanThreshold"
  treat_missing_data  = "notBreaching"
  alarm_actions       = var.alarm_actions

  dimensions = {
    Environment = var.environment
  }

  tags = var.tags
}

resource "aws_cloudwatch_metric_alarm" "flightaware_budget_soft_threshold" {
  alarm_name          = "${var.name_prefix}-flightaware-budget-soft-threshold"
  alarm_description   = "FlightAware estimated monthly spend reached the configured soft threshold."
  namespace           = var.metric_namespace
  metric_name         = "EstimatedSpendUSD"
  statistic           = "Maximum"
  period              = 300
  evaluation_periods  = 1
  threshold           = var.budget_soft_threshold_usd
  comparison_operator = "GreaterThanOrEqualToThreshold"
  treat_missing_data  = "notBreaching"
  alarm_actions       = var.alarm_actions

  dimensions = {
    Environment = var.environment
  }

  tags = var.tags
}

resource "aws_cloudwatch_metric_alarm" "flightaware_budget_hard_stop" {
  alarm_name          = "${var.name_prefix}-flightaware-budget-hard-stop"
  alarm_description   = "FlightAware hard stop guard emitted at least one stop event."
  namespace           = var.metric_namespace
  metric_name         = "BudgetHardStopCount"
  statistic           = "Sum"
  period              = 300
  evaluation_periods  = 1
  threshold           = 0
  comparison_operator = "GreaterThanThreshold"
  treat_missing_data  = "notBreaching"
  alarm_actions       = var.alarm_actions

  dimensions = {
    Environment = var.environment
  }

  tags = var.tags
}

resource "aws_cloudwatch_metric_alarm" "lambda_errors" {
  for_each = var.lambda_timeout_seconds_by_function

  alarm_name          = "${var.name_prefix}-${local.lambda_alarm_name_suffixes[each.key]}-errors"
  alarm_description   = "Lambda ${each.key} emitted at least one error."
  namespace           = "AWS/Lambda"
  metric_name         = "Errors"
  statistic           = "Sum"
  period              = 300
  evaluation_periods  = 1
  threshold           = 0
  comparison_operator = "GreaterThanThreshold"
  treat_missing_data  = "notBreaching"
  alarm_actions       = var.alarm_actions

  dimensions = {
    FunctionName = each.key
  }

  tags = var.tags
}

resource "aws_cloudwatch_metric_alarm" "lambda_timeouts" {
  for_each = var.lambda_timeout_seconds_by_function

  alarm_name          = "${var.name_prefix}-${local.lambda_alarm_name_suffixes[each.key]}-timeouts"
  alarm_description   = "Lambda ${each.key} duration is near its configured timeout window."
  namespace           = "AWS/Lambda"
  metric_name         = "Duration"
  statistic           = "Maximum"
  period              = 300
  evaluation_periods  = 1
  threshold           = max(each.value * 1000 - 2000, 1000)
  comparison_operator = "GreaterThanOrEqualToThreshold"
  treat_missing_data  = "notBreaching"
  alarm_actions       = var.alarm_actions

  dimensions = {
    FunctionName = each.key
  }

  tags = var.tags
}

resource "aws_cloudwatch_metric_alarm" "fetch_task_dlq_depth" {
  alarm_name          = "${var.name_prefix}-fetch-task-dlq-depth"
  alarm_description   = "Fetch task DLQ has visible failed messages."
  namespace           = "AWS/SQS"
  metric_name         = "ApproximateNumberOfMessagesVisible"
  statistic           = "Maximum"
  period              = 300
  evaluation_periods  = 1
  threshold           = 0
  comparison_operator = "GreaterThanThreshold"
  treat_missing_data  = "notBreaching"
  alarm_actions       = var.alarm_actions

  dimensions = {
    QueueName = var.fetch_task_dlq_name
  }

  tags = var.tags
}

resource "aws_cloudwatch_metric_alarm" "fetch_task_queue_age" {
  alarm_name          = "${var.name_prefix}-fetch-task-queue-age"
  alarm_description   = "Fetch task queue oldest message age is above the low-frequency polling window."
  namespace           = "AWS/SQS"
  metric_name         = "ApproximateAgeOfOldestMessage"
  statistic           = "Maximum"
  period              = 300
  evaluation_periods  = 1
  threshold           = 900
  comparison_operator = "GreaterThanThreshold"
  treat_missing_data  = "notBreaching"
  alarm_actions       = var.alarm_actions

  dimensions = {
    QueueName = var.fetch_task_queue_name
  }

  tags = var.tags
}

resource "aws_cloudwatch_dashboard" "flightaware" {
  dashboard_name = "${var.name_prefix}-flightaware-free-allowance"

  dashboard_body = jsonencode({
    widgets = [
      {
        type   = "metric"
        width  = 12
        height = 6
        properties = {
          title   = "FlightAware Calls And 429s"
          region  = var.aws_region
          view    = "timeSeries"
          stacked = false
          metrics = [
            [var.metric_namespace, "FlightAwareCallCount", "Environment", var.environment],
            [".", "RateLimitedCount", ".", "."]
          ]
        }
      },
      {
        type   = "metric"
        width  = 12
        height = 6
        properties = {
          title   = "Estimated Spend And Stop State"
          region  = var.aws_region
          view    = "timeSeries"
          stacked = false
          metrics = [
            [var.metric_namespace, "EstimatedSpendUSD", "Environment", var.environment],
            [".", "BudgetStopCount", ".", "."],
            [".", "BudgetHardStopCount", ".", "."]
          ]
        }
      },
      {
        type   = "metric"
        width  = 12
        height = 6
        properties = {
          title  = "Lambda Errors And Timeouts"
          region = var.aws_region
          view   = "timeSeries"
          metrics = concat(
            [
              for function_name in keys(var.lambda_timeout_seconds_by_function) :
              ["AWS/Lambda", "Errors", "FunctionName", function_name]
            ],
            [
              for function_name in keys(var.lambda_timeout_seconds_by_function) :
              [".", "Duration", ".", function_name]
            ]
          )
        }
      },
      {
        type   = "metric"
        width  = 12
        height = 6
        properties = {
          title  = "Fetch Queue And DLQ"
          region = var.aws_region
          view   = "timeSeries"
          metrics = [
            ["AWS/SQS", "ApproximateAgeOfOldestMessage", "QueueName", var.fetch_task_queue_name],
            [".", "ApproximateNumberOfMessagesVisible", ".", var.fetch_task_dlq_name]
          ]
        }
      }
    ]
  })
}
