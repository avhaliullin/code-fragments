locals {
  folder-id                = var.folder-id
  maintenance-interval-utc = "03:00-05:00"
  restart-label            = "auto-restart"
  hours-to-restart         = 23
  log-level                = "info"
  operations-limit         = 10
}

resource "yandex_function_trigger" "vm-watchdog" {
  name = "vm-watchdog"
  function {
    id                 = yandex_function.vm-watchdog.id
    service_account_id = yandex_iam_service_account.watchdog-trigger.id
  }
  timer {
    cron_expression = "* * ? * * *"
  }
  depends_on = [yandex_function_iam_binding.watchdog-trigger-fn-invoker]
}

resource "yandex_function" "vm-watchdog" {
  entrypoint = "watchdog.Handler"
  memory     = 128
  name       = "vm-watchdog"
  runtime    = "golang119"
  user_hash  = data.archive_file.vm-watchdog-src.output_base64sha256
  content {
    zip_filename = data.archive_file.vm-watchdog-src.output_path
  }
  environment = {
    MAINTENANCE_INTERVAL : local.maintenance-interval-utc
    RESTART_LABEL : local.restart-label
    OPERATIONS_LIMIT : local.operations-limit
    FOLDER_ID : local.folder-id
    HOURS_TO_RESTART : local.hours-to-restart
    LOG_LEVEL : local.log-level
  }
  execution_timeout = "50"
  depends_on        = [yandex_resourcemanager_folder_iam_member.watchdog-fn-compute-admin]
}

data "archive_file" "vm-watchdog-src" {
  output_path = "${path.module}/dist/watchdog-src.zip"
  type        = "zip"
  source_dir  = "${path.module}/src"
}

resource "yandex_iam_service_account" "watchdog-fn" {
  name = "vm-wathcdog-fn"
}

resource "yandex_iam_service_account" "watchdog-trigger" {
  name = "vm-watchdog-trigger"
}

resource "yandex_resourcemanager_folder_iam_member" "watchdog-fn-compute-admin" {
  folder_id = local.folder-id
  role      = "compute.admin"
  member    = "serviceAccount:${yandex_iam_service_account.watchdog-fn.id}"
}

resource "yandex_function_iam_binding" "watchdog-trigger-fn-invoker" {
  function_id = yandex_function.vm-watchdog.id
  members     = ["serviceAccount:${yandex_iam_service_account.watchdog-trigger.id}"]
  role        = "serverless.functions.invoker"
}

# configuration
terraform {
  required_providers {
    yandex = {
      source = "yandex-cloud/yandex"
    }
  }
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

