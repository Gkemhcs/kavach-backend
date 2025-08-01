# VPC Network
resource "google_compute_network" "vpc" {
  name                    = "${var.vpc_name}-${var.environment}"
  auto_create_subnetworks = false
  routing_mode           = "REGIONAL"
  project                = var.project_id

  lifecycle {
    prevent_destroy = true
  }
}

# Main Subnet
resource "google_compute_subnetwork" "subnet" {
  name          = "${var.subnet_name}-${var.environment}"
  ip_cidr_range = var.subnet_cidr
  region        = var.region
  network       = google_compute_network.vpc.id
  project       = var.project_id

  # Enable flow logs for network monitoring
  log_config {
    aggregation_interval = "INTERVAL_5_SEC"
    flow_sampling       = 0.5
    metadata           = "INCLUDE_ALL_METADATA"
  }

  # Enable private Google access for Cloud SQL
  private_ip_google_access = true

  lifecycle {
    prevent_destroy = true
  }
}

# Cloud Run Subnet for VPC Direct Egress Access
resource "google_compute_subnetwork" "cloud_run_subnet" {
  name          = "cloud-run-subnet-${var.environment}"
  ip_cidr_range = "10.8.0.0/28" # Smaller range (16 IPs) for Cloud Run VPC access
  region        = var.region
  network       = google_compute_network.vpc.id
  project       = var.project_id

  # Enable flow logs for network monitoring
  log_config {
    aggregation_interval = "INTERVAL_5_SEC"
    flow_sampling       = 0.5
    metadata           = "INCLUDE_ALL_METADATA"
  }

  # Enable private Google access
  private_ip_google_access = true

  lifecycle {
    prevent_destroy = true
  }
}

# Firewall rule to allow internal communication
resource "google_compute_firewall" "internal" {
  name    = "allow-internal-${var.environment}"
  network = google_compute_network.vpc.name
  project = var.project_id

  allow {
    protocol = "tcp"
    ports    = ["0-65535"]
  }

  allow {
    protocol = "udp"
    ports    = ["0-65535"]
  }

  allow {
    protocol = "icmp"
  }

  source_ranges = [var.subnet_cidr, "10.8.0.0/28"] # Include both main subnet and Cloud Run subnet
  target_tags   = ["internal"]
}

# Firewall rule to allow Cloud Run to access VPC resources
resource "google_compute_firewall" "cloud_run_vpc" {
  name    = "allow-cloud-run-vpc-${var.environment}"
  network = google_compute_network.vpc.name
  project = var.project_id

  allow {
    protocol = "tcp"
    ports    = ["5432"] # PostgreSQL
  }

  source_ranges = ["10.8.0.0/28"] # Cloud Run subnet range
  target_tags   = ["cloud-sql"]
}

# Firewall rule to allow load balancer health checks to Cloud Run
resource "google_compute_firewall" "lb_health_checks" {
  name    = "allow-lb-health-checks-${var.environment}"
  network = google_compute_network.vpc.name
  project = var.project_id

  allow {
    protocol = "tcp"
    ports    = ["80", "443"] # HTTP and HTTPS
  }

  # Google's health check IP ranges
  source_ranges = [
    "35.191.0.0/16",    # Google Cloud Load Balancer health checks
    "130.211.0.0/22"    # Google Cloud Load Balancer health checks
  ]

  # Allow health checks to reach Cloud Run services
  target_tags = ["cloud-run"]
} 