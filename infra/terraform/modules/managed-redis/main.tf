resource "yandex_mdb_redis_cluster" "this" {
  name        = var.cluster_name
  environment = "PRODUCTION"
  network_id  = var.network_id

  config {
    password = var.password
    version  = "7.2-valkey"
  }

  resources {
    resource_preset_id = var.resource_preset_id
    disk_type_id       = "network-ssd"
    disk_size          = var.disk_size
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
    hour = 5
  }
}
