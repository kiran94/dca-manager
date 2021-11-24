resource "github_repository" "main" {

  name        = "dca-manager"
  description = "Automated Dollar Cost Average Process"
  visibility  = "private"

  has_projects  = false
  has_wiki      = false
  has_downloads = false

  allow_merge_commit     = false
  allow_rebase_merge     = false
  allow_auto_merge       = false
  delete_branch_on_merge = true
}

# GitHub Action Secrets
resource "github_actions_secret" "aws_assume_role_arn" {
  repository      = github_repository.main.name
  secret_name     = "AWS_ROLE_TO_ASSUME"
  plaintext_value = aws_iam_role.github_action_role.arn
}

resource "github_actions_secret" "aws_access_key_id" {
  repository      = github_repository.main.name
  secret_name     = "AWS_ACCESS_KEY_ID"
  plaintext_value = aws_iam_access_key.github_action_user_access.id
}

resource "github_actions_secret" "aws_secret_access_key" {
  repository      = github_repository.main.name
  secret_name     = "AWS_SECRET_ACCESS_KEY"
  plaintext_value = aws_iam_access_key.github_action_user_access.secret
}

resource "github_actions_secret" "aws_default_region" {
  repository      = github_repository.main.name
  secret_name     = "AWS_DEFAULT_REGION"
  plaintext_value = var.aws_region
}

resource "github_actions_secret" "aws_lambda_bucket" {
  repository      = github_repository.main.name
  secret_name     = "AWS_LAMBDA_BUCKET"
  plaintext_value = aws_s3_bucket.main.bucket
}

resource "github_actions_secret" "aws_lambda_scripts_prefix" {
  repository      = github_repository.main.name
  secret_name     = "AWS_LAMBDA_SCRIPT_PREFIX"
  plaintext_value = local.lambda_s3_scripts_prefix
}

#Outputs

output "github_repository_ssh_clone_url" {
  value = github_repository.main.ssh_clone_url
}

output "github_action_iam_role" {
  value = aws_iam_role.github_action_role.arn
}
