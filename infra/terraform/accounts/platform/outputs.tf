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

# --- Louvre / Object Storage (секреты → GitHub Environments test и production) ---

output "louvre_s3_endpoint" {
  description = "LOUVRE_MINIO_ENDPOINT"
  value       = module.louvre_storage.s3_endpoint
}

output "louvre_s3_use_ssl" {
  description = "LOUVRE_MINIO_USE_SSL — для Object Storage всегда true"
  value       = module.louvre_storage.use_ssl
}

output "louvre_s3_bucket_test" {
  description = "LOUVRE_MINIO_BUCKET на test"
  value       = module.louvre_storage.bucket_test
}

output "louvre_s3_bucket_prod" {
  description = "LOUVRE_MINIO_BUCKET на prod"
  value       = module.louvre_storage.bucket_prod
}

output "louvre_s3_access_key" {
  description = "LOUVRE_MINIO_ACCESS_KEY"
  value       = module.louvre_storage.static_access_key
  sensitive   = true
}

output "louvre_s3_secret_key" {
  description = "LOUVRE_MINIO_SECRET_KEY"
  value       = module.louvre_storage.static_secret_key
  sensitive   = true
}
