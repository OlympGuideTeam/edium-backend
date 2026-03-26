resource "yandex_vpc_network" "this" {
  name        = var.network_name
  description = "Сеть ${var.network_name} для Edium"
}

resource "yandex_vpc_subnet" "this" {
  for_each = var.subnets

  name           = each.key
  zone           = each.value.zone
  network_id     = yandex_vpc_network.this.id
  v4_cidr_blocks = each.value.cidr_blocks
}

resource "yandex_vpc_security_group" "web" {
  name       = "${var.network_name}-web"
  network_id = yandex_vpc_network.this.id

  ingress {
    description    = "HTTP"
    protocol       = "TCP"
    port           = 80
    v4_cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description    = "HTTPS"
    protocol       = "TCP"
    port           = 443
    v4_cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description    = "SSH"
    protocol       = "TCP"
    port           = 22
    v4_cidr_blocks = var.ssh_allowed_cidrs
  }

  ingress {
    description    = "Prometheus node-exporter"
    protocol       = "TCP"
    port           = 9100
    v4_cidr_blocks = [for s in yandex_vpc_subnet.this : s.v4_cidr_blocks[0]]
  }

  egress {
    description    = "Allow all outbound"
    protocol       = "ANY"
    v4_cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "yandex_vpc_security_group" "db" {
  name       = "${var.network_name}-db"
  network_id = yandex_vpc_network.this.id

  ingress {
    description    = "PostgreSQL"
    protocol       = "TCP"
    port           = 6432
    v4_cidr_blocks = [for s in yandex_vpc_subnet.this : s.v4_cidr_blocks[0]]
  }

  ingress {
    description    = "Redis"
    protocol       = "TCP"
    port           = 6379
    v4_cidr_blocks = [for s in yandex_vpc_subnet.this : s.v4_cidr_blocks[0]]
  }

  egress {
    description    = "Allow all outbound"
    protocol       = "ANY"
    v4_cidr_blocks = ["0.0.0.0/0"]
  }
}
