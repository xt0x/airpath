resource "aws_sqs_queue" "fetch_task_dlq" {
  name                      = "${var.name_prefix}-fetch-task-dlq"
  message_retention_seconds = 1209600
  sqs_managed_sse_enabled   = true

  tags = var.tags
}

resource "aws_sqs_queue" "fetch_task" {
  name                       = "${var.name_prefix}-fetch-task"
  visibility_timeout_seconds = var.fetch_task_visibility_timeout_seconds
  message_retention_seconds  = 1209600
  sqs_managed_sse_enabled    = true

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.fetch_task_dlq.arn
    maxReceiveCount     = var.fetch_task_max_receive_count
  })

  tags = var.tags
}

resource "aws_lambda_event_source_mapping" "fetcher" {
  event_source_arn        = aws_sqs_queue.fetch_task.arn
  function_name           = var.fetcher_lambda_arn
  batch_size              = 1
  function_response_types = ["ReportBatchItemFailures"]
  enabled                 = true

  depends_on = [aws_iam_role_policy.fetcher_queue_consume]
}

resource "aws_cloudwatch_event_rule" "dispatcher" {
  name                = "${var.name_prefix}-dispatcher"
  description         = "Low-frequency dispatcher for due-flight selection."
  schedule_expression = var.dispatcher_schedule_expression

  tags = var.tags
}

resource "aws_cloudwatch_event_target" "dispatcher" {
  rule      = aws_cloudwatch_event_rule.dispatcher.name
  target_id = "dispatcher-lambda"
  arn       = var.dispatcher_lambda_arn
}

resource "aws_lambda_permission" "allow_dispatcher_eventbridge" {
  statement_id  = "AllowExecutionFromEventBridge"
  action        = "lambda:InvokeFunction"
  function_name = var.dispatcher_lambda_function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.dispatcher.arn
}

data "aws_iam_policy_document" "fetch_task_consume" {
  statement {
    actions = [
      "sqs:ChangeMessageVisibility",
      "sqs:DeleteMessage",
      "sqs:GetQueueAttributes",
      "sqs:ReceiveMessage",
    ]

    resources = [aws_sqs_queue.fetch_task.arn]
  }
}

resource "aws_iam_role_policy" "fetcher_queue_consume" {
  name   = "${var.name_prefix}-fetcher-queue-consume"
  role   = var.fetcher_lambda_role_name
  policy = data.aws_iam_policy_document.fetch_task_consume.json
}
