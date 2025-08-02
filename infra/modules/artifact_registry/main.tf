# Artifact Registry Repository
resource "google_artifact_registry_repository" "repository" {
  location      = var.location
  repository_id = "${var.repository_name}-${var.environment}"
  description   = var.description
  format        = var.format
  project       = var.project_id


  cleanup_policies {
    id     = "delete-untagged"
    action = "DELETE"
    condition {
      tag_state = "UNTAGGED"
    }
  }

  cleanup_policies {
    id     = "keep-new-untagged"
    action = "KEEP"
    condition {
      tag_state  = "UNTAGGED"
      newer_than = "604800s"  # 7 days
    }
  }

  cleanup_policies {
    id     = "delete-prerelease"
    action = "DELETE"
    condition {
      tag_state    = "TAGGED"
      tag_prefixes = ["alpha", "v0"]
      older_than   = "2592000s"  # 30 days
    }
  }

  cleanup_policies {
    id     = "keep-tagged-release"
    action = "KEEP"
    condition {
      tag_state             = "TAGGED"
      tag_prefixes          = ["release"]
      package_name_prefixes = ["webapp", "mobile"]
    }
  }

  cleanup_policies {
    id     = "keep-minimum-versions"
    action = "KEEP"
    most_recent_versions {
      package_name_prefixes = ["webapp", "mobile", "sandbox"]
      keep_count            = 6
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