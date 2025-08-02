output "security_policy_id" {
  description = "The ID of the Cloud Armor security policy"
  value       = google_compute_security_policy.policy.id
}

output "security_policy_name" {
  description = "The name of the Cloud Armor security policy"
  value       = google_compute_security_policy.policy.name
}

output "security_policy_self_link" {
  description = "The self-link of the Cloud Armor security policy"
  value       = google_compute_security_policy.policy.self_link
}

output "backend_service_with_armor_id" {
  description = "The ID of the backend service with Cloud Armor protection"
  value       = google_compute_backend_service.backend_with_armor.id
}

output "backend_service_with_armor_name" {
  description = "The name of the backend service with Cloud Armor protection"
  value       = google_compute_backend_service.backend_with_armor.name
}

output "backend_service_with_armor_self_link" {
  description = "The self-link of the backend service with Cloud Armor protection"
  value       = google_compute_backend_service.backend_with_armor.self_link
} 