resource "aws_s3_bucket" "geojson" {
  bucket        = var.bucket_name
  force_destroy = var.force_destroy

  tags = var.tags
}

resource "aws_s3_bucket_ownership_controls" "geojson" {
  bucket = aws_s3_bucket.geojson.id

  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

resource "aws_s3_bucket_public_access_block" "geojson" {
  bucket = aws_s3_bucket.geojson.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_server_side_encryption_configuration" "geojson" {
  bucket = aws_s3_bucket.geojson.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "geojson" {
  bucket = aws_s3_bucket.geojson.id

  rule {
    id     = "expire-route-geojson"
    status = "Enabled"

    filter {
      prefix = "routes/"
    }

    expiration {
      days = var.artifact_retention_days
    }
  }

  rule {
    id     = "expire-track-geojson"
    status = "Enabled"

    filter {
      prefix = "tracks/"
    }

    expiration {
      days = var.artifact_retention_days
    }
  }
}
