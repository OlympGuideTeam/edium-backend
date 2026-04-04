# Bootstrap ML-аккаунта в Yandex Cloud

Пошаговая инструкция от нуля до `terraform apply` для ML-инфраструктуры.

## Предварительные требования

- `yc` CLI установлен (`curl -sSL https://storage.yandexcloud.net/yandexcloud-yc/install.sh | bash`)
- `terraform` >= 1.5 установлен
- `.terraformrc` с зеркалом YC (если из России):

```bash
cat > ~/.terraformrc <<'EOF'
provider_installation {
  network_mirror {
    url = "https://terraform-mirror.yandexcloud.net/"
    include = ["registry.terraform.io/*/*"]
  }
  direct {
    exclude = ["registry.terraform.io/*/*"]
  }
}
EOF
```

---

## Шаг 1. Создать облако и каталог (консоль YC)

Вручную в [console.yandex.cloud](https://console.yandex.cloud):

1. Создать новое **облако** (или использовать существующее)
2. Создать **каталог** (folder) — например `edium-ml`
3. Привязать платёжный аккаунт

Запомнить:
- `cloud_id` — из карточки облака
- `folder_id` — из карточки каталога

## Шаг 2. Авторизоваться в yc CLI

```bash
yc init
# Выбрать ML-облако и каталог
# Авторизация через браузер → OAuth-токен сохранится локально
```

Проверка:
```bash
yc config get cloud-id    # должен показать cloud_id
yc config get folder-id   # должен показать folder_id
yc config get token       # OAuth-токен (понадобится для terraform)
```

## Шаг 3. Bootstrap — создать SA и S3-бакет через Terraform

```bash
cd infra/terraform/bootstrap

terraform init

terraform apply \
  -var="cloud_id=$(yc config get cloud-id)" \
  -var="folder_id=$(yc config get folder-id)" \
  -var="yc_token=$(yc config get token)"
```

Terraform создаст:
- Сервисный аккаунт `terraform-ml` с ролями `editor`, `container-registry.admin`
- Authorized key для terraform provider
- Static access keys для S3 backend
- S3-бакет `edium-tf-state-ml` с версионированием

## Шаг 4. Сохранить output-ы

```bash
# SA key для terraform provider — сохранить в файл
terraform output -raw sa_key_json > ../accounts/ml/key.json

# S3-ключи для backend — запомнить
terraform output s3_access_key
terraform output s3_secret_key
```

> **key.json** добавлен в .gitignore — не коммитить!

## Шаг 5. Сгенерировать SSH-ключ для GPU VM

```bash
ssh-keygen -t ed25519 -f ~/.ssh/edium_ml_deploy -N ""
```

## Шаг 6. Создать terraform.tfvars для ML

```bash
cat > ../accounts/ml/terraform.tfvars <<EOF
cloud_id       = "$(yc config get cloud-id)"
folder_id      = "$(yc config get folder-id)"
ssh_public_key = "$(cat ~/.ssh/edium_ml_deploy.pub)"
EOF
```

## Шаг 7. Запустить основной Terraform

```bash
cd ../accounts/ml

# S3-ключи из шага 4
export AWS_ACCESS_KEY_ID=<s3_access_key>
export AWS_SECRET_ACCESS_KEY=<s3_secret_key>

terraform init
terraform plan
terraform apply
```

Результат:
```
ml_vm_ip       = "..."     # внешний IP GPU VM
ml_registry_id = "..."     # ID container registry
```

## Шаг 8. DNS-запись (на основном аккаунте)

```bash
cd ../accounts/platform

terraform apply -var="sphinx_vm_ip=<ml_vm_ip из шага 7>"
```

Создаст `sphinx.ml.edium.online → <GPU VM IP>`.

## Шаг 9. Добавить секреты в GitHub

```bash
# ML-аккаунт
gh secret set ML_REGISTRY_ID -b "<ml_registry_id>"
gh secret set ML_VM_HOST -b "<ml_vm_ip>"
gh secret set ML_VM_SSH_KEY < ~/.ssh/edium_ml_deploy
gh secret set ML_YC_SA_KEY_JSON < ../accounts/ml/key.json
gh secret set AWS_ACCESS_KEY_ID_ML -b "<s3_access_key>"
gh secret set AWS_SECRET_ACCESS_KEY_ML -b "<s3_secret_key>"
gh secret set TF_VARS_ML < ../accounts/ml/terraform.tfvars

# Sphinx
gh secret set SPHINX_API_KEY -b "$(openssl rand -hex 32)"
gh secret set HF_TOKEN -b "hf_..."
gh secret set SPHINX_GRPC_ADDR -b "sphinx.ml.edium.online:443"
```

## Шаг 10. Установить NVIDIA драйвер на VM (первый раз)

GPU-драйвер устанавливается через cloud-init автоматически. Но если нужно проверить:

```bash
ssh -i ~/.ssh/edium_ml_deploy deploy@<ml_vm_ip>
nvidia-smi    # должна показать Tesla T4
docker run --rm --gpus all nvidia/cuda:12.1.1-base-ubuntu22.04 nvidia-smi
```

---

## Итоговая структура

```
bootstrap/          # Шаг 3: SA + S3 (локальный state)
accounts/ml/        # Шаг 7: GPU VM + Registry (state в S3)
accounts/platform/  # Шаг 8: DNS-запись sphinx.ml (state в S3)
```
