mock_provider "aws" {
  override_during = plan
}

run "rejects_zero_artifact_retention_days" {
  command = plan

  variables {
    bucket_name             = "airpath-test-geojson"
    artifact_retention_days = 0
  }

  expect_failures = [
    var.artifact_retention_days,
  ]
}
