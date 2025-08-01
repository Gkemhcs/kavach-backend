# =============================================================================
# GLOBAL CONFIGURATION
# =============================================================================

# REQUIRED: Set your GCP project ID
project_id = "taskpilot-backend-project"

# Environment configuration
environment = "dev"  # Options: dev, staging, prod
region      = "asia-south1"  # GCP region for all resources

# =============================================================================
# NETWORK CONFIGURATION
# =============================================================================

# VPC and Subnet settings
vpc_name     = "kavach-vpc"
subnet_name  = "kavach-backend-subnet"
subnet_cidr  = "10.0.0.0/24"  # Main subnet CIDR

# =============================================================================
# DATABASE CONFIGURATION
# =============================================================================

# PostgreSQL instance settings
db_instance_name = "kavach-postgres-db"
database_name    = "kavach_db"
database_version = "POSTGRES_15"

# Database resources (adjust based on your needs)
db_machine_type = "db-f1-micro"  # Smallest instance for dev
db_disk_size_gb = 10

# Database security and maintenance
db_backup_enabled = true
db_backup_start_time = "02:00"
db_deletion_protection = true

# Maintenance window (Sunday 3 AM)
db_maintenance_window = {
  day          = 7  # Sunday
  hour         = 3  # 3 AM
  update_track = "stable"
}

# Authorized networks for database access (empty for dev)
db_authorized_networks = []

# =============================================================================
# CLOUD RUN CONFIGURATION (commented out for now)
# =============================================================================

# Service settings
# cloud_run_service_name = "kavach-backend"

# REQUIRED: Set your container image URL
cloud_run_image_url = "asia-south1-docker.pkg.dev/taskpilot-backend-project/kavach-artifact-repo/kavach-backend:latest"

# Resource allocation (optimized for cost)
# cloud_run_cpu    = "1000m"    # 1 CPU
# cloud_run_memory = "512Mi"    # 512 MB RAM

# Scaling configuration
# cloud_run_max_instances = 5   # Reduced from 10 for cost optimization
# cloud_run_min_instances = 0   # Scale to zero when not in use
# cloud_run_concurrency   = 80  # Concurrent requests per instance
# cloud_run_timeout_seconds = 300

# Environment variables for Cloud Run
# cloud_run_env_vars = {
#   NODE_ENV    = "production"
#   LOG_LEVEL   = "info"
#   API_VERSION = "v1"
# }

# Secret environment variables (will be populated from Secret Manager)
# cloud_run_secrets = []

# =============================================================================
# LOAD BALANCER CONFIGURATION (commented out for now)
# =============================================================================

# Load balancer settings
# load_balancer_name = "kavach-lb"
# ssl_certificate_name = "kavach-ssl-cert"

# Domain configuration (set your domain for production)
# domain_name = ""  # e.g., "api.yourdomain.com"
# enable_ssl  = false  # Set to true when you have a domain

# =============================================================================
# CLOUD ARMOR SECURITY (commented out for now)
# =============================================================================

# Security policy
# cloud_armor_policy_name = "kavach-armor-policy"

# Basic security rules (customize based on your needs)
# cloud_armor_rules = [
#   {
#     action      = "deny(403)"
#     priority    = 1000
#     description = "Deny access from suspicious IP ranges"
#     match = {
#       versioned_expr = "SRC_IPS_V1"
#       config = {
#         src_ip_ranges = ["0.0.0.0/0"]
#       }
#     }
#   }
# ]

# =============================================================================
# ARTIFACT REGISTRY CONFIGURATION
# =============================================================================

# Container registry settings
artifact_repository_name = "kavach-artifact-repo"
artifact_format         = "DOCKER"
artifact_description    = "Kavach application container images"

# =============================================================================
# APPLICATION ENVIRONMENT VARIABLES
# =============================================================================

# Plain environment variables (non-sensitive)
# Note: Database variables (DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD) 
# are automatically handled by the PostgreSQL module and Secret Manager
app_env_vars = {
  APP_ENV     = "development"
  PORT        = "8080"
  LOG_LEVEL   = "debug"
  API_VERSION = "v1"
}

# =============================================================================
# SECRET MANAGER CONFIGURATION
# =============================================================================

# Sensitive environment variables from .env file
app_secrets = {
  JWT_ACCESS_TOKEN_SECRET  = "ii2ir2hi2irhirhihi"
  JWT_REFRESH_TOKEN_SECRET = "3jr3orj3ororooojo"
  GITHUB_CLIENT_SECRET     = "1923b5b5819ea865611d022fe939b2161547b79b"
  GITHUB_CLIENT_ID         = "Ov23liTQd2sTs4yFbcaH"
  GITHUB_CALLBACK_URL      = "http://kavach.gkem.cloud/auth/github/callback"
}

# =============================================================================
# IAM CONFIGURATION
# =============================================================================

# Service account settings
service_account_name = "kavach-sa" 

# =============================================================================
# ENVIRONMENT-SPECIFIC OVERRIDES
# =============================================================================

# Uncomment and modify these sections for different environments

# For staging environment:
# environment = "staging"
# db_machine_type = "db-g1-small"
# db_disk_size_gb = 20
# cloud_run_max_instances = 10
# cloud_run_min_instances = 1

# For production environment:
# environment = "prod"
# db_machine_type = "db-custom-1-3840"  # 1 vCPU, 3.75 GB RAM
# db_disk_size_gb = 50
# cloud_run_max_instances = 20
# cloud_run_min_instances = 2
# enable_ssl = true
# domain_name = "api.yourdomain.com" 