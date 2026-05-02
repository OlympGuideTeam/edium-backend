resource "yandex_mdb_postgresql_cluster" "this" {
  name        = var.cluster_name
  environment = "PRODUCTION"
  network_id  = var.network_id

  config {
    version = "17"

    resources {
      resource_preset_id = var.resource_preset_id
      disk_type_id       = "network-ssd"
      disk_size          = var.disk_size
    }

    postgresql_config = {
      # Для пресета Managed PG верхняя граница часто 400; нагрузку на лимит снижаем conn_limit у пользователей
      max_connections = 400
    }
  }

  host {
    zone             = var.zone
    subnet_id        = var.subnet_id
    assign_public_ip = false
  }

  security_group_ids = var.security_group_ids

  maintenance_window {
    type = "WEEKLY"
    day  = "SUN"
    hour = 4
  }
}

# Сначала создаём юзеров без permissions (чтобы не зависеть от БД)
resource "yandex_mdb_postgresql_user" "users" {
  for_each = var.users

  cluster_id = yandex_mdb_postgresql_cluster.this.id
  name       = each.key
  password   = each.value.password
  conn_limit = coalesce(each.value.conn_limit, var.default_user_conn_limit)
}

# Потом БД с owner
resource "yandex_mdb_postgresql_database" "databases" {
  for_each = var.databases

  cluster_id = yandex_mdb_postgresql_cluster.this.id
  name       = each.key
  owner      = each.value.owner

  depends_on = [yandex_mdb_postgresql_user.users]
}
