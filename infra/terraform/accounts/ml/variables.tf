variable "cloud_id" {
  description = "ID облака Yandex Cloud"
  type        = string
}

variable "folder_id" {
  description = "ID каталога Yandex Cloud"
  type        = string
}

variable "sa_key_file" {
  description = "Путь к файлу ключа сервисного аккаунта"
  type        = string
  default     = "key.json"
}

variable "ssh_public_key" {
  description = "SSH публичный ключ для доступа к VM"
  type        = string
}

variable "ssh_allowed_cidrs" {
  description = "CIDR-блоки, откуда разрешён SSH"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}
