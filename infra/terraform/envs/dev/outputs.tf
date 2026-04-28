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
