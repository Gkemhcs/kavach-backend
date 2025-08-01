output "vpc_id" {
  description = "The ID of the VPC"
  value       = google_compute_network.vpc.id
}

output "vpc_name" {
  description = "The name of the VPC"
  value       = google_compute_network.vpc.name
}

output "vpc_self_link" {
  description = "The self-link of the VPC"
  value       = google_compute_network.vpc.self_link
}

output "subnet_id" {
  description = "The ID of the subnet"
  value       = google_compute_subnetwork.subnet.id
}

output "subnet_name" {
  description = "The name of the subnet"
  value       = google_compute_subnetwork.subnet.name
}

output "subnet_self_link" {
  description = "The self-link of the subnet"
  value       = google_compute_subnetwork.subnet.self_link
}

output "cloud_run_subnet_id" {
  description = "The ID of the Cloud Run subnet"
  value       = google_compute_subnetwork.cloud_run_subnet.id
}

output "cloud_run_subnet_name" {
  description = "The name of the Cloud Run subnet"
  value       = google_compute_subnetwork.cloud_run_subnet.name
}

output "cloud_run_subnet_self_link" {
  description = "The self-link of the Cloud Run subnet"
  value       = google_compute_subnetwork.cloud_run_subnet.self_link
} 