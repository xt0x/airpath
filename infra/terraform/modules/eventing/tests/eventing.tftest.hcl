mock_provider "aws" {
  override_during = plan
}

override_resource {
  target = aws_sqs_queue.fetch_task
  values = {
    arn = "arn:aws:sqs:us-east-1:123456789012:airpath-test-fetch-task"
    url = "https://sqs.us-east-1.amazonaws.com/123456789012/airpath-test-fetch-task"
  }
}

override_resource {
  target = aws_sqs_queue.fetch_task_dlq
  values = {
    arn = "arn:aws:sqs:us-east-1:123456789012:airpath-test-fetch-task-dlq"
    url = "https://sqs.us-east-1.amazonaws.com/123456789012/airpath-test-fetch-task-dlq"
  }
}

override_resource {
  target = aws_cloudwatch_event_rule.dispatcher
  values = {
    arn = "arn:aws:events:us-east-1:123456789012:rule/airpath-test-dispatcher"
  }
}

override_data {
  target = data.aws_iam_policy_document.fetch_task_consume
  values = {
    json = "{\"Version\":\"2012-10-17\",\"Statement\":[]}"
  }
}

run "plans_fetch_task_queue_and_dispatcher_schedule" {
  command = plan

  variables {
    name_prefix                     = "airpath-test"
    fetcher_lambda_arn              = "arn:aws:lambda:us-east-1:123456789012:function:airpath-test-fetcher"
    fetcher_lambda_role_name        = "airpath-test-fetcher-role"
    dispatcher_lambda_arn           = "arn:aws:lambda:us-east-1:123456789012:function:airpath-test-dispatcher"
    dispatcher_lambda_function_name = "airpath-test-dispatcher"
    dispatcher_schedule_expression  = "rate(5 minutes)"
    tags = {
      Environment = "test"
      Project     = "airpath"
    }
  }

  assert {
    condition     = aws_sqs_queue.fetch_task.name == "airpath-test-fetch-task"
    error_message = "The fetch task queue must use the name prefix."
  }

  assert {
    condition     = aws_sqs_queue.fetch_task_dlq.name == "airpath-test-fetch-task-dlq"
    error_message = "The fetch task DLQ must use the name prefix."
  }

  assert {
    condition = (
      aws_sqs_queue.fetch_task.sqs_managed_sse_enabled == true &&
      aws_sqs_queue.fetch_task_dlq.sqs_managed_sse_enabled == true
    )
    error_message = "Fetch task queues must use SQS-managed server-side encryption."
  }

  assert {
    condition = (
      aws_lambda_event_source_mapping.fetcher.function_name == "arn:aws:lambda:us-east-1:123456789012:function:airpath-test-fetcher" &&
      aws_lambda_event_source_mapping.fetcher.batch_size == 1 &&
      aws_lambda_event_source_mapping.fetcher.function_response_types == toset(["ReportBatchItemFailures"]) &&
      aws_lambda_event_source_mapping.fetcher.enabled == true
    )
    error_message = "The fetcher event source mapping must consume fetch tasks one at a time with partial batch response."
  }

  assert {
    condition = (
      aws_cloudwatch_event_rule.dispatcher.name == "airpath-test-dispatcher" &&
      aws_cloudwatch_event_rule.dispatcher.schedule_expression == "rate(5 minutes)"
    )
    error_message = "The dispatcher EventBridge rule must use the configured name prefix and schedule."
  }

  assert {
    condition = (
      aws_cloudwatch_event_target.dispatcher.rule == aws_cloudwatch_event_rule.dispatcher.name &&
      aws_cloudwatch_event_target.dispatcher.target_id == "dispatcher-lambda" &&
      aws_cloudwatch_event_target.dispatcher.arn == "arn:aws:lambda:us-east-1:123456789012:function:airpath-test-dispatcher"
    )
    error_message = "The dispatcher target must invoke the configured dispatcher Lambda."
  }
}

run "uses_configured_fetch_task_max_receive_count" {
  command = apply

  variables {
    name_prefix                     = "airpath-test"
    fetcher_lambda_arn              = "arn:aws:lambda:us-east-1:123456789012:function:airpath-test-fetcher"
    fetcher_lambda_role_name        = "airpath-test-fetcher-role"
    dispatcher_lambda_arn           = "arn:aws:lambda:us-east-1:123456789012:function:airpath-test-dispatcher"
    dispatcher_lambda_function_name = "airpath-test-dispatcher"
    dispatcher_schedule_expression  = "rate(5 minutes)"
    fetch_task_max_receive_count    = 5
  }

  assert {
    condition = (
      aws_lambda_event_source_mapping.fetcher.event_source_arn == aws_sqs_queue.fetch_task.arn &&
      jsondecode(aws_sqs_queue.fetch_task.redrive_policy).maxReceiveCount == 5
    )
    error_message = "The fetch task mapping and DLQ threshold must follow queue configuration."
  }

  assert {
    condition     = jsondecode(aws_sqs_queue.fetch_task.redrive_policy).deadLetterTargetArn == aws_sqs_queue.fetch_task_dlq.arn
    error_message = "The fetch task queue must redrive to its DLQ."
  }
}
