variable "name" {
  description = "HTTP API name."
  type        = string
}

variable "api_lambda_invoke_arn" {
  description = "Invoke ARN for the Go API Lambda."
  type        = string
}

variable "api_lambda_function_name" {
  description = "Function name for the Go API Lambda."
  type        = string
}

variable "stage_name" {
  description = "HTTP API stage name."
  type        = string
  default     = "$default"
}

variable "tags" {
  description = "Tags applied to HTTP API resources."
  type        = map(string)
  default     = {}
}
