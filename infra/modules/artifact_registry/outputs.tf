output "repository_name" {
  description = "The name of the Artifact Registry repository"
  value       = google_artifact_registry_repository.repository.name
}

output "repository_id" {
  description = "The ID of the Artifact Registry repository"
  value       = google_artifact_registry_repository.repository.repository_id
}

output "repository_location" {
  description = "The location of the Artifact Registry repository"
  value       = google_artifact_registry_repository.repository.location
}

output "repository_format" {
  description = "The format of the Artifact Registry repository"
  value       = google_artifact_registry_repository.repository.format
}

output "repository_description" {
  description = "The description of the Artifact Registry repository"
  value       = google_artifact_registry_repository.repository.description
} 