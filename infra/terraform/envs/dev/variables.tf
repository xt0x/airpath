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
  description = "EventBridge schedule expression for the dev due-flight dispatcher."
  type        = string
  default     = "rate(15 minutes)"
}

variable "geojson_artifact_retention_days" {
  description = "Number of days to retain dev route and track GeoJSON artifacts."
  type        = number
  default     = 30

  validation {
    condition     = var.geojson_artifact_retention_days >= 1
    error_message = "geojson_artifact_retention_days must be at least 1."
  }
}

variable "fetch_task_max_receive_count" {
  description = "Number of receive attempts before a fetch task moves to the DLQ."
  type        = number
  default     = 3

  validation {
    condition     = var.fetch_task_max_receive_count >= 1
    error_message = "fetch_task_max_receive_count must be at least 1."
  }
}

variable "allow_real_flightaware_calls" {
  description = "Explicit opt-in switch for real FlightAware API calls in the personal dev demo. Keep false unless running a limited smoke test."
  type        = bool
  default     = false
}

variable "flightaware_fetch_disabled_reason" {
  description = "Runtime reason reported while real FlightAware calls are disabled."
  type        = string
  default     = "personal_demo_real_calls_disabled"
}

variable "flightaware_api_key_secret_name" {
  description = "Secrets Manager secret name for the FlightAware API key value managed outside Terraform."
  type        = string
  default     = "airpath/dev/flightaware-api-key"
}

variable "personal_demo_notice" {
  description = "Runtime notice shown by services for the personal non-commercial low-frequency demo environment."
  type        = string
  default     = "personal non-commercial low-frequency demo; real FlightAware calls are disabled unless explicitly opted in."
}
