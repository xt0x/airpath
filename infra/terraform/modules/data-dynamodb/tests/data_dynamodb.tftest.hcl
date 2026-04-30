mock_provider "aws" {
  override_during = plan
}

run "plans_table_names_keys_gsi_and_recovery" {
  command = plan

  variables {
    name_prefix                    = "airpath-test"
    point_in_time_recovery_enabled = true
    tags = {
      Environment = "test"
      Project     = "airpath"
    }
  }

  assert {
    condition = (
      aws_dynamodb_table.flights.name == "airpath-test-flights" &&
      aws_dynamodb_table.flight_lookup.name == "airpath-test-flight-lookup" &&
      aws_dynamodb_table.flight_positions.name == "airpath-test-flight-positions" &&
      aws_dynamodb_table.usage_budget.name == "airpath-test-usage-budget"
    )
    error_message = "DynamoDB table names must follow the module name prefix contract."
  }

  assert {
    condition = (
      aws_dynamodb_table.flights.hash_key == "flightId" &&
      aws_dynamodb_table.flight_lookup.hash_key == "lookupType" &&
      aws_dynamodb_table.flight_lookup.range_key == "lookupKey" &&
      aws_dynamodb_table.flight_positions.hash_key == "flightId" &&
      aws_dynamodb_table.flight_positions.range_key == "timestamp" &&
      aws_dynamodb_table.usage_budget.hash_key == "budgetScope"
    )
    error_message = "DynamoDB table key schema must match the runtime storage contract."
  }

  assert {
    condition = anytrue([
      for index in aws_dynamodb_table.flights.global_secondary_index :
      index.name == "poll-due-index" &&
      index.hash_key == "pollShard" &&
      index.range_key == "nextPollAt" &&
      index.projection_type == "ALL"
    ])
    error_message = "The flights table must expose the poll-due-index GSI."
  }

  assert {
    condition = alltrue([
      for table in [
        aws_dynamodb_table.flights,
        aws_dynamodb_table.flight_lookup,
        aws_dynamodb_table.flight_positions,
        aws_dynamodb_table.usage_budget,
      ] :
      table.billing_mode == "PAY_PER_REQUEST" &&
      table.point_in_time_recovery[0].enabled == true
    ])
    error_message = "DynamoDB tables must use on-demand billing and the configured point-in-time recovery setting."
  }

  assert {
    condition = alltrue([
      for table in [
        aws_dynamodb_table.flights,
        aws_dynamodb_table.flight_lookup,
        aws_dynamodb_table.flight_positions,
      ] :
      table.ttl[0].attribute_name == "ttl" && table.ttl[0].enabled == true
    ])
    error_message = "Runtime data tables must enable ttl on the ttl attribute."
  }
}

run "keeps_point_in_time_recovery_disabled_by_default" {
  command = plan

  variables {
    name_prefix = "airpath-test"
  }

  assert {
    condition = alltrue([
      for table in [
        aws_dynamodb_table.flights,
        aws_dynamodb_table.flight_lookup,
        aws_dynamodb_table.flight_positions,
        aws_dynamodb_table.usage_budget,
      ] :
      table.point_in_time_recovery[0].enabled == false
    ])
    error_message = "DynamoDB point-in-time recovery must remain disabled by default."
  }
}
