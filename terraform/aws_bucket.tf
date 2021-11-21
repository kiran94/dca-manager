resource "aws_s3_bucket" "main" {
  bucket = "dca-manager"
}

resource "aws_s3_bucket_object" "config" {
  bucket = aws_s3_bucket.main.bucket
  key    = "config.json"
  source = "../configuration/example_config.json"
  etag   = filemd5("../configuration/example_config.json")

  lifecycle = {
    ignore_changes = all
  }
}
