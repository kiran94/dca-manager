locals {
  lambda_s3_scripts_prefix            = "scripts"
  lambda_cloudwatch_default_retention = 7
  lambda_execute_order_object         = "dcs-execute-orders.zip"
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
      "DCA_BUCKET" = aws_s3_bucket.main.bucket
      "DCA_CONFIG" = aws_s3_bucket_object.config.id,
      # "DCA_ALLOW_REAL" = "1"
      "DCA_PENDING_ORDERS_QUEUE_URL" = aws_sqs_queue.pending_orders_queue.url
    }
  }

  lifecycle {
    ignore_changes = [source_code_hash, source_code_size, layers]
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
            "${aws_sqs_queue.pending_orders_queue.arn}"
          ]
        }
      ]
    })
  }
}

resource "aws_iam_policy_attachment" "attach_lambda_basic_execution_role" {
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
  name                = "aws_lambda_execute_orders_schedule"
  description         = "At 6:00 UTC on every Friday"
  schedule_expression = "cron(0 6 ? * FRI *)"
  # schedule_expression = "cron(* 6 ? * FRI *)"
  # https://docs.aws.amazon.com/AmazonCloudWatch/latest/events/ScheduledEvents.html
  # https://docs.aws.amazon.com/lambda/latest/dg/services-cloudwatchevents-expressions.html
}

resource "aws_cloudwatch_event_target" "aws_lambda_execute_orders_schedule_target" {
  target_id = "aws_lambda_execute_orders_schedule_target"
  rule      = aws_cloudwatch_event_rule.aws_lambda_execute_orders_schedule.name
  arn       = aws_lambda_function.execute_orders.arn

  input_transformer {
    input_template = <<EOF
{
  "operation": "ExecuteOrders"
}
EOF
  }
}

resource "aws_lambda_permission" "allow_cloudwatch_to_call_execute_orders" {
  statement_id  = "AllowExecutionFromCloudWatch"
  action        = "lambda:InvokeFunction"
  principal     = "events.amazonaws.com"
  function_name = aws_lambda_function.execute_orders.function_name
  source_arn    = aws_cloudwatch_event_rule.aws_lambda_execute_orders_schedule.arn
}

# Outputs
output "aws_lambda_execute_orders" {
  value = aws_lambda_function.execute_orders.function_name
}
