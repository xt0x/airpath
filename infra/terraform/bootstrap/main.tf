data "aws_caller_identity" "current" {}

locals {
  allowed_environments = ["local", "dev", "stg", "prod"]

  state_bucket_name = coalesce(
    var.state_bucket_name,
    "airpath-${var.bootstrap_environment}-terraform-state-${data.aws_caller_identity.current.account_id}-${var.aws_region}"
  )

  github_ci_plan_role_name = coalesce(
    var.github_ci_plan_role_name,
    "airpath-${var.bootstrap_environment}-github-ci-plan"
  )

  state_object_keys = [
    for environment in local.allowed_environments : "${environment}/terraform.tfstate"
  ]

  lock_object_keys = [
    for environment in local.allowed_environments : "${environment}/terraform.tfstate.tflock"
  ]

  common_tags = {
    Project     = "airpath"
    Environment = var.bootstrap_environment
    ManagedBy   = "terraform"
  }
}

data "aws_iam_policy_document" "github_ci_plan_assume_role" {
  statement {
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = [aws_iam_openid_connect_provider.github.arn]
    }

    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:aud"
      values   = ["sts.amazonaws.com"]
    }

    condition {
      test     = "StringLike"
      variable = "token.actions.githubusercontent.com:sub"
      values   = ["repo:${var.github_repository}:*"]
    }
  }
}

resource "aws_iam_openid_connect_provider" "github" {
  url             = "https://token.actions.githubusercontent.com"
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = var.github_oidc_thumbprints

  tags = local.common_tags
}

resource "aws_iam_role" "github_ci_plan" {
  name               = local.github_ci_plan_role_name
  assume_role_policy = data.aws_iam_policy_document.github_ci_plan_assume_role.json

  tags = local.common_tags
}

data "aws_iam_policy_document" "terraform_state_key" {
  statement {
    sid = "EnableAccountRootKeyAdministration"

    actions   = ["kms:*"]
    resources = ["*"]

    principals {
      type        = "AWS"
      identifiers = ["arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"]
    }
  }

  statement {
    sid = "AllowGithubCiPlanStateKeyUse"

    actions = [
      "kms:Decrypt",
      "kms:DescribeKey",
      "kms:Encrypt",
      "kms:GenerateDataKey",
      "kms:ReEncryptFrom",
      "kms:ReEncryptTo",
    ]

    resources = ["*"]

    principals {
      type        = "AWS"
      identifiers = [aws_iam_role.github_ci_plan.arn]
    }
  }
}

resource "aws_kms_key" "terraform_state" {
  description             = "KMS key for Airpath Terraform state encryption."
  deletion_window_in_days = var.kms_deletion_window_in_days
  enable_key_rotation     = true
  policy                  = data.aws_iam_policy_document.terraform_state_key.json

  tags = merge(local.common_tags, {
    Name = "airpath-${var.bootstrap_environment}-terraform-state"
  })
}

resource "aws_kms_alias" "terraform_state" {
  name          = "alias/airpath-${var.bootstrap_environment}-terraform-state"
  target_key_id = aws_kms_key.terraform_state.key_id
}

resource "aws_s3_bucket" "terraform_state" {
  bucket        = local.state_bucket_name
  force_destroy = var.terraform_state_force_destroy

  tags = merge(local.common_tags, {
    Name = local.state_bucket_name
  })
}

