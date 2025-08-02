variable "project_id" {
  description = "The GCP project ID"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "instance_name" {
  description = "Name of the PostgreSQL instance"
  type        = string
}

variable "database_name" {
  description = "Name of the database"
  type        = string
}

variable "database_version" {
  description = "PostgreSQL version"
  type        = string
}

variable "machine_type" {
  description = "Machine type for the database instance"
  type        = string
}

variable "disk_size_gb" {
  description = "Disk size in GB for the database"
  type        = number
}

variable "region" {
  description = "The GCP region"
  type        = string
}

variable "vpc_network_id" {
  description = "The VPC network ID"
  type        = string
}

variable "subnet_id" {
  description = "The subnet ID"
  type        = string
}

variable "authorized_networks" {
  description = "List of authorized networks for database access"
  type = list(object({
    name  = string
    value = string
  }))
  default = []
}

variable "backup_enabled" {
  description = "Enable automated backups"
  type        = bool
}

variable "backup_start_time" {
  description = "Start time for automated backups (HH:MM format)"
  type        = string
}

variable "maintenance_window" {
  description = "Maintenance window configuration"
  type = object({
    day          = number
    hour         = number
    update_track = string
  })
}

variable "deletion_protection" {
  description = "Enable deletion protection"
  type        = bool
} 