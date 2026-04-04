# ============================================================
# Bootstrap для ML-аккаунта
# Создаёт: сервисный аккаунт, S3-бакет для terraform state,
# static access keys для S3 backend.
#
# Использует ЛОКАЛЬНЫЙ state (бакет ещё не существует).
# После apply основной accounts/ml/ использует созданный бакет.
# ============================================================

# --- Сервисный аккаунт для Terraform ---

resource "yandex_iam_service_account" "terraform" {
  name        = "terraform-${var.account_name}"
  description = "Сервисный аккаунт для Terraform (${var.account_name})"
  folder_id   = var.folder_id
}

# Роли для SA
resource "yandex_resourcemanager_folder_iam_member" "editor" {
  folder_id = var.folder_id
  role      = "editor"
  member    = "serviceAccount:${yandex_iam_service_account.terraform.id}"
}

resource "yandex_resourcemanager_folder_iam_member" "registry_admin" {
  folder_id = var.folder_id
  role      = "container-registry.admin"
  member    = "serviceAccount:${yandex_iam_service_account.terraform.id}"
}

# Authorized key (для provider yandex в основном terraform)
resource "yandex_iam_service_account_key" "terraform" {
  service_account_id = yandex_iam_service_account.terraform.id
  description        = "Ключ для Terraform provider"
}

# --- Static Access Keys для S3 backend ---

resource "yandex_iam_service_account_static_access_key" "s3" {
  service_account_id = yandex_iam_service_account.terraform.id
  description        = "S3 access key для Terraform state"
}

# --- S3-бакет для Terraform state ---

resource "yandex_storage_bucket" "tfstate" {
  bucket     = var.bucket_name
  access_key = yandex_iam_service_account_static_access_key.s3.access_key
  secret_key = yandex_iam_service_account_static_access_key.s3.secret_key

  versioning {
    enabled = true
  }
}
