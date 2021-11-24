resource "aws_iam_user" "github_action_user" {
  name = "github_action_user_dcs"
}

resource "aws_iam_access_key" "github_action_user_access" {
  user = aws_iam_user.github_action_user.name
}

resource "aws_iam_role" "github_action_role" {
  name = "dcs_github_action_role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = [
          "sts:AssumeRole",
          "sts:TagSession"
        ]
        Effect = "Allow"
        Sid    = "AssumeGitHubActionAwsRole"
        Principal = {
          AWS = [
            data.aws_caller_identity.current.arn,
            aws_iam_user.github_action_user.arn
          ]
        }
      }
    ]
  })

  inline_policy {
    name = "github_action_role_policy"
    policy = jsonencode({
      Version = "2012-10-17"
      Statement = [
        {
          Action = [
            "s3:GetObject",
            "s3:PutObject",
            "s3:ListBucket",
            "lambda:UpdateFunctionCode",
            "lambda:PublishLayerVersion",
            "lambda:ListLayerVersions",
            "lambda:UpdateFunctionConfiguration",
            "lambda:GetLayerVersion"
          ]
          Effect = "Allow"
          Resource = [
            "${aws_s3_bucket.main.arn}",
            "${aws_s3_bucket.main.arn}/*"
            # "arn:aws:lambda:***:***:function:${aws_lambda_function.TODO.function_name}"
          ]
        }
      ]
    })
  }
}
