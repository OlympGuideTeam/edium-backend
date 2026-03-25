variable "cluster_name" {
  description = "Имя кластера PostgreSQL"
  type        = string
  default     = "edium-pg"
}

variable "network_id" {
  description = "ID VPC сети"
  type        = string
}

variable "subnet_id" {
  description = "ID подсети для хоста"
  type        = string
}

variable "zone" {
  description = "Зона доступности"
  type        = string
  default     = "ru-central1-a"
}

variable "resource_preset_id" {
  description = "Класс ресурсов (c3-c2-m4)"
  type        = string
  default     = "c3-c2-m4"
}

variable "disk_size" {
  description = "Размер диска в GB"
  type        = number
  default     = 10
}

variable "security_group_ids" {
  description = "ID security groups"
  type        = list(string)
  default     = []
}

variable "databases" {
  description = "Базы данных: имя → {owner}"
  type = map(object({
    owner = string
  }))
}

variable "users" {
  description = "Пользователи: имя → {password}"
  type = map(object({
    password = string
  }))
}
