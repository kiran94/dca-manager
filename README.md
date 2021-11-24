# dca-manager

[![main](https://github.com/kiran94/dca-manager/actions/workflows/main.yml/badge.svg)](https://github.com/kiran94/dca-manager/actions/workflows/main.yml)

Dollar Cost Average Manager

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

### Code

Assuming you are at the root of the repository.

```sh
export DCA_BUCKET=$(terraform -chdir=terraform output -raw bucket)
export DCA_CONFIG=$(terraform -chdir=terraform output -raw config_path)

make

# for debugging
make debug
```

### Running

Once the infrastructure is up you can either run the code locally or via lambda. Locally you can run:

```sh
go run main.go
```

This will pull data from a combination of Environment and SSM variables and additionally the config uploaded in S3 to determine what to do.

By default, dcs-manager should not execute real transactions on an exchange without the `DCA_ALLOW_REAL` being set to any value.
