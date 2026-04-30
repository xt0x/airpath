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

variable "tags" {
  description = "Tags applied to eventing resources."
  type        = map(string)
  default     = {}
}
