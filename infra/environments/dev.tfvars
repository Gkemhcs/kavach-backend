# Development Environment Configuration
project_id = "your-dev-project-id"
environment = "dev"
region     = "us-central1"

# Network Configuration
vpc_name    = "kavach-vpc"
subnet_name = "kavach-subnet"
subnet_cidr = "10.0.0.0/24"

# Database Configuration
db_instance_name = "kavach-postgres"
database_name    = "kavach"
database_version = "POSTGRES_15"
db_machine_type  = "db-f1-micro"
db_disk_size_gb  = 10

db_backup_enabled = true
db_backup_start_time = "02:00"
db_maintenance_window = {
  day          = 7
  hour         = 3
  update_track = "stable"
}
db_deletion_protection = false

# Cloud Run Configuration
cloud_run_service_name = "kavach-backend"
cloud_run_image_url    = "gcr.io/your-dev-project-id/kavach-backend:latest"
cloud_run_cpu          = "1000m"
cloud_run_memory       = "512Mi"
cloud_run_max_instances = 5
cloud_run_min_instances = 0
cloud_run_concurrency   = 80
cloud_run_timeout_seconds = 300

# Environment variables for Cloud Run
cloud_run_env_vars = {
  ENVIRONMENT = "development"
  LOG_LEVEL   = "debug"
  PORT        = "8080"
}

# Load Balancer Configuration
load_balancer_name = "kavach-lb"
enable_ssl         = false
domain_name        = ""

# Cloud Armor Configuration
cloud_armor_policy_name = "kavach-armor-policy"

# Artifact Registry Configuration
artifact_repository_name = "kavach-repo"
artifact_format = "DOCKER"
artifact_description = "Kavach application container images"

# Secret Manager Configuration
application_secrets = [
  {
    name        = "database-password"
    description = "Database password for Kavach application"
    value       = ""  # Will be auto-generated if empty
  },
  {
    name        = "jwt-secret"
    description = "JWT signing secret for authentication"
    value       = ""  # Will be auto-generated if empty
  },
  {
    name        = "encryption-key"
    description = "Encryption key for sensitive data"
    value       = ""  # Will be auto-generated if empty
  },
  {
    name        = "api-key"
    description = "API key for external services"
    value       = ""  # Will be auto-generated if empty
  },
  {
    name        = "redis-password"
    description = "Redis password for caching"
    value       = ""  # Will be auto-generated if empty
  }
]

# Service Account Configuration
service_account_name = "kavach-sa" 