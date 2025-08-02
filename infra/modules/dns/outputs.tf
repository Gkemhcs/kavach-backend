output "a_record_name" {
  description = "The name of the A record"
  value       = google_dns_record_set.a_record.name
}

output "a_record_ip" {
  description = "The IP address in the A record"
  value       = google_dns_record_set.a_record.rrdatas
}


output "dns_zone_name" {
  description = "The name of the DNS zone"
  value       = data.google_dns_managed_zone.zone.name
} 