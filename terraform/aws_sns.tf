resource "aws_sns_topic" "lambda_failure_dlq" {
  name = "dcs-lambda-failure-dlq"
}

resource "aws_sns_topic_subscription" "lambda_failure_dlq" {
  count = length(var.lambda_failure_dlq_email)

  topic_arn = aws_sns_topic.lambda_failure_dlq.arn
  protocol  = "email"
  endpoint  = var.lambda_failure_dlq_email[count.index]
}
