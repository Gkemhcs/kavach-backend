# Artifact Registry Repository
resource "google_artifact_registry_repository" "repository" {
  location      = var.location
  repository_id = "${var.repository_name}-${var.environment}"
  description   = var.description
  format        = var.format
  project       = var.project_id



  # Cleanup policy to manage old images
  cleanup_policies {
    id     = "keep-minimum-versions"
    action = "KEEP"
    most_recent_versions {
      keep_count = 10
    }
  }

  cleanup_policies {
    id     = "delete-old-versions"
    action = "DELETE"
    condition {
      older_than = "2592000s"
    }
  }

  # Docker configuration
  docker_config {
    immutable_tags = true
  }

  lifecycle {
    prevent_destroy = true
  }
} 