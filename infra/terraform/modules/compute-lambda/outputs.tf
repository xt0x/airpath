output "function_name" {
  description = "Lambda function name."
  value       = aws_lambda_function.this.function_name
}

output "function_arn" {
  description = "Lambda function ARN."
  value       = aws_lambda_function.this.arn
}

output "invoke_arn" {
  description = "Lambda invoke ARN."
  value       = aws_lambda_function.this.invoke_arn
}

output "role_arn" {
  description = "Lambda execution role ARN."
  value       = aws_iam_role.this.arn
}

output "role_name" {
  description = "Lambda execution role name."
  value       = aws_iam_role.this.name
}

output "environment_variables" {
  description = "Lambda environment variables planned for the function."
  value       = aws_lambda_function.this.environment[0].variables
}
