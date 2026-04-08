data "yandex_compute_image" "ubuntu" {
  count  = var.image_id == "" ? 1 : 0
  family = var.gpus > 0 ? "ubuntu-2204-lts-gpu" : "ubuntu-2204-lts"
}

resource "yandex_compute_instance" "this" {
  name        = var.name
  platform_id = var.platform_id
  zone        = var.zone

  resources {
    cores         = var.cores
    memory        = var.memory
    core_fraction = var.core_fraction
    gpus          = var.gpus > 0 ? var.gpus : null
  }

  boot_disk {
    initialize_params {
      image_id = var.image_id != "" ? var.image_id : data.yandex_compute_image.ubuntu[0].id
      size     = var.disk_size
      type     = var.disk_type
    }
  }

  network_interface {
    subnet_id          = var.subnet_id
    nat                = true
    nat_ip_address     = var.nat_ip_address != "" ? var.nat_ip_address : null
    security_group_ids = var.security_group_ids
  }

  metadata = {
    user-data = templatefile(
      var.gpu_cloud_init
      ? "${path.module}/cloud-init-gpu.yaml"
      : "${path.module}/cloud-init.yaml",
      {
        ssh_public_key = var.ssh_public_key
        username       = var.username
      }
    )
  }

  scheduling_policy {
    preemptible = var.preemptible
  }
}
