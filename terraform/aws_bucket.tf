resource "aws_s3_bucket" "main" {
  bucket = "dca-manager"

  lifecycle_rule {
    enabled = true
    id      = "autoclean_pending_transactions"
    prefix  = local.lambda_s3_pending_transaction_prefix

    transition {
      days          = 30
      storage_class = "STANDARD_IA"
    }

    transition {
      days          = 60
      storage_class = "GLACIER"
    }
  }


  lifecycle_rule {
    enabled = true
    id      = "autoclean_processed_transactions"
    prefix  = local.lambda_s3_processed_transaction_prefix

    transition {
      days          = 30
      storage_class = "STANDARD_IA"
    }

    transition {
      days          = 60
      storage_class = "GLACIER"
    }
  }
}

resource "aws_s3_bucket_object" "config" {
  bucket = aws_s3_bucket.main.bucket
  key    = "config.json"
  source = "../pkg/configuration/example_config.json"
  etag   = filemd5("../pkg/configuration/example_config.json")

  lifecycle {
    ignore_changes = all
  }
}

output "bucket" {
  value = aws_s3_bucket.main.bucket
}

output "config_path" {
  value = aws_s3_bucket_object.config.id
}
