# ============================================================
# Edium ML — Account 1: ML VM
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

module "vm_ml" {
  source = "../../modules/compute"

  name               = "edium-ml"
  cores              = 8
  memory             = 16
  disk_size          = 100
  disk_type          = "network-ssd"
  subnet_id          = module.vpc.subnet_ids["edium-ml"]
  security_group_ids = [module.vpc.web_security_group_id]
  ssh_public_key     = var.ssh_public_key
}
