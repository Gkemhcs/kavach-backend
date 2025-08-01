output "secret_names" {
  description = "The names of the created secrets"
  value       = [for secret in google_secret_manager_secret.secrets : secret.secret_id]
}

output "secret_ids" {
  description = "The IDs of the created secrets"
  value       = [for secret in google_secret_manager_secret.secrets : secret.id]
}

output "secret_references" {
  description = "Secret references for Cloud Run environment variables"
  value = [
    for secret in google_secret_manager_secret.secrets : {
      name    = upper(replace(secret.secret_id, "-", "_"))
      secret  = secret.secret_id
      version = "latest"
    }
  ]
}

output "secret_versions" {
  description = "The versions of the created secrets"
  value       = [for version in google_secret_manager_secret_version.secret_versions : version.version]
}

output "app_env_vars" {
  description = "Plain environment variables for Cloud Run"
  value       = var.app_env_vars
}

output "app_secrets" {
  description = "Sensitive environment variables (keys only for security)"
  value       = keys(var.app_secrets)
} 