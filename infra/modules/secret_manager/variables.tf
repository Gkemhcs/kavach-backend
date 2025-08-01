variable "project_id" {
  description = "The GCP project ID"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "app_env_vars" {
  description = "Map of plain environment variable names and values"
  type        = map(string)
  default     = {}
  sensitive   = false
}

variable "app_secrets" {
  description = "Map of sensitive environment variable names and values"
  type        = map(string)
  default     = {}
  sensitive   = false
}

# Example usage:
# app_env_vars = {
#   "NODE_ENV"        = "production"
#   "LOG_LEVEL"       = "info"
#   "API_VERSION"     = "v1"
# }
#
# app_secrets = {
#   "JWT_SECRET"      = "supersecretjwt"
#   "DB_PASSWORD"     = "mypassword"
#   "API_KEY"         = "abc123"
# } 