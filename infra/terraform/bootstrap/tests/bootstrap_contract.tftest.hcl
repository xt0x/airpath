provider "aws" {
  region                      = "ap-northeast-1"
  access_key                  = "test"
  secret_key                  = "test"
  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_requesting_account_id  = true
  skip_region_validation      = true
}

override_data {
  target = data.aws_caller_identity.current
  values = {
    account_id = "123456789012"
  }
}

override_resource {
  target = aws_iam_openid_connect_provider.github
  values = {
    arn = "arn:aws:iam::123456789012:oidc-provider/token.actions.githubusercontent.com"
  }
}

override_resource {
  target = aws_iam_role.github_ci_plan
  values = {
    arn = "arn:aws:iam::123456789012:role/airpath-dev-github-ci-plan"
  }
}

override_resource {
  target = aws_kms_key.terraform_state
  values = {
    arn    = "arn:aws:kms:ap-northeast-1:123456789012:key/test-terraform-state"
    key_id = "test-terraform-state"
  }
}

override_resource {
  target = aws_s3_bucket.terraform_state
  values = {
    arn = "arn:aws:s3:::airpath-dev-terraform-state-123456789012-ap-northeast-1"
  }
}

run "plans_secure_state_backend_defaults" {
  command = plan

  assert {
    condition = (
      aws_s3_bucket.terraform_state.bucket == "airpath-dev-terraform-state-123456789012-ap-northeast-1" &&
      aws_s3_bucket.terraform_state.force_destroy == false &&
      aws_s3_bucket_versioning.terraform_state.versioning_configuration[0].status == "Enabled"
    )
    error_message = "Bootstrap must create a stable, versioned Terraform state bucket without force destroy by default."
  }

  assert {
    condition = (
      aws_s3_bucket_public_access_block.terraform_state.block_public_acls == true &&
      aws_s3_bucket_public_access_block.terraform_state.block_public_policy == true &&
      aws_s3_bucket_public_access_block.terraform_state.ignore_public_acls == true &&
      aws_s3_bucket_public_access_block.terraform_state.restrict_public_buckets == true
    )
    error_message = "Terraform state bucket public access must remain fully blocked."
  }

  assert {
    condition = (
      one(aws_s3_bucket_server_side_encryption_configuration.terraform_state.rule).apply_server_side_encryption_by_default[0].sse_algorithm == "aws:kms" &&
      one(aws_s3_bucket_server_side_encryption_configuration.terraform_state.rule).bucket_key_enabled == true &&
      aws_kms_key.terraform_state.enable_key_rotation == true
    )
    error_message = "Terraform state must use a rotating KMS key with S3 bucket keys enabled."
  }
}

run "plans_github_oidc_subject_and_state_lock_permissions" {
  command = plan

  variables {
    github_repository = "example/airpath"
  }

  assert {
    condition = (
      aws_iam_openid_connect_provider.github.url == "https://token.actions.githubusercontent.com" &&
      contains(aws_iam_openid_connect_provider.github.client_id_list, "sts.amazonaws.com") &&
      aws_iam_role.github_ci_plan.name == "airpath-dev-github-ci-plan" &&
      local.github_oidc_subjects[0] == "repo:example/airpath:ref:refs/heads/main"
    )
    error_message = "Bootstrap must create the GitHub OIDC provider, deterministic CI plan role, and repository-scoped default subject."
  }

  assert {
    condition = local.state_object_keys == [
      "local/terraform.tfstate",
      "dev/terraform.tfstate",
      "stg/terraform.tfstate",
      "prod/terraform.tfstate",
    ]
    error_message = "CI state access policy inputs must include every environment state object key."
  }

  assert {
    condition = local.lock_object_keys == [
      "local/terraform.tfstate.tflock",
      "dev/terraform.tfstate.tflock",
      "stg/terraform.tfstate.tflock",
      "prod/terraform.tfstate.tflock",
    ]
    error_message = "CI state access policy inputs must include every environment native S3 lockfile key."
  }
}

run "outputs_backend_config_for_all_environment_roots" {
  command = plan

  assert {
    condition = (
      output.backend_config.bucket == "airpath-dev-terraform-state-123456789012-ap-northeast-1" &&
      output.backend_config.region == "ap-northeast-1" &&
      output.backend_config.encrypt == true &&
      output.backend_config.use_lockfile == true &&
      output.backend_config.keys.local == "local/terraform.tfstate" &&
      output.backend_config.keys.dev == "dev/terraform.tfstate" &&
      output.backend_config.keys.stg == "stg/terraform.tfstate" &&
      output.backend_config.keys.prod == "prod/terraform.tfstate"
    )
    error_message = "Bootstrap backend_config output must expose every environment key for remote backend initialization."
  }
}
