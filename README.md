# dca-manager

[![main](https://github.com/kiran94/dca-manager/actions/workflows/main.yml/badge.svg)](https://github.com/kiran94/dca-manager/actions/workflows/main.yml)

Dollar Cost Average Manager

<!-- toc GFM -->

* [Getting Started](#getting-started)
    * [Infrastructure](#infrastructure)
    * [Code](#code)
    * [Running](#running)
* [Configuration](#configuration)

<!-- /toc -->

## Getting Started

### Infrastructure

Assuming you are at the root of the repository.

```sh
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

make

# for debugging
make debug
```

### Running

Once the infrastructure is up you can either run the code locally or via lambda. Locally you can run:

```sh
go run main.go
```

This will pull data from a combination of sources such as the environment and SSM. Additionally the config uploaded in S3 will be used to determine what to do.

By default, dcs-manager should not execute real transactions on an exchange without the `DCA_ALLOW_REAL` being set to any value.
