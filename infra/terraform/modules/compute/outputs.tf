output "instance_id" {
  description = "ID VM"
  value       = yandex_compute_instance.this.id
}

output "external_ip" {
  description = "Внешний IP VM"
  value       = yandex_compute_instance.this.network_interface[0].nat_ip_address
}

output "internal_ip" {
  description = "Внутренний IP VM"
  value       = yandex_compute_instance.this.network_interface[0].ip_address
}

output "fqdn" {
  description = "FQDN VM"
  value       = yandex_compute_instance.this.fqdn
}
