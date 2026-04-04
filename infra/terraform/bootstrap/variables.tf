variable "cloud_id" {
  description = "ID облака Yandex Cloud"
  type        = string
}

variable "folder_id" {
  description = "ID каталога Yandex Cloud"
  type        = string
}

variable "yc_token" {
  description = "OAuth-токен для Yandex Cloud (из `yc config get token`)"
  type        = string
  sensitive   = true
}

variable "bucket_name" {
  description = "Имя S3-бакета для terraform state"
  type        = string
  default     = "edium-tf-state-ml"
}

variable "account_name" {
  description = "Префикс для ресурсов (terraform-<account_name>)"
  type        = string
  default     = "ml"
}
