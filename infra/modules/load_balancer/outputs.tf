output "load_balancer_ip" {
  description = "The IP address of the load balancer"
  value       = google_compute_global_address.lb_ip.address
}

output "load_balancer_name" {
  description = "The name of the load balancer"
  value       = var.lb_name
}

output "backend_service_name" {
  description = "The name of the backend service"
  value       = google_compute_backend_service.backend.name
}

output "backend_service_id" {
  description = "The ID of the backend service"
  value       = google_compute_backend_service.backend.id
}

output "ssl_certificate_id" {
  description = "The ID of the SSL certificate (if enabled)"
  value       = var.enable_ssl ? google_compute_managed_ssl_certificate.ssl_cert[0].id : null
}

output "url_map_id" {
  description = "The ID of the URL map"
  value       = google_compute_url_map.url_map.id
}

output "health_check_id" {
  description = "The ID of the health check"
  value       = google_compute_health_check.health_check.id
}

output "network_endpoint_group_id" {
  description = "The ID of the network endpoint group"
  value       = google_compute_region_network_endpoint_group.neg.id
}

output "https_proxy_id" {
  description = "The ID of the HTTPS proxy (if SSL enabled)"
  value       = var.enable_ssl ? google_compute_target_https_proxy.https_proxy[0].id : null
}

output "http_proxy_id" {
  description = "The ID of the HTTP proxy"
  value       = var.enable_ssl ? google_compute_target_http_proxy.http_proxy[0].id : google_compute_target_http_proxy.http_proxy_no_ssl[0].id
} 