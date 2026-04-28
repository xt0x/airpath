output "state_bucket_name" {
  description = "S3 bucket name for Terraform state."
  value       = aws_s3_bucket.terraform_state.bucket
}

output "state_bucket_arn" {
  description = "S3 bucket ARN for Terraform state."
  value       = aws_s3_bucket.terraform_state.arn
}

output "state_kms_key_arn" {
  description = "KMS key ARN used for Terraform state encryption."
  value       = aws_kms_key.terraform_state.arn
}

output "state_kms_key_id" {
  description = "KMS key ID used for Terraform state encryption."
  value       = aws_kms_key.terraform_state.key_id
}

output "github_oidc_provider_arn" {
  description = "GitHub Actions OIDC provider ARN."
  value       = aws_iam_openid_connect_provider.github.arn
}

output "github_ci_plan_role_arn" {
  description = "IAM role ARN for GitHub Actions Terraform plan jobs."
  value       = aws_iam_role.github_ci_plan.arn
}

output "backend_config" {
  description = "Values used with terraform init -backend-config for environment roots."
  value = {
    bucket       = aws_s3_bucket.terraform_state.bucket
    region       = var.aws_region
    encrypt      = true
    use_lockfile = true
    kms_key_id   = aws_kms_key.terraform_state.arn
    keys = {
      for environment in local.allowed_environments : environment => "${environment}/terraform.tfstate"
    }
  }
}
