variable "project_id" {
  description = "The GCP project ID"
  type        = string
}

variable "environment" {
  description = "The environment (dev, staging, prod)"
  type        = string
}

variable "domain_name" {
  description = "The domain name for the DNS records"
  type        = string
}

variable "load_balancer_ip" {
  description = "The IPv4 address of the load balancer"
  type        = string
}

variable "load_balancer_ipv6" {
  description = "The IPv6 address of the load balancer"
  type        = string
}

variable "dns_zone_name" {
  description = "The name of the existing DNS zone"
  type        = string
} 