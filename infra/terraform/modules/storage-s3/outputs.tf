output "bucket_name" {
  description = "S3 bucket name for route and track GeoJSON artifacts."
  value       = aws_s3_bucket.geojson.bucket
}

output "bucket_arn" {
  description = "S3 bucket ARN for route and track GeoJSON artifacts."
  value       = aws_s3_bucket.geojson.arn
}
