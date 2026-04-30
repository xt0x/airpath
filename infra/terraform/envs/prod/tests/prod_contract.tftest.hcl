run "default_environment_is_prod" {
  command = plan

  assert {
    condition     = output.environment == "prod"
    error_message = "The prod root must default to the prod environment."
  }
}

run "rejects_other_environment_names" {
  command = plan

  variables {
    environment = "dev"
  }

  expect_failures = [
    var.environment,
  ]
}
