output "environment" {
  description = "Environment managed by this Terraform root."
  value       = var.environment
}

output "http_api_endpoint" {
  description = "HTTP API endpoint for /v1/* routes."
  value       = module.http_api.api_endpoint
}

output "api_lambda_function_name" {
  description = "Go API Lambda shell function name."
  value       = module.api_lambda.function_name
}

output "fetcher_lambda_function_name" {
  description = "Fetcher Lambda shell function name."
  value       = module.fetcher_lambda.function_name
}

output "dispatcher_lambda_function_name" {
  description = "Dispatcher Lambda shell function name."
  value       = module.dispatcher_lambda.function_name
}

output "fetch_task_queue_url" {
  description = "Fetch task queue URL."
  value       = module.fetch_task_queue.fetch_task_queue_url
}

output "fetch_task_dlq_url" {
  description = "Fetch task dead-letter queue URL."
  value       = module.fetch_task_queue.fetch_task_dlq_url
}

output "dynamodb_table_names" {
  description = "DynamoDB table names used by the dev API and worker shells."
  value       = module.data_tables.table_names
}

output "geojson_bucket_name" {
  description = "S3 bucket name for route and track GeoJSON artifacts."
  value       = module.geojson_storage.bucket_name
}

output "flightaware_api_key_secret_arn" {
  description = "Secrets Manager secret ARN for the FlightAware API key."
  value       = module.secret_references.flightaware_api_key_secret_arn
}

output "cloudwatch_alarm_names" {
  description = "CloudWatch alarm names for dev monitoring."
  value       = module.observability.alarm_names
}
