variable "aws_region" {
  description = "AWS region for the dev environment."
  type        = string
  default     = "ap-northeast-1"
}

variable "environment" {
  description = "Deployment environment name."
  type        = string
  default     = "dev"

  validation {
    condition     = contains(["dev"], var.environment)
    error_message = "This directory only manages the dev environment."
  }
}

variable "api_lambda_artifact_path" {
  description = "Local path to the API Lambda zip artifact."
  type        = string
  default     = "../../../artifacts/dev/api-lambda.zip"
}

variable "fetcher_lambda_artifact_path" {
  description = "Local path to the fetcher Lambda zip artifact."
  type        = string
  default     = "../../../artifacts/dev/fetcher-lambda.zip"
}

variable "dispatcher_lambda_artifact_path" {
  description = "Local path to the dispatcher Lambda zip artifact."
  type        = string
  default     = "../../../artifacts/dev/dispatcher-lambda.zip"
}

variable "dispatcher_schedule_expression" {
  description = "EventBridge schedule expression for the dev no-op dispatcher shell."
  type        = string
  default     = "rate(15 minutes)"
}
