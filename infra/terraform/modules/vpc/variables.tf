variable "network_name" {
  description = "Имя VPC сети"
  type        = string
}

variable "subnets" {
  description = "Карта подсетей: имя → {zone, cidr_blocks}"
  type = map(object({
    zone        = string
    cidr_blocks = list(string)
  }))
}

variable "ssh_allowed_cidrs" {
  description = "CIDR-блоки, откуда разрешён SSH"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}
