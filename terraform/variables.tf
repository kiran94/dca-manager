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


variable "lambda_failure_dlq_email" {
  type = string
}
