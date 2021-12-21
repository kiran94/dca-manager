locals {
  glue_script_prefix            = "glue/scripts"
  glue_hudi_prefix              = "glue/hudi"
  glue_load_transactions_script = "load_transactions.py"
}

// S3 BUCKET
resource "aws_s3_bucket_object" "glue_load_transactions_script" {
  count = var.enable_analytics ? 1 : 0

  bucket = aws_s3_bucket.main.bucket
  key    = join("/", [local.glue_script_prefix, local.glue_load_transactions_script])
  source = "../glue/scripts/load_transactions.py"
  etag   = filemd5("../glue/scripts/load_transactions.py")
}

// IAM ROLE
resource "aws_iam_role" "glue" {
  count = var.enable_analytics ? 1 : 0

  name = "dca-glue-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = [
          "sts:AssumeRole"
        ]
        Effect = "Allow"
        Sid    = "AssumGlueRole"
        Principal = {
          Service = "glue.amazonaws.com"
        }
      }
    ]
  })

  inline_policy {
    name = "dca_glue_policy"
    policy = jsonencode({
      Version = "2012-10-17"
      Statement = [
        {
          Sid    = "AllowS3ReadTransactions"
          Effect = "Allow"
          Action = [
            "s3:GetObject",
            "s3:PutObject",
          ]
          Resource = [
            "${aws_s3_bucket.main.arn}/transactions/*",
          ]
        },
        {
          Sid    = "AllowS3AllDataLake"
          Effect = "Allow"
          Action = [
            "s3:*"
          ]
          Resource = [
            "${aws_s3_bucket.main.arn}/glue/*",
          ]
        }
      ]
    })
  }
}

resource "aws_iam_policy_attachment" "attach_glue_service_role" {
  count = var.enable_analytics ? 1 : 0

  name       = "AttachAWSGlueBasicExecutionRole"
  roles      = [aws_iam_role.glue[count.index].name]
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSGlueServiceRole"
}

# NOTE: Required to read from Glue marketplace
resource "aws_iam_policy_attachment" "attach_ec2_container_registry" {
  count = var.enable_analytics ? 1 : 0

  name       = "AttachAWSEC2ContainerExecutionRole"
  roles      = [aws_iam_role.glue[count.index].name]
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryFullAccess"
}

resource "aws_glue_job" "load_transactions" {
  count = var.enable_analytics ? 1 : 0

  name        = "dca-load-transactions"
  connections = var.glue_connections
  role_arn    = aws_iam_role.glue[count.index].arn

  max_retries       = 0
  timeout           = 10 # mins
  worker_type       = "Standard"
  number_of_workers = 2
  glue_version      = "3.0"

  execution_property {
    max_concurrent_runs = 1
  }

  command {
    script_location = "s3://${aws_s3_bucket.main.bucket}/${aws_s3_bucket_object.glue_load_transactions_script[count.index].id}"
    python_version  = "3"
  }

  default_arguments = {
    "--job-language"                     = "python"
    "--job-bookmark-option"              = "job-bookmark-disable"
    "--input_path"                       = "s3a://${join("/", [aws_s3_bucket.main.bucket, local.lambda_s3_processed_transaction_prefix])}/"
    "--output_path"                      = "s3a://${join("/", [aws_s3_bucket.main.bucket, local.glue_hudi_prefix])}/"
    "--glue_database"                    = aws_glue_catalog_database.main[count.index].name
    "--glue_table"                       = "transactions"
    "--write_operation"                  = "bulk_insert"
    "--enable-metrics"                   = ""
    "--enable-glue-datacatalog"          = ""
    "--enable-continuous-cloudwatch-log" = "true"
  }
}

# DATABASE
resource "aws_glue_catalog_database" "main" {
  count = var.enable_analytics ? 1 : 0

  name        = "dca_manager"
  description = "Dollar Cost Average Analytics"
}

# Github Action
resource "github_actions_secret" "aws_glue_bucket" {
  repository      = github_repository.main.name
  secret_name     = "AWS_GLUE_BUCKET"
  plaintext_value = aws_s3_bucket.main.bucket
}

resource "github_actions_secret" "aws_glue_scripts_prefix" {
  repository      = github_repository.main.name
  secret_name     = "AWS_GLUE_SCRIPT_PREFIX"
  plaintext_value = local.glue_script_prefix
}

