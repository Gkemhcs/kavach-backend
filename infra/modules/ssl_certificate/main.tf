# Managed SSL Certificate
resource "google_compute_managed_ssl_certificate" "certificate" {
  name = var.certificate_name

  managed {
    domains = [var.domain_name, "www.${var.domain_name}"]
  }

  lifecycle {
    create_before_destroy = true
  }
} 