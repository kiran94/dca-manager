# Lambda
resource "aws_lambda_function" "process_orders" {
  function_name = "dca-process-orders"
  handler       = "main"
  runtime       = "go1.x"
  role          = aws_iam_role.process_order_iam_role.arn
  description   = "Process DCS Orders from the Queue"

  s3_bucket = aws_s3_bucket.main.bucket
  s3_key    = "${local.lambda_s3_scripts_prefix}/${local.lambda_process_order_object}"
  timeout   = 3

  environment {
    variables = {
      "DCA_BUCKET"                             = aws_s3_bucket.main.bucket
      "DCA_PENDING_ORDER_S3_PREFIX"            = local.lambda_s3_pending_transaction_prefix
      "DCA_PROCESSED_ORDER_S3_PREFIX"          = local.lambda_s3_processed_transaction_prefix,
      "DCA_GLUE_PROCESS_TRANSACTION_JOB"       = aws_glue_job.load_transactions.id
      "DCA_GLUE_PROCESS_TRANSACTION_OPERATION" = "upsert"
    }
  }

  lifecycle {
    ignore_changes = [
      source_code_hash,
      source_code_size,
      layers,
      last_modified
    ]
  }
}


# IAM Role
resource "aws_iam_role" "process_order_iam_role" {
  name = "process_orders_iam_role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = [
          "sts:AssumeRole"
        ]
        Effect = "Allow"
        Sid    = "AssumeLambdaRole"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })

  inline_policy {
    name = "process_orders_iam_role_policy"
    policy = jsonencode({
      Version = "2012-10-17"
      Statement = [
        {
          Action = [
            "s3:GetObject",
            "s3:PutObject",
            "ssm:GetParameter",
            "sns:Publish",
            "sqs:ReceiveMessage",
            "sqs:DeleteMessage",
            "sqs:ChangeMessageVisibility",
            "sqs:GetQueueAttributes"
          ]
          Effect = "Allow"
          Resource = [
            "${aws_s3_bucket.main.arn}",
            "${aws_s3_bucket.main.arn}/*",
            "${aws_ssm_parameter.kraken_api_key.arn}",
            "${aws_ssm_parameter.kraken_api_secret.arn}",
            "${aws_sns_topic.lambda_failure_dlq.arn}",
            "${aws_sqs_queue.pending_orders_queue.arn}"
          ]
        }
      ]
    })
  }
}

resource "aws_iam_policy_attachment" "attach_lambda_basic_execution_role_process_order" {
  name       = "AttachAWSLambdaBasicExecutionRole"
  roles      = [aws_iam_role.process_order_iam_role.name]
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

# Logs
resource "aws_cloudwatch_log_group" "process_orders_log_group" {
  name              = "/aws/lambda/${aws_lambda_function.process_orders.function_name}"
  retention_in_days = local.lambda_cloudwatch_default_retention
}

resource "aws_lambda_function_event_invoke_config" "process_orders_lambda_failure_dlq" {
  function_name                = aws_lambda_function.process_orders.function_name
  maximum_event_age_in_seconds = 60
  maximum_retry_attempts       = 0

  destination_config {
    on_failure {
      destination = aws_sns_topic.lambda_failure_dlq.arn
    }
  }
}

# GitHub Action
resource "github_actions_secret" "aws_lambda_process_orders_key" {
  repository      = github_repository.main.name
  secret_name     = "AWS_LAMBDA_PROCESS_ORDERS_KEY"
  plaintext_value = local.lambda_process_order_object
}

resource "github_actions_secret" "aws_lambda_process_orders_name" {
  repository      = github_repository.main.name
  secret_name     = "AWS_LAMBDA_PROCESS_ORDERS_NAME"
  plaintext_value = aws_lambda_function.process_orders.function_name
}

resource "aws_lambda_event_source_mapping" "source_sqs_to_process_orders" {
  event_source_arn = aws_sqs_queue.pending_orders_queue.arn
  function_name    = aws_lambda_function.process_orders.function_name
}
