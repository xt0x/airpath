variable "aws_region" {
  description = "AWS region for the stg environment."
  type        = string
  default     = "ap-northeast-1"
}

variable "environment" {
  description = "Deployment environment name."
  type        = string
  default     = "stg"

  validation {
    condition     = contains(["stg"], var.environment)
    error_message = "This directory only manages the stg environment."
  }
}

