output "service_account_id" {
  description = "ID сервисного аккаунта"
  value       = yandex_iam_service_account.terraform.id
}

output "sa_key_id" {
  description = "ID authorized key (для key.json)"
  value       = yandex_iam_service_account_key.terraform.id
}

output "sa_key_json" {
  description = "Authorized key JSON (сохранить в файл для terraform provider)"
  value = jsonencode({
    id                 = yandex_iam_service_account_key.terraform.id
    service_account_id = yandex_iam_service_account.terraform.id
    created_at         = yandex_iam_service_account_key.terraform.created_at
    key_algorithm      = yandex_iam_service_account_key.terraform.key_algorithm
    public_key         = yandex_iam_service_account_key.terraform.public_key
    private_key        = yandex_iam_service_account_key.terraform.private_key
  })
  sensitive = true
}

output "s3_access_key" {
  description = "AWS_ACCESS_KEY_ID для S3 backend"
  value       = yandex_iam_service_account_static_access_key.s3.access_key
  sensitive   = true
}

output "s3_secret_key" {
  description = "AWS_SECRET_ACCESS_KEY для S3 backend"
  value       = yandex_iam_service_account_static_access_key.s3.secret_key
  sensitive   = true
}

output "bucket_name" {
  description = "Имя S3-бакета"
  value       = yandex_storage_bucket.tfstate.bucket
}
