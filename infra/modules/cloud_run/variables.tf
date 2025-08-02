variable "project_id" {
  description = "The GCP project ID"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "service_name" {
  description = "Name of the Cloud Run service"
  type        = string
}

variable "region" {
  description = "The GCP region"
  type        = string
}

variable "image_url" {
  description = "Container image URL"
  type        = string
}

variable "vpc_network" {
  description = "VPC network for direct VPC access"
  type        = string
}

variable "vpc_subnetwork" {
  description = "VPC subnetwork for direct VPC access"
  type        = string
}

variable "network_tags" {
  description = "Network tags for direct VPC access"
  type        = list(string)
  default     = ["cloud-run-direct-vpc"]
}

variable "cpu" {
  description = "CPU allocation for the service"
  type        = string
  default     = "1000m"
}

variable "memory" {
  description = "Memory allocation for the service"
  type        = string
  default     = "512Mi"
}

variable "min_instances" {
  description = "Minimum number of instances"
  type        = number
  default     = 0
}

variable "max_instances" {
  description = "Maximum number of instances"
  type        = number
  default     = 10
}

variable "timeout_seconds" {
  description = "Request timeout in seconds"
  type        = number
  default     = 300
}

variable "service_account" {
  description = "Service account email"
  type        = string
}

variable "env_vars" {
  description = "Environment variables"
  type        = map(string)
  default     = {}
}

variable "secrets" {
  description = "Secret environment variables"
  type = map(object({
    secret  = string
    version = string
  }))
  default = {}
}

variable "allow_unauthenticated" {
  description = "Allow unauthenticated access"
  type        = bool
  default     = false
} 