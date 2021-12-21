// SCHEDULES
variable "execute_orders_schedules" {
  type = list(object({
    description         = string
    schedule_expression = string
    # https://docs.aws.amazon.com/AmazonCloudWatch/latest/events/ScheduledEvents.html
    # https://docs.aws.amazon.com/lambda/latest/dg/services-cloudwatchevents-expressions.html
  }))

  default = [
    {
      description         = "At 6:00 UTC on every Friday"
      schedule_expression = "cron(0 6 ? * FRI *)"
    }
  ]
}

// ALERTS
variable "lambda_failure_dlq_email" {
  type = list(string)
}

variable "lambda_success_email" {
  type = list(string)
}

// SECRETS
// Override with TF_VAR_
variable "KRAKEN_API_KEY" {
  default = "dummy"
}
variable "KRAKEN_API_SECRET" {
  default = "dummy"
}

// ANALYTICS
variable "enable_analytics" {
  type        = bool
  description = "Enables Glue/Hudi Infrastructure"
  default     = true
}

variable "glue_connections" {
  type        = list(string)
  description = "The AWS Glue Connector for Apache Hudi"
  default     = ["hudi-connection3"]
}
