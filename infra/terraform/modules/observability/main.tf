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

resource "aws_cloudwatch_metric_alarm" "lambda_errors" {
  for_each = toset(var.lambda_function_names)

  alarm_name          = "${var.name_prefix}-${each.value}-errors"
  alarm_description   = "Lambda ${each.value} emitted at least one error."
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
    FunctionName = each.value
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
