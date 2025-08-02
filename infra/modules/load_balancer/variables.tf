variable "project_id" {
  description = "The GCP project ID"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "lb_name" {
  description = "Name of the load balancer"
  type        = string
}

variable "region" {
  description = "The GCP region"
  type        = string
}

variable "cloud_run_service" {
  description = "Name of the Cloud Run service"
  type        = string
}

variable "cloud_run_location" {
  description = "Location of the Cloud Run service"
  type        = string
}

variable "ssl_certificate_name" {
  description = "Name of the SSL certificate"
  type        = string
  default     = "kavach-ssl-cert"
}

variable "domain_name" {
  description = "Domain name for SSL certificate"
  type        = string
  default     = ""
}

variable "enable_ssl" {
  description = "Enable SSL/TLS termination"
  type        = bool
  default     = false
} 