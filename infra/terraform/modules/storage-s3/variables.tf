variable "bucket_name" {
  description = "S3 bucket name for route and track GeoJSON artifacts."
  type        = string
}

variable "artifact_retention_days" {
  description = "Number of days to retain route and track GeoJSON artifacts."
  type        = number
  default     = 30

  validation {
    condition     = var.artifact_retention_days >= 1
    error_message = "artifact_retention_days must be at least 1."
  }
}

variable "force_destroy" {
  description = "Whether Terraform may delete the bucket even when objects remain."
  type        = bool
  default     = false
}

variable "tags" {
  description = "Tags applied to S3 resources."
  type        = map(string)
  default     = {}
}
