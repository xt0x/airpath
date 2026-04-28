output "fetch_task_queue_arn" {
  description = "Fetch task SQS queue ARN."
  value       = aws_sqs_queue.fetch_task.arn
}

output "fetch_task_queue_url" {
  description = "Fetch task SQS queue URL."
  value       = aws_sqs_queue.fetch_task.url
}

output "dispatcher_rule_arn" {
  description = "Dispatcher EventBridge rule ARN."
  value       = aws_cloudwatch_event_rule.dispatcher.arn
}
