terraform {
  required_version = ">= 1.5"

  required_providers {
    yandex = {
      source  = "yandex-cloud/yandex"
      version = "~> 0.130"
    }
  }

  # Локальный state — S3-бакет ещё не существует
}

provider "yandex" {
  cloud_id  = var.cloud_id
  folder_id = var.folder_id
  zone      = "ru-central1-a"
  # Авторизация через yc CLI (OAuth token) — SA ещё не создан
  # При запуске terraform будет использовать токен из `yc config get token`
  token = var.yc_token
}
