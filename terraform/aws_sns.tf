// FAILURE NOTFICATIONS
resource "aws_sns_topic" "lambda_failure_dlq" {
  name = "dca-lambda-failure-dlq"
}

resource "aws_sns_topic_subscription" "lambda_failure_dlq" {
  count = length(var.lambda_failure_dlq_email)

  topic_arn = aws_sns_topic.lambda_failure_dlq.arn
  protocol  = "email"
  endpoint  = var.lambda_failure_dlq_email[count.index]
}

// SUCCESS NOTFICATIONS
resource "aws_sns_topic" "lambda_success" {
  name = "dca-lambda-sucess"
}

resource "aws_sns_topic_subscription" "lambda_success" {
  count = length(var.lambda_success_email)

  topic_arn = aws_sns_topic.lambda_success.arn
  protocol  = "email"
  endpoint  = var.lambda_success_email[count.index]
}
