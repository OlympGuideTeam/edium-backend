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
  }

  users = {
    "doorman_test" = { password = var.doorman_pg_password_test }
    "doorman_prod" = { password = var.doorman_pg_password_prod }
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

# --- DNS ---

module "dns" {
  source = "../../modules/dns"

  domain = var.domain
  records = {
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
  }
}
