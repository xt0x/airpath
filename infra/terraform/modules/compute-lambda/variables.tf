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
}

variable "timeout_seconds" {
  description = "Lambda timeout in seconds."
  type        = number
  default     = 10
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
