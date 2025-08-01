# Configure Terraform and providers
terraform {
  required_version = ">= 1.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 4.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.0"
    }
  }
}

# Configure Google Cloud Provider
provider "google" {
  project = var.project_id
  region  = var.region
}

# Artifact Registry Module
module "artifact_registry" {
  source = "./modules/artifact_registry"

  project_id          = var.project_id
  environment         = var.environment
  repository_name     = var.artifact_repository_name
  location            = var.region
  format              = var.artifact_format
  description         = var.artifact_description
}

# Secret Manager Module
module "secret_manager" {
  source = "./modules/secret_manager"

  project_id   = var.project_id
  environment  = var.environment
  app_env_vars = var.app_env_vars
  app_secrets  = var.app_secrets
}

# Network Module
module "network" {
  source = "./modules/network"

  project_id     = var.project_id
  environment    = var.environment
  vpc_name       = var.vpc_name
  subnet_name    = var.subnet_name
  subnet_cidr    = var.subnet_cidr
  region         = var.region
}

# PostgreSQL Database Module
module "postgresql" {
  source = "./modules/postgresql"

  project_id           = var.project_id
  environment          = var.environment
  instance_name        = var.db_instance_name
  database_name        = var.database_name
  database_version     = var.database_version
  machine_type         = var.db_machine_type
  disk_size_gb         = var.db_disk_size_gb
  region               = var.region
  vpc_network_id       = module.network.vpc_id
  subnet_id            = module.network.subnet_id
  authorized_networks  = var.db_authorized_networks
  backup_enabled       = var.db_backup_enabled
  backup_start_time    = var.db_backup_start_time
  maintenance_window   = var.db_maintenance_window
  deletion_protection  = var.db_deletion_protection
}

# IAM Module for Service Accounts
module "iam" {
  source = "./modules/iam"

  project_id     = var.project_id
  environment    = var.environment
  service_account_name = var.service_account_name
}

# Cloud Run Module (commented out for now - will be enabled after Docker image is ready)
# module "cloud_run" {
#   source = "./modules/cloud_run"
# 
#   project_id      = var.project_id
#   environment     = var.environment
#   service_name    = var.cloud_run_service_name
#   image_url       = var.cloud_run_image_url
#   region          = var.region
#   vpc_network     = module.network.vpc_self_link
#   vpc_subnetwork  = module.network.cloud_run_subnet_self_link
#   
#   # Database environment variables from PostgreSQL module
#   run_env_vars = merge({
#     DB_HOST = module.postgresql.private_ip_address
#     DB_PORT = "5432"
#     DB_NAME = module.postgresql.database_name
#     DB_USER = module.postgresql.user_name
#   }, module.secret_manager.app_env_vars)
#   
#   # Database password from Secret Manager + other secrets
#   run_env_secrets = merge({
#     DB_PASSWORD = {
#       secret  = module.postgresql.password_secret_id
#       version = "latest"
#     }
#   }, {
#     for secret_ref in module.secret_manager.secret_references : secret_ref.name => {
#       secret  = secret_ref.secret
#       version = secret_ref.version
#     }
#   })
#   cpu             = var.cloud_run_cpu
#   memory          = var.cloud_run_memory
#   max_instances   = var.cloud_run_max_instances
#   min_instances   = var.cloud_run_min_instances
#   concurrency     = var.cloud_run_concurrency
#   timeout_seconds = var.cloud_run_timeout_seconds
#   service_account = module.iam.cloud_run_service_account_email
# }

# Load Balancer Module (commented out for now - will be enabled after Cloud Run is ready)
# module "load_balancer" {
#   source = "./modules/load_balancer"
# 
#   project_id           = var.project_id
#   environment          = var.environment
#   lb_name              = var.load_balancer_name
#   region               = var.region
#   cloud_run_service    = module.cloud_run.service_name
#   cloud_run_location   = module.cloud_run.service_location
#   ssl_certificate_name = var.ssl_certificate_name
#   domain_name          = var.domain_name
#   enable_ssl           = var.enable_ssl
# }

# Cloud Armor Module (commented out for now - will be enabled after Load Balancer is ready)
# module "cloud_armor" {
#   source = "./modules/cloud_armor"
# 
#   project_id    = var.project_id
#   environment   = var.environment
#   policy_name   = var.cloud_armor_policy_name
#   region        = var.region
#   load_balancer = module.load_balancer.backend_service_name
#   rules         = var.cloud_armor_rules
# } 