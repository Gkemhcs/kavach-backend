# Global Variables
variable "project_id" {
  description = "The GCP project ID"
  type        = string
}

variable "environment" {
  description = "Environment name (dev, staging, prod)"
  type        = string
  default     = "dev"
  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Environment must be one of: dev, staging, prod."
  }
}

variable "region" {
  description = "The GCP region for resources"
  type        = string
  default     = "asia-south1"
}

# Network Variables
variable "vpc_name" {
  description = "Name of the VPC"
  type        = string
  default     = "kavach-vpc"
}

variable "subnet_name" {
  description = "Name of the subnet"
  type        = string
  default     = "kavach-subnet"
}

variable "subnet_cidr" {
  description = "CIDR block for the subnet"
  type        = string
  default     = "10.0.0.0/24"
}

# Database Variables
variable "db_instance_name" {
  description = "Name of the PostgreSQL instance"
  type        = string
  default     = "kavach-postgres"
}

variable "database_name" {
  description = "Name of the database"
  type        = string
  default     = "kavach"
}

variable "database_version" {
  description = "PostgreSQL version"
  type        = string
  default     = "POSTGRES_15"
}

variable "db_machine_type" {
  description = "Machine type for the database instance"
  type        = string
  default     = "db-f1-micro"
}

variable "db_disk_size_gb" {
  description = "Disk size in GB for the database"
  type        = number
  default     = 10
}

variable "db_authorized_networks" {
  description = "List of authorized networks for database access"
  type = list(object({
    name  = string
    value = string
  }))
  default = []
}

variable "db_backup_enabled" {
  description = "Enable automated backups"
  type        = bool
  default     = true
}

variable "db_backup_start_time" {
  description = "Start time for automated backups (HH:MM format)"
  type        = string
  default     = "02:00"
}

variable "db_maintenance_window" {
  description = "Maintenance window configuration"
  type = object({
    day          = number
    hour         = number
    update_track = string
  })
  default = {
    day          = 7
    hour         = 2
    update_track = "stable"
  }
}

variable "db_deletion_protection" {
  description = "Enable deletion protection for the database"
  type        = bool
  default     = true
}

# Cloud Run Variables
variable "cloud_run_service_name" {
  description = "Name of the Cloud Run service"
  type        = string
  default     = "kavach-api"
}

variable "cloud_run_image_url" {
  description = "Container image URL for Cloud Run"
  type        = string
  default     = ""
}

variable "cloud_run_cpu" {
  description = "CPU allocation for Cloud Run service"
  type        = string
  default     = "1000m"
}

variable "cloud_run_memory" {
  description = "Memory allocation for Cloud Run service"
  type        = string
  default     = "512Mi"
}

variable "cloud_run_max_instances" {
  description = "Maximum number of Cloud Run instances"
  type        = number
  default     = 10
}

variable "cloud_run_min_instances" {
  description = "Minimum number of Cloud Run instances"
  type        = number
  default     = 0
}

variable "cloud_run_concurrency" {
  description = "Number of concurrent requests per instance"
  type        = number
  default     = 80
}

variable "cloud_run_timeout_seconds" {
  description = "Request timeout in seconds"
  type        = number
  default     = 300
}

variable "cloud_run_env_vars" {
  description = "Environment variables for Cloud Run service"
  type        = map(string)
  default     = {}
}

variable "cloud_run_secrets" {
  description = "Secret environment variables for Cloud Run service"
  type = map(object({
    secret  = string
    version = string
  }))
  default = {}
}

# Load Balancer Variables
variable "load_balancer_name" {
  description = "Name of the load balancer"
  type        = string
  default     = "kavach-lb"
}

variable "ssl_certificate_name" {
  description = "Name of the SSL certificate"
  type        = string
  default     = "kavach-ssl-cert"
}

variable "domain_name" {
  description = "Domain name for the load balancer"
  type        = string
  default     = ""
}

variable "dns_zone_name" {
  description = "Name of the existing DNS zone"
  type        = string
  default     = "kavach-zone"
}

variable "enable_ssl" {
  description = "Enable SSL for the load balancer"
  type        = bool
  default     = false
}

# Cloud Armor Variables
variable "cloud_armor_policy_name" {
  description = "Name of the Cloud Armor security policy"
  type        = string
  default     = "kavach-armor-policy"
}

variable "cloud_armor_rules" {
  description = "Cloud Armor security rules"
  type = list(object({
    action      = string
    priority    = number
    description = string
    match = object({
      versioned_expr = string
      config = object({
        src_ip_ranges = list(string)
      })
    })
  }))
  default = [
    {
      action      = "deny(403)"
      priority    = 1000
      description = "Deny access from suspicious IP ranges"
      match = {
        versioned_expr = "SRC_IPS_V1"
        config = {
          src_ip_ranges = ["0.0.0.0/0"]
        }
      }
    }
  ]
}

# Artifact Registry Variables
variable "artifact_repository_name" {
  description = "Name of the Artifact Registry repository"
  type        = string
  default     = "kavach-repo"
}

variable "artifact_format" {
  description = "Format of the Artifact Registry repository"
  type        = string
  default     = "DOCKER"
}

variable "artifact_description" {
  description = "Description of the Artifact Registry repository"
  type        = string
  default     = "Kavach application container images"
}

# Environment Variables
variable "app_env_vars" {
  description = "Map of plain environment variable names and values"
  type        = map(string)
  default = {
    NODE_ENV    = "production"
    LOG_LEVEL   = "info"
    API_VERSION = "v1"
  }
  sensitive = false
}

# Secret Manager Variables
variable "app_secrets" {
  description = "Map of sensitive environment variable names and values"
  type        = map(string)
  default = {
    JWT_SECRET     = ""
    DB_PASSWORD    = ""
    ENCRYPTION_KEY = ""
    API_KEY        = ""
    REDIS_PASSWORD = ""
  }
  sensitive = false
}

# IAM Variables
variable "service_account_name" {
  description = "Name of the service account"
  type        = string
  default     = "kavach-sa"
} 