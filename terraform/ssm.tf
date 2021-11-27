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
