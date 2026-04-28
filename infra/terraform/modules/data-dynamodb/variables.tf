variable "name_prefix" {
  description = "Name prefix for DynamoDB tables."
  type        = string
}

variable "point_in_time_recovery_enabled" {
  description = "Whether point-in-time recovery is enabled for DynamoDB tables."
  type        = bool
  default     = false
}

variable "tags" {
  description = "Tags applied to DynamoDB tables."
  type        = map(string)
  default     = {}
}
