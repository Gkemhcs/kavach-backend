# Service Account for Cloud Run
resource "google_service_account" "cloud_run" {
  account_id   = "${var.service_account_name}-${var.environment}"
  display_name = "Kavach Cloud Run Service Account"
  description  = "Service account for Kavach Cloud Run service"
  project      = var.project_id

  lifecycle {
    prevent_destroy = true
  }
}

# IAM binding for Cloud Run service account to access Secret Manager
resource "google_project_iam_member" "secret_manager_access" {
  project = var.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.cloud_run.email}"
}

# IAM binding for Cloud Run service account to access Cloud SQL
resource "google_project_iam_member" "cloud_sql_client" {
  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:${google_service_account.cloud_run.email}"
}

# IAM binding for Cloud Run service account to access Cloud Run
resource "google_project_iam_member" "cloud_run_invoker" {
  project = var.project_id
  role    = "roles/monitoring.metricWriter"
  member  = "serviceAccount:${google_service_account.cloud_run.email}"
}


# IAM binding for Cloud Run service account to access Artifact Registry
resource "google_project_iam_member" "artifact_registry_reader" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.cloud_run.email}"
}