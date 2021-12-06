# dca-manager

[![main](https://github.com/kiran94/dca-manager/actions/workflows/main.yml/badge.svg)](https://github.com/kiran94/dca-manager/actions/workflows/main.yml)

Dollar Cost Average Manager

<!-- toc GFM -->

* [Getting Started](#getting-started)
    * [Infrastructure](#infrastructure)
    * [Code](#code)
    * [Running](#running)
* [Configuration](#configuration)
* [Schedules](#schedules)

<!-- /toc -->

## Getting Started

### Infrastructure

Assuming you are at the root of the repository.

```sh
export TF_VAR_KRAKEN_API_KEY=your_key
export TF_VAR_KRAKEN_API_SECRET=your_secret
export TF_VAR_lambda_failure_dlq_email='["you@email.com"]'
export TF_VAR_lambda_success_email='["you@email.com"]'

cd terraform

# The aws and github providers require authentication
# https://registry.terraform.io/providers/hashicorp/aws/latest/docs#authentication
# https://registry.terraform.io/providers/integrations/github/latest/docs#authentication

# The terraform backend configured will also need to be updated
# The aws region may also need to be updated

terraform init

# Apply Infrastructure
terraform plan
terraform apply

# Add remote repository to local
git remote add origin $(terraform output -raw github_repository_ssh_clone_url)
```

*Note some coordination might be required here. Where the lambda function cannot be created without the zip file existing in the S3 location first. Therefore this may need to be at least uploaded to S3 first before `terraform apply` will complete successfully.*

### Code

Assuming you are at the root of the repository.

```sh
export DCA_BUCKET=$(terraform -chdir=terraform output -raw bucket)
export DCA_CONFIG=$(terraform -chdir=terraform output -raw config_path)
export DCA_PENDING_ORDERS_QUEUE_URL=$(terraform -chdir=terraform output -raw pending_orders_queue_url)
export DCA_PENDING_ORDER_S3_PREFIX=$(terraform -chdir=terraform output -raw aws_lambda_pending_order_path)

make

# for debugging
make debug
```

### Running

Once the infrastructure is up you can either run the code locally or via lambda. This repository consists of multiple lambdas, you can run them locally like so:

```sh
go run lambda/execute_orders/main.go
go run lambda/process_orders/main.go
```

This will pull data from a combination of sources such as the environment and SSM. Additionally the config uploaded in S3 will be used to determine what to do.

By default, dca-manager should not execute real transactions on an exchange without the `DCA_ALLOW_REAL` being set to any value.

## Configuration

The configuration drives the orders which are executed regularly. At a given interval of time, the configuration is pulled and the process runs through the list of orders.

For example if you wanted to setup a regular order for 5 `ADAGBP` via kraken then the configuration might look like this:

*config.json*

```json5
{
  "orders": [
    {
      "exchange": "kraken",
      "direction": "buy",
      "ordertype": "market",
      "volume": "5",
      "pair": "ADAGBP",
      "validate": true,
      "enabled": true
    }
  ]
}
```

See [example_config.json](./configuration/example_config.json) will by default upload to the designated location in S3 via terraform.

## Schedules

Scheduling for execution of new orders can be found in the terraform variable `execute_orders_schedules`. Multiple schedules are supported.

For example, if we wanted to define a schedule for 6AM UTC on Friday and Wednesday then we could configure this:

```terraform
variable "execute_orders_schedules" {
  type = list(object({
    description         = string
    schedule_expression = string
  }))

  default = [
    {
      description         = "At 6:00 UTC on every Friday"
      schedule_expression = "cron(0 6 ? * FRI *)"
    },
    {
      description         = "At 6:00 UTC on every Wednesday"
      schedule_expression = "cron(0 6 ? * WED *)"
    }
  ]
}
```

See [AWS Schedule Expressions](https://docs.aws.amazon.com/lambda/latest/dg/services-cloudwatchevents-expressions.html) for all the supported options.

See [variables.tf](./terraform/variables.tf)
