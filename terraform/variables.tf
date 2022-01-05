// SCHEDULES
variable "execute_orders_schedules" {
  type = list(object({
    description         = string
    schedule_expression = string
    # https://docs.aws.amazon.com/AmazonCloudWatch/latest/events/ScheduledEvents.html
    # https://docs.aws.amazon.com/lambda/latest/dg/services-cloudwatchevents-expressions.html
  }))

  description = "The schedule in which to execute orders"
  default = [
    {
      description         = "At 6:00 UTC on every Friday"
      schedule_expression = "cron(0 6 ? * FRI *)"
    },
    {
      description         = "At 19:45 UTC on every Wednesday"
      schedule_expression = "cron(45 19 ? * WED *)"
    }
  ]
}

variable "lambda_timeout_seconds" {
  type    = number
  default = 3
}

// ALERTS
variable "lambda_failure_dlq_email" {
  type        = list(string)
  description = "The Email to notify when a failed lambda execution completes"
}

variable "lambda_success_email" {
  type        = list(string)
  description = "The Email to notify when a successful lambda execution completes"
}

// SECRETS
// Override with TF_VAR_
variable "KRAKEN_API_KEY" {
  description = "The Kraken API Key"
  default     = "dummy"
}
variable "KRAKEN_API_SECRET" {
  description = "The Kraken API Secret"
  default     = "dummy"
}

// GLUE
variable "glue_connections" {
  type        = list(string)
  description = "The AWS Glue Connector for Apache Hudi"
  default     = ["hudi-connection3"]
}
