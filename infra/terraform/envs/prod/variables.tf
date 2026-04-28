variable "aws_region" {
  description = "AWS region for the prod environment."
  type        = string
  default     = "ap-northeast-1"
}

variable "environment" {
  description = "Deployment environment name."
  type        = string
  default     = "prod"

  validation {
    condition     = contains(["prod"], var.environment)
    error_message = "This directory only manages the prod environment."
  }
}

