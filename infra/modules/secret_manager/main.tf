# Secret Manager secrets for sensitive environment variables
resource "google_secret_manager_secret" "secrets" {
  for_each = var.app_secrets

  secret_id = each.key
  project   = var.project_id

  replication {
    auto {}
  }

  labels = {
    environment = var.environment
    application = "kavach"
  }

  lifecycle {
    prevent_destroy = true
  }
}

# Secret versions for sensitive environment variables
resource "google_secret_manager_secret_version" "secret_versions" {
  for_each = var.app_secrets

  secret      = google_secret_manager_secret.secrets[each.key].id
  secret_data = each.value

  lifecycle {
    ignore_changes = [
      secret_data
    ]
  }
} 