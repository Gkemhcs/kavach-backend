# DNS Zone (assuming it already exists outside Terraform)
data "google_dns_managed_zone" "zone" {
  name = var.dns_zone_name
}

# A Record for the domain
resource "google_dns_record_set" "a_record" {
  name         = var.domain_name
  managed_zone = data.google_dns_managed_zone.zone.name
  type         = "A"
  ttl          = 300

  rrdatas = [var.load_balancer_ip]
}

# AAAA Record for the domain (required for SSL certificate provisioning)
resource "google_dns_record_set" "aaaa_record" {
  name         = var.domain_name
  managed_zone = data.google_dns_managed_zone.zone.name
  type         = "AAAA"
  ttl          = 300

  rrdatas = [var.load_balancer_ipv6]
}

