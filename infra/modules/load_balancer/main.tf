# Google Cloud Load Balancer for Cloud Run
resource "google_compute_global_address" "lb_ip" {
  name         = "${var.lb_name}-ip-${var.environment}"
  project      = var.project_id
  address_type = "EXTERNAL"
  ip_version   = "IPV4"

  lifecycle {
    prevent_destroy = true
  }
}

# SSL Certificate (if enabled)
resource "google_compute_managed_ssl_certificate" "ssl_cert" {
  count   = var.enable_ssl ? 1 : 0
  name    = "${var.ssl_certificate_name}-${var.environment}"
  project = var.project_id

  managed {
    domains = [var.domain_name]
  }

  lifecycle {
    prevent_destroy = true
  }
}

# Backend Service for Cloud Run
resource "google_compute_backend_service" "backend" {
  name        = "${var.lb_name}-backend-${var.environment}"
  project     = var.project_id
  protocol    = "HTTP"
  port_name   = "http"
  timeout_sec = 30

  backend {
    group = google_compute_region_network_endpoint_group.neg.id
  }

  health_checks = [google_compute_health_check.health_check.id]

  log_config {
    enable = true
  }

  lifecycle {
    prevent_destroy = true
  }
}

# Network Endpoint Group for Cloud Run
resource "google_compute_region_network_endpoint_group" "neg" {
  name                  = "${var.lb_name}-neg-${var.environment}"
  project               = var.project_id
  region                = var.region
  network_endpoint_type = "SERVERLESS"
  cloud_run {
    service = var.cloud_run_service
  }
}

# Health Check
resource "google_compute_health_check" "health_check" {
  name    = "${var.lb_name}-health-check-${var.environment}"
  project = var.project_id

  http_health_check {
    port         = 80
    request_path = "/health"
  }

  timeout_sec        = 5
  check_interval_sec = 5
  healthy_threshold  = 2
  unhealthy_threshold = 3
}

# URL Map
resource "google_compute_url_map" "url_map" {
  name            = "${var.lb_name}-url-map-${var.environment}"
  project         = var.project_id
  default_service = google_compute_backend_service.backend.id

  lifecycle {
    prevent_destroy = true
  }
}

# HTTPS Proxy (if SSL enabled)
resource "google_compute_target_https_proxy" "https_proxy" {
  count           = var.enable_ssl ? 1 : 0
  name            = "${var.lb_name}-https-proxy-${var.environment}"
  project         = var.project_id
  url_map         = google_compute_url_map.url_map.id
  ssl_certificates = [google_compute_managed_ssl_certificate.ssl_cert[0].id]

  lifecycle {
    prevent_destroy = true
  }
}

# HTTP Proxy (for redirect to HTTPS)
resource "google_compute_target_http_proxy" "http_proxy" {
  count   = var.enable_ssl ? 1 : 0
  name    = "${var.lb_name}-http-proxy-${var.environment}"
  project = var.project_id
  url_map = google_compute_url_map.url_map.id

  lifecycle {
    prevent_destroy = true
  }
}

# HTTP Proxy (for non-SSL)
resource "google_compute_target_http_proxy" "http_proxy_no_ssl" {
  count   = var.enable_ssl ? 0 : 1
  name    = "${var.lb_name}-http-proxy-${var.environment}"
  project = var.project_id
  url_map = google_compute_url_map.url_map.id

  lifecycle {
    prevent_destroy = true
  }
}

# Global Forwarding Rule for HTTPS
resource "google_compute_global_forwarding_rule" "https_forwarding_rule" {
  count      = var.enable_ssl ? 1 : 0
  name       = "${var.lb_name}-https-forwarding-rule-${var.environment}"
  project    = var.project_id
  target     = google_compute_target_https_proxy.https_proxy[0].id
  port_range = "443"
  ip_address = google_compute_global_address.lb_ip.address

  lifecycle {
    prevent_destroy = true
  }
}

# Global Forwarding Rule for HTTP (redirect to HTTPS)
resource "google_compute_global_forwarding_rule" "http_forwarding_rule" {
  count      = var.enable_ssl ? 1 : 0
  name       = "${var.lb_name}-http-forwarding-rule-${var.environment}"
  project    = var.project_id
  target     = google_compute_target_http_proxy.http_proxy[0].id
  port_range = "80"
  ip_address = google_compute_global_address.lb_ip.address

  lifecycle {
    prevent_destroy = true
  }
}

# Global Forwarding Rule for HTTP (non-SSL)
resource "google_compute_global_forwarding_rule" "http_forwarding_rule_no_ssl" {
  count      = var.enable_ssl ? 0 : 1
  name       = "${var.lb_name}-http-forwarding-rule-${var.environment}"
  project    = var.project_id
  target     = google_compute_target_http_proxy.http_proxy_no_ssl[0].id
  port_range = "80"
  ip_address = google_compute_global_address.lb_ip.address

  lifecycle {
    prevent_destroy = true
  }
}

# URL Map for HTTP to HTTPS redirect
resource "google_compute_url_map" "redirect_url_map" {
  count   = var.enable_ssl ? 1 : 0
  name    = "${var.lb_name}-redirect-url-map-${var.environment}"
  project = var.project_id

  default_url_redirect {
    https_redirect = true
    strip_query    = false
  }

  lifecycle {
    prevent_destroy = true
  }
} 