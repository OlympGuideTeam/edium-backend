output "s3_endpoint" {
  description = "Хост без схемы для LOUVRE_MINIO_ENDPOINT"
  value       = "storage.yandexcloud.net"
}

output "use_ssl" {
  description = "Для LOUVRE_MINIO_USE_SSL при Object Storage — true"
  value       = true
}

output "bucket_test" {
  description = "LOUVRE_MINIO_BUCKET на test VM"
  value       = yandex_storage_bucket.test.bucket
}

output "bucket_prod" {
  description = "LOUVRE_MINIO_BUCKET на prod VM"
  value       = yandex_storage_bucket.prod.bucket
}

output "static_access_key" {
  description = "LOUVRE_MINIO_ACCESS_KEY"
  value       = yandex_iam_service_account_static_access_key.louvre.access_key
  sensitive   = true
}

output "static_secret_key" {
  description = "LOUVRE_MINIO_SECRET_KEY"
  value       = yandex_iam_service_account_static_access_key.louvre.secret_key
  sensitive   = true
}
