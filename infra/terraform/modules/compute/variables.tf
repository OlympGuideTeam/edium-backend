variable "name" {
  description = "Имя VM"
  type        = string
}

variable "platform_id" {
  description = "Платформа (standard-v3)"
  type        = string
  default     = "standard-v3"
}

variable "zone" {
  description = "Зона доступности"
  type        = string
  default     = "ru-central1-a"
}

variable "cores" {
  description = "Количество vCPU"
  type        = number
  default     = 2
}

variable "memory" {
  description = "RAM в GB"
  type        = number
  default     = 4
}

variable "core_fraction" {
  description = "Гарантированная доля vCPU (%)"
  type        = number
  default     = 100
}

variable "disk_size" {
  description = "Размер диска в GB"
  type        = number
  default     = 30
}

variable "disk_type" {
  description = "Тип диска (network-hdd, network-ssd)"
  type        = string
  default     = "network-hdd"
}

variable "subnet_id" {
  description = "ID подсети"
  type        = string
}

variable "security_group_ids" {
  description = "Список ID security groups"
  type        = list(string)
  default     = []
}

variable "ssh_public_key" {
  description = "SSH публичный ключ для доступа"
  type        = string
}

variable "username" {
  description = "Имя пользователя на VM"
  type        = string
  default     = "deploy"
}

variable "preemptible" {
  description = "Прерываемая VM (дешевле, но может быть остановлена)"
  type        = bool
  default     = false
}

variable "gpus" {
  description = "Количество GPU (0 = без GPU)"
  type        = number
  default     = 0
}

variable "gpu_cloud_init" {
  description = "Использовать cloud-init с NVIDIA Container Toolkit"
  type        = bool
  default     = false
}

variable "nat_ip_address" {
  description = "Статический внешний IP (пусто — динамический)"
  type        = string
  default     = ""
}

variable "image_id" {
  description = "ID образа диска (пусто — автовыбор по family)"
  type        = string
  default     = ""
}
