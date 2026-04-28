output "fetch_task_queue_arn" {
  description = "Fetch task SQS queue ARN."
  value       = aws_sqs_queue.fetch_task.arn
}

output "fetch_task_queue_url" {
  description = "Fetch task SQS queue URL."
  value       = aws_sqs_queue.fetch_task.url
}

output "fetch_task_dlq_arn" {
  description = "Fetch task dead-letter queue ARN."
  value       = aws_sqs_queue.fetch_task_dlq.arn
}

output "fetch_task_dlq_name" {
  description = "Fetch task dead-letter queue name."
  value       = aws_sqs_queue.fetch_task_dlq.name
}

output "fetch_task_dlq_url" {
  description = "Fetch task dead-letter queue URL."
  value       = aws_sqs_queue.fetch_task_dlq.url
}

output "dispatcher_rule_arn" {
  description = "Dispatcher EventBridge rule ARN."
  value       = aws_cloudwatch_event_rule.dispatcher.arn
}
