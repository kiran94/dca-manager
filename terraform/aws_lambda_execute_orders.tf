locals {
  lambda_s3_scripts_prefix               = "scripts"
  lambda_cloudwatch_default_retention    = 7
  lambda_execute_order_object            = "dca-execute-orders.zip"
  lambda_process_order_object            = "dca-process-orders.zip"
  lambda_s3_pending_transaction_prefix   = "transactions/status=pending"
  lambda_s3_processed_transaction_prefix = "transactions/status=complete"
}

# Lambda
resource "aws_lambda_function" "execute_orders" {
  function_name = "dcs-execute-orders"
  handler       = "main"
  runtime       = "go1.x"
  role          = aws_iam_role.execute_orders_iam_role.arn
  description   = "Executes Orders from the DCS Configuration"


  s3_bucket = aws_s3_bucket.main.bucket
  s3_key    = "${local.lambda_s3_scripts_prefix}/${local.lambda_execute_order_object}"
  timeout   = 3

  environment {
    variables = {
      "DCA_BUCKET"                   = aws_s3_bucket.main.bucket
      "DCA_CONFIG"                   = aws_s3_bucket_object.config.id,
      "DCA_ALLOW_REAL"               = "1"
      "DCA_PENDING_ORDERS_QUEUE_URL" = aws_sqs_queue.pending_orders_queue.url,
      "DCA_PENDING_ORDER_S3_PREFIX"  = local.lambda_s3_pending_transaction_prefix,
      "DCA_OPERATION"                = "EXECUTE_ORDERS"
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

resource "aws_lambda_function_event_invoke_config" "lambda_failure_dlq" {
  function_name                = aws_lambda_function.execute_orders.function_name
  maximum_event_age_in_seconds = 60
  maximum_retry_attempts       = 0

  destination_config {
    on_failure {
      destination = aws_sns_topic.lambda_failure_dlq.arn
    }

    on_success {
      destination = aws_sns_topic.lambda_success.arn
    }
  }
}

# IAM Role
resource "aws_iam_role" "execute_orders_iam_role" {
  name = "execute_orders_iam_role"

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
    name = "execute_orders_iam_role_policy"
    policy = jsonencode({
      Version = "2012-10-17"
      Statement = [
        {
          Action = [
            "s3:GetObject",
            "s3:PutObject",
            "ssm:GetParameter",
            "sns:Publish",
            "sqs:SendMessage",
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
            "${aws_sqs_queue.pending_orders_queue.arn}",
            "${aws_sns_topic.lambda_failure_dlq.arn}",
            "${aws_sns_topic.lambda_success.arn}"
          ]
        }
      ]
    })
  }
}

resource "aws_iam_policy_attachment" "attach_lambda_basic_execution_role_execute_order" {
  name       = "AttachAWSLambdaBasicExecutionRole"
  roles      = [aws_iam_role.execute_orders_iam_role.name]
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

# Logs
resource "aws_cloudwatch_log_group" "execute_orders_log_group" {
  name              = "/aws/lambda/${aws_lambda_function.execute_orders.function_name}"
  retention_in_days = local.lambda_cloudwatch_default_retention
}

# GitHub Action
resource "github_actions_secret" "aws_lambda_execute_orders_key" {
  repository      = github_repository.main.name
  secret_name     = "AWS_LAMBDA_EXECUTE_ORDERS_KEY"
  plaintext_value = local.lambda_execute_order_object
}

resource "github_actions_secret" "aws_lambda_execute_orders_name" {
  repository      = github_repository.main.name
  secret_name     = "AWS_LAMBDA_EXECUTE_ORDERS_NAME"
  plaintext_value = aws_lambda_function.execute_orders.function_name
}

# Triggers
resource "aws_cloudwatch_event_rule" "aws_lambda_execute_orders_schedule" {
  count = length(var.execute_orders_schedules)

  name                = "aws_lambda_execute_orders_schedule_${count.index}"
  description         = var.execute_orders_schedules[count.index].description
  schedule_expression = var.execute_orders_schedules[count.index].schedule_expression
}

resource "aws_cloudwatch_event_target" "aws_lambda_execute_orders_schedule_target" {
  count = length(var.execute_orders_schedules)

  target_id = "aws_lambda_execute_orders_schedule_target"
  rule      = aws_cloudwatch_event_rule.aws_lambda_execute_orders_schedule[count.index].name
  arn       = aws_lambda_function.execute_orders.arn

  retry_policy {
    maximum_retry_attempts       = 0
    maximum_event_age_in_seconds = 60
  }

  input_transformer {
    input_template = <<EOF
{
  "operation": "ExecuteOrders"
}
EOF
  }
}

resource "aws_lambda_permission" "allow_cloudwatch_to_call_execute_orders" {
  count = length(var.execute_orders_schedules)

  statement_id  = "AllowExecutionFromCloudWatch_${count.index}"
  action        = "lambda:InvokeFunction"
  principal     = "events.amazonaws.com"
  function_name = aws_lambda_function.execute_orders.function_name
  source_arn    = aws_cloudwatch_event_rule.aws_lambda_execute_orders_schedule[count.index].arn
}

# Outputs
output "aws_lambda_execute_orders" {
  value = aws_lambda_function.execute_orders.function_name
}

output "aws_lambda_pending_order_path" {
  value = local.lambda_s3_pending_transaction_prefix
}

output "aws_lambda_processed_order_path" {
  value = local.lambda_s3_processed_transaction_prefix
}
