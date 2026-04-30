run "default_environment_is_stg" {
  command = plan

  assert {
    condition     = output.environment == "stg"
    error_message = "The stg root must default to the stg environment."
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
