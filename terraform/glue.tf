locals {
  glue_script_prefix            = "glue/scripts"
  glue_jars_prefix              = "glue/jars"
  glue_hudi_prefix              = "glue/hudi"
  glue_load_transactions_script = "load_transactions.py"
  glue_hudi_spark_jar           = "hudi-spark-bundle_2.11-0.8.0.jar"
  glue_hudi_avro_jar            = "spark-avro_2.11-2.4.7.jar"
  glue_local_jars_path          = "../glue/jars"
}

// S3 BUCKET
resource "aws_s3_bucket_object" "glue_load_transactions_script" {
  bucket = aws_s3_bucket.main.bucket
  key    = join("/", [local.glue_script_prefix, local.glue_load_transactions_script])
  source = "../glue/scripts/load_transactions.py"
  etag   = filemd5("../glue/scripts/load_transactions.py")
}

// IAM ROLE
resource "aws_iam_role" "glue" {
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
          Effect = "Allow"
          Action = [
            "s3:GetObject",
            "s3:PutObject",
          ]
          Resource = [
            "${aws_s3_bucket.main.arn}",
            "${aws_s3_bucket.main.arn}/glue/*",
          ]
        }
      ]
    })
  }
}

resource "aws_iam_policy_attachment" "attach_glue_service_role" {
  name       = "AttachAWSGlueBasicExecutionRole"
  roles      = [aws_iam_role.glue.name]
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSGlueServiceRole"
}


resource "aws_iam_policy_attachment" "attach_ec2_container_registry" {
  name       = "AttachAWSEC2ContainerExecutionRole"
  roles      = [aws_iam_role.glue.name]
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryFullAccess"
}

resource "aws_iam_policy_attachment" "attach_ec2_full_access" {
  name       = "AttachAWSEC2FullAccess"
  roles      = [aws_iam_role.glue.name]
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2FullAccess"
}

# JOB
# WARN: Document into README
variable "glue_connections" {
  type    = list(string)
  default = ["hudi-connection"]
}

resource "aws_glue_job" "load_transactions" {
  name        = "dca-load-transactions"
  connections = var.glue_connections
  role_arn    = aws_iam_role.glue.arn

  max_retries       = 0
  timeout           = 600
  worker_type       = "G.2X" # TODO: Add
  number_of_workers = 4
  glue_version      = "2.0"

  execution_property {
    max_concurrent_runs = 1
  }

  command {
    script_location = "s3://${aws_s3_bucket.main.bucket}/${aws_s3_bucket_object.glue_load_transactions_script.id}"
    python_version  = "3"
  }

  default_arguments = {
    "--job-language"                     = "python"
    "--job-bookmark-option"              = "job-bookmark-disable"
    "--input_path"                       = "s3://${join("/", [aws_s3_bucket.main.bucket, local.lambda_s3_processed_transaction_prefix])}/"
    "--output_path"                      = "s3://${join("/", [aws_s3_bucket.main.bucket, local.glue_hudi_prefix])}/"
    "--glue_database"                    = aws_glue_catalog_database.main.name
    "--glue_table"                       = "test"
    "--write_operation"                  = "upsert"
    "--enable-metrics"                   = ""
    "--enable-glue-datacatalog"          = ""
    "--enable-continuous-cloudwatch-log" = "true"
    "--extra-jars" = join(",", [
      "s3://${aws_s3_bucket.main.id}/${aws_s3_bucket_object.hudi_spark.id}",
      "s3://${aws_s3_bucket.main.id}/${aws_s3_bucket_object.spark_avro.id}"
    ])
  }
}

# EXTRA JARS
resource "aws_s3_bucket_object" "hudi_spark" {
  bucket = aws_s3_bucket.main.bucket
  key    = join("/", [local.glue_jars_prefix, local.glue_hudi_spark_jar])
  source = join("/", [local.glue_local_jars_path, local.glue_hudi_spark_jar])
  etag   = filemd5(join("/", [local.glue_local_jars_path, local.glue_hudi_spark_jar]))
}

resource "aws_s3_bucket_object" "spark_avro" {
  bucket = aws_s3_bucket.main.bucket
  key    = join("/", [local.glue_jars_prefix, local.glue_hudi_avro_jar])
  source = join("/", [local.glue_local_jars_path, local.glue_hudi_avro_jar])
  etag   = filemd5(join("/", [local.glue_local_jars_path, local.glue_hudi_avro_jar]))
}

# DATABASE
resource "aws_glue_catalog_database" "main" {
  name        = "dca_manager"
  description = "Dollar Cost Average Analytics"
}
