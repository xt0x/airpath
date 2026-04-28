output "table_names" {
  description = "DynamoDB table names keyed by logical table."
  value = {
    flights          = aws_dynamodb_table.flights.name
    flight_lookup    = aws_dynamodb_table.flight_lookup.name
    flight_positions = aws_dynamodb_table.flight_positions.name
    usage_budget     = aws_dynamodb_table.usage_budget.name
  }
}

output "table_arns" {
  description = "DynamoDB table ARNs keyed by logical table."
  value = {
    flights          = aws_dynamodb_table.flights.arn
    flight_lookup    = aws_dynamodb_table.flight_lookup.arn
    flight_positions = aws_dynamodb_table.flight_positions.arn
    usage_budget     = aws_dynamodb_table.usage_budget.arn
  }
}
