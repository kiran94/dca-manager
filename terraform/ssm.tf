resource "aws_ssm_parameter" "kraken_api_key" {
  name  = "/dca-manager/kraken/key"
  type  = "SecureString"
  value = var.KRAKEN_API_KEY

  lifecycle {
    ignore_changes = all
  }
}

resource "aws_ssm_parameter" "kraken_api_secret" {
  name  = "/dca-manager/kraken/secret"
  type  = "SecureString"
  value = var.KRAKEN_API_SECRET

  lifecycle {
    ignore_changes = all
  }
}

# Override with TF_VAR_
variable "KRAKEN_API_KEY" {
  default = "dummy"
}
variable "KRAKEN_API_SECRET" {
  default = "dummy"
}
