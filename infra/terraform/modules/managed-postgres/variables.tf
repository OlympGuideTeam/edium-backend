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

variable "default_user_conn_limit" {
  description = "Лимит соединений на пользователя, если в записи не задан conn_limit (Managed PG: сумма по всем юзерам + системные ≤ max_connections)"
  type        = number
  default     = 8
}

variable "users" {
  description = "Пользователи: имя → {password, conn_limit?}"
  type = map(object({
    password   = string
    conn_limit = optional(number)
  }))
}
