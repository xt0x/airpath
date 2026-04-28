resource "aws_secretsmanager_secret" "flightaware_api_key" {
  name        = var.flightaware_api_key_secret_name
  description = "FlightAware AeroAPI key for Airpath. The secret value is managed outside Terraform."

  tags = var.tags
}
