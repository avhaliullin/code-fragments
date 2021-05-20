// Подключим terraform provider Яндекс.Облака
terraform {
  required_providers {
    yandex = {
      source = "yandex-cloud/yandex"
    }
  }
}

locals {
  // Описываем переменные, которые мы хотим передать в функцию в качестве конфигурации
  app-env-vars = {
    DATABASE               = var.database
    DATABASE_ENDPOINT      = var.database-endpoint
  }
}

// Описываем саму функцию
resource "yandex_function" "app" {
  // Точка входа в функцию - метод Handler
  entrypoint         = "main.Handler"
  memory             = 128
  name               = "test-app"
  runtime            = "golang114"
  // Здесь мы подсказываем terraform, что в функции есть изменения,
  // если поменялся hash архива с кодом
  user_hash          = data.archive_file.app-code.output_base64sha256
  content {
    zip_filename = data.archive_file.app-code.output_path
  }
  environment        = local.app-env-vars
  // Сервисный аккаунт позволяет получать IAM-токен изнутри функции
  service_account_id = yandex_iam_service_account.app-sa.id
  execution_timeout  = "3"
}

resource "yandex_iam_service_account" "app-sa" {
  name = "test-app"
}

// Разрешаем сервисному аккаунту функции обращаться в БД
resource "yandex_resourcemanager_folder_iam_binding" "app-sa-ydb-admin" {
  folder_id = var.folder-id
  members   = [
    "serviceAccount:${yandex_iam_service_account.app-sa.id}"
  ]
  role      = "ydb.admin"
}

data "archive_file" "app-code" {
  output_path = "${path.module}/dist/app-code.zip"
  type        = "zip"
  source_dir  = "${path.module}/app"
}

// При первом деплое будет создана новая функция - давайте попросим
// terraform печатать id функции после деплоя
output "function-id" {
  value = yandex_function.app.id
}

provider "yandex" {
  folder_id = var.folder-id
  token     = var.yc-token
}

variable "folder-id" {
  type = string
}

variable "yc-token" {
  type = string
}

variable "database" {
  type = string
}

variable "database-endpoint" {
  type = string
}
