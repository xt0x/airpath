variable "aws_region" {
  description = "AWS region used for Terraform bootstrap resources."
  type        = string
  default     = "ap-northeast-1"
}

variable "bootstrap_environment" {
  description = "Environment name used for bootstrap resource naming and tags."
  type        = string
  default     = "dev"

  validation {
    condition     = contains(["local", "dev", "stg", "prod"], var.bootstrap_environment)
    error_message = "bootstrap_environment must be one of local, dev, stg, or prod."
  }
}

variable "github_repository" {
  description = "GitHub repository allowed to assume the CI plan role, in owner/name form."
  type        = string
  default     = "example/airpath"

  validation {
    condition     = can(regex("^[^/]+/[^/]+$", var.github_repository))
    error_message = "github_repository must be in owner/name form."
  }
}

variable "github_oidc_subjects" {
  description = "Explicit GitHub OIDC subject claims allowed to assume the CI plan role."
  type        = list(string)
  default     = null

  validation {
    condition = var.github_oidc_subjects == null || (
      length(var.github_oidc_subjects) > 0 &&
      alltrue([
        for subject in var.github_oidc_subjects :
        can(regex("^repo:[^/]+/[^:]+:(ref:refs/heads/.+|pull_request|environment:.+)$", subject))
      ])
    )
    error_message = "github_oidc_subjects must contain repo-scoped branch, pull_request, or environment subjects."
  }
}

variable "github_oidc_thumbprints" {
  description = "SHA-1 thumbprints for the GitHub Actions OIDC provider certificate chain."
  type        = list(string)
  default     = ["6938fd4d98bab03faadb97b34396831e3780aea1"]
}

variable "github_ci_plan_role_name" {
  description = "Optional explicit IAM role name for GitHub Actions Terraform plan jobs."
  type        = string
  default     = null
}

variable "state_bucket_name" {
  description = "Optional explicit Terraform state bucket name. Defaults to an account- and region-scoped Airpath name."
  type        = string
  default     = null
}

variable "terraform_state_force_destroy" {
  description = "Whether Terraform may delete the state bucket even when objects remain. Keep false for shared accounts."
  type        = bool
  default     = false
}

variable "kms_deletion_window_in_days" {
  description = "KMS key deletion window for the Terraform state key."
  type        = number
  default     = 30

  validation {
    condition     = var.kms_deletion_window_in_days >= 7 && var.kms_deletion_window_in_days <= 30
    error_message = "kms_deletion_window_in_days must be between 7 and 30."
  }
}