resource "aws_s3_bucket_ownership_controls" "terraform_state" {
  bucket = aws_s3_bucket.terraform_state.id

  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

resource "aws_s3_bucket_public_access_block" "terraform_state" {
  bucket = aws_s3_bucket.terraform_state.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_versioning" "terraform_state" {
  bucket = aws_s3_bucket.terraform_state.id

  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "terraform_state" {
  bucket = aws_s3_bucket.terraform_state.id

  rule {
    apply_server_side_encryption_by_default {
      kms_master_key_id = aws_kms_key.terraform_state.arn
      sse_algorithm     = "aws:kms"
    }

    bucket_key_enabled = true
  }
}

data "aws_iam_policy_document" "terraform_state_bucket" {
  statement {
    sid = "DenyInsecureTransport"

    effect = "Deny"

    principals {
      type        = "*"
      identifiers = ["*"]
    }

    actions = ["s3:*"]

    resources = [
      aws_s3_bucket.terraform_state.arn,
      "${aws_s3_bucket.terraform_state.arn}/*",
    ]

    condition {
      test     = "Bool"
      variable = "aws:SecureTransport"
      values   = ["false"]
    }
  }
}

resource "aws_s3_bucket_policy" "terraform_state" {
  bucket = aws_s3_bucket.terraform_state.id
  policy = data.aws_iam_policy_document.terraform_state_bucket.json
}

data "aws_iam_policy_document" "terraform_state_access" {
  statement {
    sid = "ListTerraformStateObjects"

    actions   = ["s3:ListBucket"]
    resources = [aws_s3_bucket.terraform_state.arn]

    condition {
      test     = "StringLike"
      variable = "s3:prefix"
      values   = concat(local.state_object_keys, local.lock_object_keys)
    }
  }

  statement {
    sid = "ReadWriteTerraformStateAndLocks"

    actions = [
      "s3:DeleteObject",
      "s3:GetObject",
      "s3:PutObject",
    ]

    resources = [
      for key in concat(local.state_object_keys, local.lock_object_keys) : "${aws_s3_bucket.terraform_state.arn}/${key}"
    ]
  }

  statement {
    sid = "UseTerraformStateKmsKey"

    actions = [
      "kms:Decrypt",
      "kms:DescribeKey",
      "kms:Encrypt",
      "kms:GenerateDataKey",
      "kms:ReEncryptFrom",
      "kms:ReEncryptTo",
    ]

    resources = [aws_kms_key.terraform_state.arn]
  }
}

resource "aws_iam_policy" "terraform_state_access" {
  name        = "airpath-${var.bootstrap_environment}-terraform-state-access"
  description = "Least-privilege access to Airpath Terraform state and S3 lockfiles."
  policy      = data.aws_iam_policy_document.terraform_state_access.json

  tags = local.common_tags
}

resource "aws_iam_role_policy_attachment" "github_ci_plan_state_access" {
  role       = aws_iam_role.github_ci_plan.name
  policy_arn = aws_iam_policy.terraform_state_access.arn
}

data "aws_iam_policy_document" "terraform_plan_read" {
  statement {
    sid = "ReadOnlyTerraformPlanApis"

    actions = [
      "apigateway:GET",
      "cloudfront:Get*",
      "cloudfront:List*",
      "cloudwatch:Describe*",
      "cloudwatch:Get*",
      "cloudwatch:List*",
      "dynamodb:Describe*",
      "dynamodb:List*",
      "events:Describe*",
      "events:List*",
      "iam:Get*",
      "iam:List*",
      "kms:DescribeKey",
      "kms:ListAliases",
      "kms:ListKeys",
      "lambda:Get*",
      "lambda:List*",
      "logs:Describe*",
      "logs:Get*",
      "logs:List*",
      "s3:Get*",
      "s3:List*",
      "secretsmanager:DescribeSecret",
      "secretsmanager:ListSecrets",
      "sqs:Get*",
      "sqs:List*",
      "sts:GetCallerIdentity",
    ]

    resources = ["*"]
  }
}

resource "aws_iam_policy" "terraform_plan_read" {
  name        = "airpath-${var.bootstrap_environment}-terraform-plan-read"
  description = "Read-only AWS API access used by GitHub Actions Terraform plan jobs."
  policy      = data.aws_iam_policy_document.terraform_plan_read.json

  tags = local.common_tags
}

resource "aws_iam_role_policy_attachment" "github_ci_plan_read" {
  role       = aws_iam_role.github_ci_plan.name
  policy_arn = aws_iam_policy.terraform_plan_read.arn
}
