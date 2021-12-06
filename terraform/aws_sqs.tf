resource "aws_sqs_queue" "pending_orders_queue" {
  name                       = "dcs-pending-orders-queue"
  visibility_timeout_seconds = 30
  message_retention_seconds  = 1209600
}

output "pending_orders_queue_url" {
  value = aws_sqs_queue.pending_orders_queue.url
}
