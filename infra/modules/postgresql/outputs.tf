output "instance_name" {
  description = "The name of the database instance"
  value       = google_sql_database_instance.instance.name
}

output "instance_connection_name" {
  description = "The connection name of the database instance"
  value       = google_sql_database_instance.instance.connection_name
  sensitive   = true
}

output "private_ip_address" {
  description = "The private IP address of the database instance"
  value       = google_sql_database_instance.instance.private_ip_address
  sensitive   = true
}

output "public_ip_address" {
  description = "The public IP address of the database instance"
  value       = google_sql_database_instance.instance.public_ip_address
  sensitive   = true
}

output "database_name" {
  description = "The name of the database"
  value       = google_sql_database.database.name
}

output "user_name" {
  description = "The name of the database user"
  value       = google_sql_user.user.name
}

output "password_secret_id" {
  description = "The Secret Manager secret ID for the database password"
  value       = google_secret_manager_secret.db_password.secret_id
}

output "password" {
  description = "The database password (sensitive)"
  value       = random_password.db_password.result
  sensitive   = true
} 