mock_provider "aws" {
  override_during = plan
}

run "plans_secure_geojson_bucket_defaults" {
  command = plan

  variables {
    bucket_name = "airpath-test-geojson"
    tags = {
      Environment = "test"
      Project     = "airpath"
    }
  }

  assert {
    condition     = aws_s3_bucket.geojson.bucket == "airpath-test-geojson"
    error_message = "The GeoJSON bucket name must come from bucket_name."
  }

  assert {
    condition     = aws_s3_bucket.geojson.force_destroy == false
    error_message = "The GeoJSON bucket must not force destroy objects by default."
  }

  assert {
    condition = (
      aws_s3_bucket_public_access_block.geojson.block_public_acls &&
      aws_s3_bucket_public_access_block.geojson.block_public_policy &&
      aws_s3_bucket_public_access_block.geojson.ignore_public_acls &&
      aws_s3_bucket_public_access_block.geojson.restrict_public_buckets
    )
    error_message = "The GeoJSON bucket must block all public access paths."
  }

  assert {
    condition = alltrue([
      for rule in aws_s3_bucket_server_side_encryption_configuration.geojson.rule :
      rule.apply_server_side_encryption_by_default[0].sse_algorithm == "AES256"
    ])
    error_message = "The GeoJSON bucket must use AES256 server-side encryption."
  }

  assert {
    condition = alltrue([
      for rule in aws_s3_bucket_lifecycle_configuration.geojson.rule :
      rule.status == "Enabled" && rule.expiration[0].days == 30
    ])
    error_message = "All GeoJSON lifecycle rules must be enabled and use the default retention period."
  }

  assert {
    condition = toset([
      for rule in aws_s3_bucket_lifecycle_configuration.geojson.rule :
      rule.filter[0].prefix
    ]) == toset(["routes/", "tracks/"])
    error_message = "GeoJSON lifecycle rules must cover routes/ and tracks/ prefixes."
  }
}

run "uses_configured_retention_and_force_destroy" {
  command = plan

  variables {
    bucket_name             = "airpath-test-geojson"
    artifact_retention_days = 7
    force_destroy           = true
  }

  assert {
    condition     = aws_s3_bucket.geojson.force_destroy == true
    error_message = "force_destroy must follow the caller input."
  }

  assert {
    condition = alltrue([
      for rule in aws_s3_bucket_lifecycle_configuration.geojson.rule :
      rule.expiration[0].days == 7
    ])
    error_message = "GeoJSON lifecycle expiration must follow artifact_retention_days."
  }
}
