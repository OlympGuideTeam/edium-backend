output "ml_vm_id" {
  description = "ID GPU VM (для управления через YC API / CI-деплоя)"
  value       = module.vm_gpu.instance_id
}

output "ml_vm_ip" {
  description = "Статический внешний IP GPU VM (для DNS-записи sphinx.ml.edium.online)"
  value       = yandex_vpc_address.sphinx.external_ipv4_address[0].address
}

output "ml_vm_internal_ip" {
  description = "Внутренний IP GPU VM"
  value       = module.vm_gpu.internal_ip
}

output "ml_registry_id" {
  description = "ID Container Registry ML-аккаунта"
  value       = module.registry.registry_id
}
