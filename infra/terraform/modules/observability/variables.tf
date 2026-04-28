variable "name_prefix" {
  description = "Name prefix for CloudWatch alarms."
  type        = string
}

variable "environment" {
  description = "Environment name used as a custom metric dimension."
  type        = string
}

variable "metric_namespace" {
  description = "CloudWatch namespace for Airpath FlightAware guard metrics."
  type        = string
  default     = "Airpath/FlightAware"
}

variable "lambda_function_names" {
  description = "Lambda function names monitored for errors."
  type        = list(string)
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
