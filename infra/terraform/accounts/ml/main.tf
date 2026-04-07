# ============================================================
# Edium ML — отдельный YC аккаунт: GPU VM + Container Registry
# ============================================================

module "vpc" {
  source = "../../modules/vpc"

  network_name = "edium-ml"
  subnets = {
    "edium-ml" = {
      zone        = "ru-central1-a"
      cidr_blocks = ["10.3.0.0/24"]
    }
  }
  ssh_allowed_cidrs = var.ssh_allowed_cidrs
}

# --- Container Registry ---

module "registry" {
  source = "../../modules/container-registry"

  registry_name = "edium-ml"
  folder_id     = var.folder_id
}

# --- GPU VM (прерываемая Tesla T4) ---

module "vm_gpu" {
  source = "../../modules/compute"

  name               = "edium-sphinx"
  platform_id        = "standard-v3-t4"
  cores              = 4
  memory             = 16
  gpus               = 1
  gpu_cloud_init     = true
  preemptible        = true
  disk_size          = 60
  disk_type          = "network-ssd"
  subnet_id          = module.vpc.subnet_ids["edium-ml"]
  security_group_ids = [module.vpc.web_security_group_id]
  ssh_public_key     = var.ssh_public_key
}
