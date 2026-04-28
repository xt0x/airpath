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
  description = "EventBridge schedule expression for the dispatcher shell."
  type        = string
}

variable "tags" {
  description = "Tags applied to eventing resources."
  type        = map(string)
  default     = {}
}
