variable "name_prefix" {
  description = "Name prefix for eventing resources."
  type        = string
}

variable "fetcher_lambda_arn" {
  description = "Fetcher Lambda ARN."
  type        = string
}

variable "fetcher_lambda_role_name" {
  description = "Fetcher Lambda execution role name."
  type        = string
}

variable "dispatcher_lambda_arn" {
  description = "Dispatcher Lambda ARN."
  type        = string
}

variable "dispatcher_lambda_function_name" {
  description = "Dispatcher Lambda function name."
  type        = string
}

variable "dispatcher_schedule_expression" {
  description = "EventBridge schedule expression for the dispatcher."
  type        = string
}

variable "fetch_task_max_receive_count" {
  description = "Number of receive attempts before a fetch task is moved to the DLQ."
  type        = number
  default     = 3

  validation {
    condition     = var.fetch_task_max_receive_count >= 1
    error_message = "fetch_task_max_receive_count must be at least 1."
  }
}

variable "fetch_task_visibility_timeout_seconds" {
  description = "Fetch task SQS visibility timeout in seconds. Set this above the fetcher Lambda timeout."
  type        = number
  default     = 60

  validation {
    condition     = var.fetch_task_visibility_timeout_seconds >= 1 && var.fetch_task_visibility_timeout_seconds <= 43200 && floor(var.fetch_task_visibility_timeout_seconds) == var.fetch_task_visibility_timeout_seconds
    error_message = "fetch_task_visibility_timeout_seconds must be an integer between 1 and 43200."
  }
}

variable "tags" {
  description = "Tags applied to eventing resources."
  type        = map(string)
  default     = {}
}
