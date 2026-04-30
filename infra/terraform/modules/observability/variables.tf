variable "name_prefix" {
  description = "Name prefix for CloudWatch alarms."
  type        = string
}

variable "environment" {
  description = "Environment name used as a custom metric dimension."
  type        = string
}

variable "aws_region" {
  description = "AWS region used by CloudWatch dashboard metric widgets."
  type        = string
}

variable "metric_namespace" {
  description = "CloudWatch namespace for Airpath FlightAware guard metrics."
  type        = string
  default     = "Airpath/FlightAware"
}

variable "lambda_timeout_seconds_by_function" {
  description = "Lambda timeout seconds keyed by function name for error and timeout alarms."
  type        = map(number)

  validation {
    condition = alltrue([
      for timeout_seconds in values(var.lambda_timeout_seconds_by_function) :
      timeout_seconds >= 1 && timeout_seconds <= 900 && floor(timeout_seconds) == timeout_seconds
    ])
    error_message = "lambda_timeout_seconds_by_function values must be integer Lambda timeouts between 1 and 900 seconds."
  }
}

variable "fetch_task_dlq_name" {
  description = "Fetch task dead-letter queue name."
  type        = string
}

variable "fetch_task_queue_name" {
  description = "Fetch task queue name."
  type        = string
}

variable "budget_soft_threshold_usd" {
  description = "Estimated monthly spend threshold that warns before the hard stop."
  type        = number
  default     = 4
}

variable "alarm_actions" {
  description = "Optional CloudWatch alarm action ARNs."
  type        = list(string)
  default     = []
}

variable "tags" {
  description = "Tags applied to CloudWatch alarms."
  type        = map(string)
  default     = {}
}
