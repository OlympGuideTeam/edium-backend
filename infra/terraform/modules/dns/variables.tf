variable "zone_name" {
  description = "Имя DNS-зоны в Yandex Cloud"
  type        = string
  default     = "edium-online"
}

variable "domain" {
  description = "Доменное имя (без точки в конце)"
  type        = string
  default     = "edium.online"
}

variable "records" {
  description = "DNS-записи: имя → {type, ttl, data}"
  type = map(object({
    type = string
    ttl  = number
    data = list(string)
  }))
}
