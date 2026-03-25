variable "cluster_name" {
  description = "Имя кластера Redis"
  type        = string
  default     = "edium-redis"
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
  description = "Класс ресурсов (hm3-c2-m8)"
  type        = string
  default     = "b3-c1-m4"
}

variable "disk_size" {
  description = "Размер диска в GB"
  type        = number
  default     = 8
}

variable "password" {
  description = "Пароль Redis"
  type        = string
  sensitive   = true
}

variable "security_group_ids" {
  description = "ID security groups"
  type        = list(string)
  default     = []
}
