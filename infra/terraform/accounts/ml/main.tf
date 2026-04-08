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

# --- Статический IP для GPU VM ---

resource "yandex_vpc_address" "sphinx" {
  name      = "edium-sphinx"
  folder_id = var.folder_id

  external_ipv4_address {
    zone_id = "ru-central1-a"
  }
}

# --- Cloud Function: авто-рестарт прерываемой VM ---

data "archive_file" "sphinx_restarter" {
  type        = "zip"
  source_dir  = "${path.module}/functions/restarter"
  output_path = "${path.module}/functions/restarter.zip"
}

resource "yandex_iam_service_account" "sphinx_restarter" {
  name      = "edium-sphinx-restarter"
  folder_id = var.folder_id
}

resource "yandex_resourcemanager_folder_iam_member" "sphinx_restarter_compute" {
  folder_id = var.folder_id
  role      = "compute.operator"
  member    = "serviceAccount:${yandex_iam_service_account.sphinx_restarter.id}"
}

resource "yandex_resourcemanager_folder_iam_member" "sphinx_restarter_invoker" {
  folder_id = var.folder_id
  role      = "serverless.functions.invoker"
  member    = "serviceAccount:${yandex_iam_service_account.sphinx_restarter.id}"
}

resource "yandex_function" "sphinx_restarter" {
  name               = "sphinx-vm-restarter"
  folder_id          = var.folder_id
  runtime            = "python312"
  entrypoint         = "handler.handler"
  memory             = 128
  execution_timeout  = "30"
  service_account_id = yandex_iam_service_account.sphinx_restarter.id

  environment = {
    VM_ID = module.vm_gpu.instance_id
  }

  user_hash = data.archive_file.sphinx_restarter.output_sha256

  content {
    zip_filename = data.archive_file.sphinx_restarter.output_path
  }

  depends_on = [
    yandex_resourcemanager_folder_iam_member.sphinx_restarter_compute,
  ]
}

resource "yandex_function_trigger" "sphinx_restarter_timer" {
  name      = "sphinx-restarter-timer"
  folder_id = var.folder_id

  timer {
    cron_expression = "0 */5 * * ? *"
  }

  function {
    id                 = yandex_function.sphinx_restarter.id
    service_account_id = yandex_iam_service_account.sphinx_restarter.id
  }
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
  image_id           = "fd80on0ma1ic60hees6n"
  disk_size          = 60
  disk_type          = "network-ssd"
  nat_ip_address     = yandex_vpc_address.sphinx.external_ipv4_address[0].address
  subnet_id          = module.vpc.subnet_ids["edium-ml"]
  security_group_ids = [module.vpc.web_security_group_id]
  ssh_public_key     = var.ssh_public_key
}
