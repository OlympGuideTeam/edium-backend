output "test_vm_ip" {
  description = "Внешний IP test VM"
  value       = module.vm_test.external_ip
}

output "prod_vm_ip" {
  description = "Внешний IP prod VM"
  value       = module.vm_prod.external_ip
}

output "registry_id" {
  description = "ID Container Registry"
  value       = module.registry.registry_id
}

output "postgres_host" {
  description = "FQDN хоста PostgreSQL"
  value       = module.postgres.cluster_host
}

output "redis_host" {
  description = "FQDN хоста Redis"
  value       = module.redis.cluster_host
}

output "dns_zone_id" {
  description = "ID DNS-зоны"
  value       = module.dns.zone_id
}
