output "flightaware_api_key_secret_name" {
  description = "Secrets Manager secret name for the FlightAware API key."
  value       = aws_secretsmanager_secret.flightaware_api_key.name
}

output "flightaware_api_key_secret_arn" {
  description = "Secrets Manager secret ARN for the FlightAware API key."
  value       = aws_secretsmanager_secret.flightaware_api_key.arn
}
