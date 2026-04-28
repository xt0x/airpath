output "alarm_names" {
  description = "CloudWatch alarm names keyed by alarm purpose."
  value = merge(
    {
      flightaware_rate_limited          = aws_cloudwatch_metric_alarm.flightaware_rate_limited.alarm_name
      flightaware_budget_stop           = aws_cloudwatch_metric_alarm.flightaware_budget_stop.alarm_name
      flightaware_budget_soft_threshold = aws_cloudwatch_metric_alarm.flightaware_budget_soft_threshold.alarm_name
      flightaware_budget_hard_stop      = aws_cloudwatch_metric_alarm.flightaware_budget_hard_stop.alarm_name
      fetch_task_dlq_depth              = aws_cloudwatch_metric_alarm.fetch_task_dlq_depth.alarm_name
      fetch_task_queue_age              = aws_cloudwatch_metric_alarm.fetch_task_queue_age.alarm_name
    },
    {
      for function_name, alarm in aws_cloudwatch_metric_alarm.lambda_errors :
      "lambda_errors_${function_name}" => alarm.alarm_name
    },
    {
      for function_name, alarm in aws_cloudwatch_metric_alarm.lambda_timeouts :
      "lambda_timeouts_${function_name}" => alarm.alarm_name
    }
  )
}

output "dashboard_name" {
  description = "CloudWatch dashboard name for FlightAware free allowance operations."
  value       = aws_cloudwatch_dashboard.flightaware.dashboard_name
}
