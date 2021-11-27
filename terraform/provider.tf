terraform {

  backend "s3" {
    bucket = "terraform-kiran"
    key    = "dca-manager.tfstate"
  }

  required_providers {

    github = {
      source  = "integrations/github"
      version = "~> 4.0"
    }

    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.0"
    }
  }
}

variable "aws_region" {
  type    = string
  default = "eu-west-2"
}

provider "github" {}
provider "aws" {
  region = var.aws_region
}

data "aws_caller_identity" "current" {}
