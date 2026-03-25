#!/usr/bin/env bash
# setup-vm.sh — Первичная настройка VM для Edium
# Запуск: ssh deploy@<VM_IP> 'bash -s' < setup-vm.sh
set -euo pipefail

echo "=== Обновление системы ==="
sudo apt-get update && sudo apt-get upgrade -y

echo "=== Установка Docker ==="
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo tee /etc/apt/keyrings/docker.asc > /dev/null
sudo chmod a+r /etc/apt/keyrings/docker.asc

echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

sudo systemctl enable docker
sudo systemctl start docker

echo "=== Настройка пользователя ==="
sudo usermod -aG docker "${USER}"

echo "=== Создание рабочей директории ==="
sudo mkdir -p /opt/edium
sudo chown "${USER}:${USER}" /opt/edium

echo "=== Настройка firewall (UFW) ==="
sudo apt-get install -y ufw
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP
sudo ufw allow 443/tcp   # HTTPS
# Prometheus node-exporter — только из внутренней сети
sudo ufw allow from 10.0.0.0/8 to any port 9100
sudo ufw --force enable

echo "=== Настройка логротации Docker ==="
sudo tee /etc/docker/daemon.json > /dev/null <<'EOF'
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  }
}
EOF
sudo systemctl restart docker

echo "=== Готово ==="
echo "VM настроена. Docker версия:"
docker --version
docker compose version
