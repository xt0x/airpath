mock_provider "aws" {
  override_during = plan
}

run "plans_secret_metadata_without_secret_value" {
  command = plan

  variables {
    flightaware_api_key_secret_name = "airpath/test/flightaware-api-key"
    tags = {
      Environment = "test"
      Project     = "airpath"
    }
  }

  assert {
    condition = (
      aws_secretsmanager_secret.flightaware_api_key.name == "airpath/test/flightaware-api-key" &&
      strcontains(aws_secretsmanager_secret.flightaware_api_key.description, "managed outside Terraform")
    )
    error_message = "The secrets module must manage only FlightAware secret metadata."
  }

  assert {
    condition     = output.flightaware_api_key_secret_name == "airpath/test/flightaware-api-key"
    error_message = "The secrets module output must expose the secret name reference only."
  }
}
