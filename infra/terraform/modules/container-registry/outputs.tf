output "registry_id" {
  description = "ID Container Registry"
  value       = yandex_container_registry.this.id
}

output "registry_name" {
  description = "Имя Container Registry"
  value       = yandex_container_registry.this.name
}
