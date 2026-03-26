output "network_id" {
  description = "ID VPC сети"
  value       = yandex_vpc_network.this.id
}

output "subnet_ids" {
  description = "Карта имя → ID подсети"
  value       = { for k, v in yandex_vpc_subnet.this : k => v.id }
}

output "web_security_group_id" {
  description = "ID security group для web-серверов"
  value       = yandex_vpc_security_group.web.id
}

output "db_security_group_id" {
  description = "ID security group для managed БД"
  value       = yandex_vpc_security_group.db.id
}
