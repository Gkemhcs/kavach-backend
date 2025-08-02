output "certificate_id" {
  description = "The ID of the SSL certificate"
  value       = google_compute_managed_ssl_certificate.certificate.id
}

output "certificate_name" {
  description = "The name of the SSL certificate"
  value       = google_compute_managed_ssl_certificate.certificate.name
}

output "certificate_status" {
  description = "The status of the SSL certificate"
  value       = google_compute_managed_ssl_certificate.certificate.managed[0].status
} 