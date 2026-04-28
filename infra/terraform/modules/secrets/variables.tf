variable "flightaware_api_key_secret_name" {
  description = "Secrets Manager secret name that stores the FlightAware API key value outside Terraform."
  type        = string
}

variable "tags" {
  description = "Tags applied to secret metadata."
  type        = map(string)
  default     = {}
}
