variable "project_id" {
  description = "The GCP project ID"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "policy_name" {
  description = "Name of the Cloud Armor security policy"
  type        = string
}

variable "region" {
  description = "The GCP region"
  type        = string
}

variable "backend_service_name" {
  description = "Name of the backend service to protect"
  type        = string
}

variable "backend_service_group" {
  description = "Backend service group ID"
  type        = string
}

# Health check variable removed - not supported with Serverless NEGs for Cloud Run

variable "blocked_ips" {
  description = "List of IP addresses to block"
  type        = list(string)
  default     = []
}

variable "custom_rules" {
  description = "List of custom security rules"
  type = list(object({
    action              = string
    priority            = string
    description         = string
    match_expression    = optional(string)
    match_versioned_expr = optional(string)
    match_config        = optional(object({
      src_ip_ranges = list(string)
    }))
  }))
  default = []
} 