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

variable "access_log_retention_days" {
  description = "CloudWatch log retention period in days for HTTP API access logs."
  type        = number
  default     = 30

  validation {
    condition     = contains([1, 3, 5, 7, 14, 30, 60, 90, 120, 150, 180, 365, 400, 545, 731, 1096, 1827, 2192, 2557, 2922, 3288, 3653], var.access_log_retention_days)
    error_message = "access_log_retention_days must be a CloudWatch Logs supported retention value."
  }
}

variable "throttling_burst_limit" {
  description = "Default HTTP API stage throttling burst limit."
  type        = number
  default     = 20

  validation {
    condition     = var.throttling_burst_limit >= 1 && floor(var.throttling_burst_limit) == var.throttling_burst_limit
    error_message = "throttling_burst_limit must be a positive integer."
  }
}

variable "throttling_rate_limit" {
  description = "Default HTTP API stage throttling rate limit per second."
  type        = number
  default     = 10

  validation {
    condition     = var.throttling_rate_limit > 0
    error_message = "throttling_rate_limit must be positive."
  }
}

variable "tags" {
  description = "Tags applied to HTTP API resources."
  type        = map(string)
  default     = {}
}
