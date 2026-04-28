resource "aws_dynamodb_table" "flights" {
  name         = "${var.name_prefix}-flights"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "flightId"

  attribute {
    name = "flightId"
    type = "S"
  }

  ttl {
    attribute_name = "ttl"
    enabled        = true
  }

  point_in_time_recovery {
    enabled = var.point_in_time_recovery_enabled
  }

  tags = var.tags
}

resource "aws_dynamodb_table" "flight_lookup" {
  name         = "${var.name_prefix}-flight-lookup"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "lookupKey"

  attribute {
    name = "lookupKey"
    type = "S"
  }

  ttl {
    attribute_name = "ttl"
    enabled        = true
  }

  point_in_time_recovery {
    enabled = var.point_in_time_recovery_enabled
  }

  tags = var.tags
}

resource "aws_dynamodb_table" "flight_positions" {
  name         = "${var.name_prefix}-flight-positions"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "flightId"
  range_key    = "timestamp"

  attribute {
    name = "flightId"
    type = "S"
  }

  attribute {
    name = "timestamp"
    type = "S"
  }

  ttl {
    attribute_name = "ttl"
    enabled        = true
  }

  point_in_time_recovery {
    enabled = var.point_in_time_recovery_enabled
  }

  tags = var.tags
}

resource "aws_dynamodb_table" "usage_budget" {
  name         = "${var.name_prefix}-usage-budget"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "budgetScope"

  attribute {
    name = "budgetScope"
    type = "S"
  }

  point_in_time_recovery {
    enabled = var.point_in_time_recovery_enabled
  }

  tags = var.tags
}
