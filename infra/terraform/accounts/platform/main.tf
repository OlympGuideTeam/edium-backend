# ============================================================
# Edium Platform — Account 2: test + prod
# ============================================================

# --- VPC ---

module "vpc" {
  source = "../../modules/vpc"

  network_name = "edium-platform"
  subnets = {
    "edium-test" = {
      zone        = "ru-central1-a"
      cidr_blocks = ["10.1.0.0/24"]
    }
    "edium-prod" = {
      zone        = "ru-central1-a"
      cidr_blocks = ["10.2.0.0/24"]
    }
  }
  ssh_allowed_cidrs = var.ssh_allowed_cidrs
}

# --- Container Registry ---

module "registry" {
  source = "../../modules/container-registry"

  registry_name = "edium"
  folder_id     = var.folder_id
}

# --- Static IPs ---

resource "yandex_vpc_address" "test" {
  name = "edium-test-ip"

  external_ipv4_address {
    zone_id = "ru-central1-a"
  }
}

resource "yandex_vpc_address" "prod" {
  name = "edium-prod-ip"

  external_ipv4_address {
    zone_id = "ru-central1-a"
  }
}

# --- Test VM ---

module "vm_test" {
  source = "../../modules/compute"

  name               = "edium-test"
  cores              = 2
  memory             = 4
  disk_size          = 30
  disk_type          = "network-hdd"
  subnet_id          = module.vpc.subnet_ids["edium-test"]
  security_group_ids = [module.vpc.web_security_group_id]
  ssh_public_key     = var.ssh_public_key
  nat_ip_address     = yandex_vpc_address.test.external_ipv4_address[0].address
}

# --- Prod VM ---

module "vm_prod" {
  source = "../../modules/compute"

  name               = "edium-prod"
  cores              = 2
  memory             = 4
  disk_size          = 30
  disk_type          = "network-hdd"
  subnet_id          = module.vpc.subnet_ids["edium-prod"]
  security_group_ids = [module.vpc.web_security_group_id]
  ssh_public_key     = var.ssh_public_key
  nat_ip_address     = yandex_vpc_address.prod.external_ipv4_address[0].address
}

# --- Managed PostgreSQL ---

module "postgres" {
  source = "../../modules/managed-postgres"

  cluster_name       = "edium-pg"
  network_id         = module.vpc.network_id
  subnet_id          = module.vpc.subnet_ids["edium-prod"]
  security_group_ids = [module.vpc.db_security_group_id]

  databases = {
    "doorman_test" = { owner = "doorman_test" }
    "doorman_prod" = { owner = "doorman_prod" }
    "herald_test"  = { owner = "herald_test" }
    "herald_prod"  = { owner = "herald_prod" }
    "charon_test"  = { owner = "charon_test" }
    "charon_prod"  = { owner = "charon_prod" }
    "caesar_test"  = { owner = "caesar_test" }
    "caesar_prod"  = { owner = "caesar_prod" }
    "sphinx"       = { owner = "sphinx" }
  }

  users = {
    "doorman_test" = { password = var.doorman_pg_password_test }
    "doorman_prod" = { password = var.doorman_pg_password_prod }
    "herald_test"  = { password = var.herald_pg_password_test }
    "herald_prod"  = { password = var.herald_pg_password_prod }
    "charon_test"  = { password = var.charon_pg_password_test }
    "charon_prod"  = { password = var.charon_pg_password_prod }
    "caesar_test"  = { password = var.caesar_pg_password_test }
    "caesar_prod"  = { password = var.caesar_pg_password_prod }
    "sphinx"       = { password = var.sphinx_pg_password }
  }
}

# --- Managed Redis ---

module "redis" {
  source = "../../modules/managed-redis"

  cluster_name       = "edium-redis"
  network_id         = module.vpc.network_id
  subnet_id          = module.vpc.subnet_ids["edium-prod"]
  password           = var.redis_password
  security_group_ids = [module.vpc.db_security_group_id]
}

# --- Доступ к PG из ML-аккаунта (Sphinx GPU VM) ---
# Создаётся только если sphinx_vm_ip задан

resource "yandex_vpc_security_group_rule" "sphinx_pg" {
  count = var.sphinx_vm_ip != "" ? 1 : 0

  security_group_binding = module.vpc.db_security_group_id
  direction              = "ingress"
  description            = "PostgreSQL from Sphinx ML VM"
  protocol               = "TCP"
  port                   = 6432
  v4_cidr_blocks         = ["${var.sphinx_vm_ip}/32"]
}

# --- DNS ---

locals {
  base_dns_records = {
    "api" = {
      type = "A"
      ttl  = 300
      data = [module.vm_prod.external_ip]
    }
    "test" = {
      type = "A"
      ttl  = 300
      data = [module.vm_test.external_ip]
    }
    "grafana" = {
      type = "A"
      ttl  = 300
      data = [module.vm_prod.external_ip]
    }
    "grafana-test" = {
      type = "A"
      ttl  = 300
      data = [module.vm_test.external_ip]
    }
  }

  sphinx_dns_record = var.sphinx_vm_ip != "" ? {
    "sphinx.ml" = {
      type = "A"
      ttl  = 300
      data = [var.sphinx_vm_ip]
    }
  } : {}
}

module "dns" {
  source = "../../modules/dns"

  domain  = var.domain
  records = merge(local.base_dns_records, local.sphinx_dns_record)
}
