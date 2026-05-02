# Object Storage для Louvre (S3 API — те же переменные, что для MinIO SDK в приложении).
# Отдельные yandex_storage_bucket_grant — добавляют права SA Louvre, не затирая ACL владельца бакета.

resource "yandex_iam_service_account" "louvre" {
  name        = "edium-louvre-storage"
  description = "Доступ приложения Louvre к Object Storage"
  folder_id   = var.folder_id
}

resource "yandex_storage_bucket" "test" {
  bucket    = var.bucket_test_name
  folder_id = var.folder_id
}

resource "yandex_storage_bucket" "prod" {
  bucket    = var.bucket_prod_name
  folder_id = var.folder_id
}

resource "yandex_storage_bucket_grant" "louvre_test" {
  bucket = yandex_storage_bucket.test.bucket
  grant {
    type        = "CanonicalUser"
    id          = yandex_iam_service_account.louvre.id
    permissions = ["FULL_CONTROL"]
  }
}

resource "yandex_storage_bucket_grant" "louvre_prod" {
  bucket = yandex_storage_bucket.prod.bucket
  grant {
    type        = "CanonicalUser"
    id          = yandex_iam_service_account.louvre.id
    permissions = ["FULL_CONTROL"]
  }
}

resource "yandex_iam_service_account_static_access_key" "louvre" {
  service_account_id = yandex_iam_service_account.louvre.id
  description        = "Louvre: S3 API для бакетов test и prod"

  depends_on = [
    yandex_storage_bucket_grant.louvre_test,
    yandex_storage_bucket_grant.louvre_prod,
  ]
}
