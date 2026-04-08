variable "cloud_id" {
  description = "ID облака Yandex Cloud"
  type        = string
}

variable "folder_id" {
  description = "ID каталога Yandex Cloud"
  type        = string
}

variable "sa_key_file" {
  description = "Путь к файлу ключа сервисного аккаунта"
  type        = string
  default     = "key.json"
}

variable "ssh_public_key" {
  description = "SSH публичный ключ для доступа к VM"
  type        = string
}

variable "ssh_allowed_cidrs" {
  description = "CIDR-блоки, откуда разрешён SSH"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

# --- Doorman ---

variable "doorman_pg_user" {
  description = "PostgreSQL пользователь для Doorman"
  type        = string
  default     = "doorman"
}

variable "doorman_pg_password_test" {
  description = "Пароль PostgreSQL для Doorman (test)"
  type        = string
  sensitive   = true
}

variable "doorman_pg_password_prod" {
  description = "Пароль PostgreSQL для Doorman (prod)"
  type        = string
  sensitive   = true
}

# --- Herald ---

variable "herald_pg_password_test" {
  description = "Пароль PostgreSQL для Herald (test)"
  type        = string
  sensitive   = true
}

variable "herald_pg_password_prod" {
  description = "Пароль PostgreSQL для Herald (prod)"
  type        = string
  sensitive   = true
}

# --- Charon ---

variable "charon_pg_password_test" {
  description = "Пароль PostgreSQL для Charon (test)"
  type        = string
  sensitive   = true
}

variable "charon_pg_password_prod" {
  description = "Пароль PostgreSQL для Charon (prod)"
  type        = string
  sensitive   = true
}

# --- Caesar ---

variable "caesar_pg_password_test" {
  description = "Пароль PostgreSQL для Caesar (test)"
  type        = string
  sensitive   = true
}

variable "caesar_pg_password_prod" {
  description = "Пароль PostgreSQL для Caesar (prod)"
  type        = string
  sensitive   = true
}

variable "redis_password" {
  description = "Пароль Redis"
  type        = string
  sensitive   = true
}

variable "domain" {
  description = "Доменное имя"
  type        = string
  default     = "edium.online"
}

# --- Sphinx (ML-аккаунт) ---

variable "sphinx_vm_ip" {
  description = "Внешний IP GPU VM из ML-аккаунта (для DNS и доступа к PG)"
  type        = string
  default     = ""
}

variable "sphinx_pg_password" {
  description = "Пароль PostgreSQL для Sphinx"
  type        = string
  sensitive   = true
}
