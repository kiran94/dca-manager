resource "aws_sqs_queue" "pending_orders_queue" {
  name                       = "dca-pending-orders-queue"
  visibility_timeout_seconds = 30
  message_retention_seconds  = 1209600
}

output "pending_orders_queue_arn" {
  value = aws_sqs_queue.pending_orders_queue.arn
}
