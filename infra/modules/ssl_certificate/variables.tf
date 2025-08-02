variable "project_id" {
  description = "The GCP project ID"
  type        = string
}

variable "environment" {
  description = "The environment (dev, staging, prod)"
  type        = string
}

variable "domain_name" {
  description = "The domain name for the SSL certificate"
  type        = string
}

variable "certificate_name" {
  description = "The name of the SSL certificate"
  type        = string
} 