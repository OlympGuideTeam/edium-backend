output "ml_vm_ip" {
  description = "Внешний IP ML VM"
  value       = module.vm_ml.external_ip
}

output "ml_vm_internal_ip" {
  description = "Внутренний IP ML VM"
  value       = module.vm_ml.internal_ip
}
