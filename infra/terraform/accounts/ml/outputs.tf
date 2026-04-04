output "ml_vm_ip" {
  description = "Внешний IP GPU VM (для DNS-записи sphinx.ml.edium.online)"
  value       = module.vm_gpu.external_ip
}

output "ml_vm_internal_ip" {
  description = "Внутренний IP GPU VM"
  value       = module.vm_gpu.internal_ip
}

output "ml_registry_id" {
  description = "ID Container Registry ML-аккаунта"
  value       = module.registry.registry_id
}
