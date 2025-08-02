# Network Outputs
output "vpc_id" {
  description = "The ID of the VPC"
  value       = module.network.vpc_id
}

output "vpc_name" {
  description = "The name of the VPC"
  value       = module.network.vpc_name
}

output "subnet_id" {
  description = "The ID of the subnet"
  value       = module.network.subnet_id
}

output "subnet_name" {
  description = "The name of the subnet"
  value       = module.network.subnet_name
}

output "cloud_run_subnet_id" {
  description = "The ID of the Cloud Run subnet"
  value       = module.network.cloud_run_subnet_id
}

output "cloud_run_subnet_name" {
  description = "The name of the Cloud Run subnet"
  value       = module.network.cloud_run_subnet_name
}

output "direct_vpc_network" {
  description = "The VPC network for direct VPC access"
  value       = module.network.direct_vpc_network
}

output "direct_vpc_subnetwork" {
  description = "The VPC subnetwork for direct VPC access"
  value       = module.network.direct_vpc_subnetwork
}

output "direct_vpc_network_tags" {
  description = "Network tags for direct VPC access"
  value       = module.network.direct_vpc_network_tags
}

# Database Outputs
output "database_instance_name" {
  description = "The name of the database instance"
  value       = module.postgresql.instance_name
}

output "database_connection_name" {
  description = "The connection name of the database instance"
  value       = module.postgresql.instance_connection_name
  sensitive   = true
}

output "database_private_ip_address" {
  description = "The private IP address of the database instance"
  value       = module.postgresql.private_ip_address
  sensitive   = true
}

output "database_public_ip_address" {
  description = "The public IP address of the database instance"
  value       = module.postgresql.public_ip_address
  sensitive   = true
}

# Cloud Run Outputs
output "cloud_run_service_name" {
  description = "The name of the Cloud Run service"
  value       = module.cloud_run.service_name
}

output "cloud_run_service_url" {
  description = "The URL of the Cloud Run service"
  value       = module.cloud_run.service_url
}

output "cloud_run_service_location" {
  description = "The location of the Cloud Run service"
  value       = module.cloud_run.service_location
}

output "cloud_run_service_id" {
  description = "The ID of the Cloud Run service"
  value       = module.cloud_run.service_id
}

output "cloud_run_service_account" {
  description = "The service account used by Cloud Run"
  value       = module.cloud_run.service_account
}

# Load Balancer Outputs
output "load_balancer_ip" {
  description = "The IP address of the load balancer"
  value       = module.load_balancer.load_balancer_ip
}

output "load_balancer_name" {
  description = "The name of the load balancer"
  value       = module.load_balancer.load_balancer_name
}

output "backend_service_name" {
  description = "The name of the backend service"
  value       = module.load_balancer.backend_service_name
}

output "backend_service_id" {
  description = "The ID of the backend service"
  value       = module.load_balancer.backend_service_id
}

output "ssl_certificate_id" {
  description = "The ID of the SSL certificate (if enabled)"
  value       = module.load_balancer.ssl_certificate_id
}

# Cloud Armor Outputs
output "cloud_armor_policy_name" {
  description = "The name of the Cloud Armor security policy"
  value       = module.cloud_armor.security_policy_name
}

output "cloud_armor_policy_id" {
  description = "The ID of the Cloud Armor security policy"
  value       = module.cloud_armor.security_policy_id
}

output "backend_service_with_armor_name" {
  description = "The name of the backend service with Cloud Armor protection"
  value       = module.cloud_armor.backend_service_with_armor_name
}

# IAM Outputs
output "service_account_email" {
  description = "The email of the service account"
  value       = module.iam.cloud_run_service_account_email
}

output "service_account_name" {
  description = "The name of the service account"
  value       = module.iam.cloud_run_service_account_name
}

# Environment-specific outputs
output "environment" {
  description = "The current environment"
  value       = var.environment
}

output "project_id" {
  description = "The GCP project ID"
  value       = var.project_id
}

# Artifact Registry Outputs
output "artifact_repository_name" {
  description = "The name of the Artifact Registry repository"
  value       = module.artifact_registry.repository_name
}

output "artifact_repository_location" {
  description = "The location of the Artifact Registry repository"
  value       = module.artifact_registry.repository_location
}

output "artifact_repository_id" {
  description = "The ID of the Artifact Registry repository"
  value       = module.artifact_registry.repository_id
}

# Secret Manager Outputs
output "secret_names" {
  description = "The names of the created secrets"
  value       = module.secret_manager.secret_names
}

output "secret_ids" {
  description = "The IDs of the created secrets"
  value       = module.secret_manager.secret_ids
}

output "region" {
  description = "The GCP region"
  value       = var.region
}

# DNS Outputs
output "dns_a_record_name" {
  description = "The name of the A record"
  value       = module.dns.a_record_name
}

output "dns_a_record_ip" {
  description = "The IP address in the A record"
  value       = module.dns.a_record_ip
}

