variable "function_name" {
  description = "Lambda function name."
  type        = string
}

variable "description" {
  description = "Lambda function description."
  type        = string
  default     = null
}

variable "runtime" {
  description = "Lambda runtime."
  type        = string
  default     = "provided.al2023"
}

variable "handler" {
  description = "Lambda handler."
  type        = string
  default     = "bootstrap"
}

variable "artifact_path" {
  description = "Path to the deployable Lambda zip artifact."
  type        = string
}

variable "memory_size" {
  description = "Lambda memory size in MB."
  type        = number
  default     = 128

  validation {
    condition     = var.memory_size >= 128 && var.memory_size <= 10240 && floor(var.memory_size) == var.memory_size
    error_message = "memory_size must be an integer between 128 and 10240 MB."
  }
}

variable "timeout_seconds" {
  description = "Lambda timeout in seconds."
  type        = number
  default     = 10

  validation {
    condition     = var.timeout_seconds >= 1 && var.timeout_seconds <= 900 && floor(var.timeout_seconds) == var.timeout_seconds
    error_message = "timeout_seconds must be an integer between 1 and 900 seconds."
  }
}

variable "environment_variables" {
  description = "Non-secret Lambda environment variables."
  type        = map(string)
  default     = {}
}

variable "policy_json" {
  description = "Optional IAM policy JSON attached to the Lambda execution role."
  type        = string
  default     = null
}

variable "tags" {
  description = "Tags applied to Lambda resources."
  type        = map(string)
  default     = {}
}
