variable "project_id" {
  description = "The GCP project ID"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "repository_name" {
  description = "Name of the Artifact Registry repository"
  type        = string
}

variable "location" {
  description = "Location of the Artifact Registry repository"
  type        = string
}

variable "format" {
  description = "Format of the Artifact Registry repository"
  type        = string
  default     = "DOCKER"
}

variable "description" {
  description = "Description of the Artifact Registry repository"
  type        = string
  default     = "Kavach application container images"
} 