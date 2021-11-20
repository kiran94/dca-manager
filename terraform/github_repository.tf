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

output "github_repository_ssh_clone_url" {
  value = github_repository.main.ssh_clone_url
}
