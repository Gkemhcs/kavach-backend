# Random password for database
resource "random_password" "db_password" {
  length  = 16
  special = true
  upper   = true
  lower   = true
  numeric = true
}

# Secret Manager secret for database password
resource "google_secret_manager_secret" "db_password" {
  secret_id = "kavach-db-password-${var.environment}"
  project   = var.project_id

  replication {
    auto {}
  }

  lifecycle {
    prevent_destroy = true
  }
}

resource "google_secret_manager_secret_version" "db_password" {
  secret      = google_secret_manager_secret.db_password.id
  secret_data = random_password.db_password.result
}

# Global address for VPC peering with private services
resource "google_compute_global_address" "private_ip_address" {
  name          = "private-ip-address-${var.environment}"
  purpose       = "VPC_PEERING"
  address_type  = "INTERNAL"
  prefix_length = 16
  network       = var.vpc_network_id
}

# Service networking connection for private services access
resource "google_service_networking_connection" "private_vpc_connection" {
  network                 = var.vpc_network_id
  service                 = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.private_ip_address.name]
}

# PostgreSQL Cloud SQL Instance
resource "google_sql_database_instance" "instance" {
  name             = "${var.instance_name}-${var.environment}"
  database_version = var.database_version
  region           = var.region
  project          = var.project_id

  depends_on = [google_service_networking_connection.private_vpc_connection, google_secret_manager_secret_version.db_password]

  settings {
    tier                        = var.machine_type
    disk_size                   = var.disk_size_gb
    disk_type                   = "PD_SSD"
    disk_autoresize             = true
    disk_autoresize_limit       = var.disk_size_gb * 2
    availability_type           = "ZONAL"
    backup_configuration {
      enabled                        = var.backup_enabled
      start_time                     = var.backup_start_time
      point_in_time_recovery_enabled = true
      transaction_log_retention_days = 7
      backup_retention_settings {
        retained_backups = 7
      }
    }

    maintenance_window {
      day          = var.maintenance_window.day
      hour         = var.maintenance_window.hour
      update_track = var.maintenance_window.update_track
    }

    ip_configuration {
      ipv4_enabled                                  = false
      private_network                               = var.vpc_network_id
      enable_private_path_for_google_cloud_services = true
      
      dynamic "authorized_networks" {
        for_each = var.authorized_networks
        content {
          name  = authorized_networks.value.name
          value = authorized_networks.value.value
        }
      }
    }

    insights_config {
      query_insights_enabled  = true
      query_string_length     = 1024
      record_application_tags = true
      record_client_address   = true
    }

    # Enable Cloud SQL Proxy for secure connections
    database_flags {
      name  = "max_connections"
      value = "100"
    }
  }

  deletion_protection = var.deletion_protection

  lifecycle {
    prevent_destroy = true
  }
}

# Database
resource "google_sql_database" "database" {
  name     = var.database_name
  instance = google_sql_database_instance.instance.name
  project  = var.project_id

  charset   = "UTF8"
  collation = "en_US.UTF8"
}

# Database user
resource "google_sql_user" "user" {
  name     = "kavach_user"
  instance = google_sql_database_instance.instance.name
  project  = var.project_id
  password = random_password.db_password.result

  lifecycle {
    prevent_destroy = true
  }
} 