terraform {
  required_version = ">= 1.7"
}

variable "secret" {
  type      = string
  sensitive = true
  default   = "hunter3"
}

locals {
  user_data_lines = [
    for i in range(400) :
    (i == 120 ? "echo CHANGED-120" : i == 121 ? "echo CHANGED-121" : "echo step-${i}")
  ]
}

resource "terraform_data" "subnet" {
  for_each = toset(["a", "c", "d"])
  input    = "subnet-${each.key}"
}

resource "terraform_data" "worker" {
  count = 2
  input = "worker-${count.index}"
}

resource "terraform_data" "lb" {
  input = [for s in terraform_data.subnet : s.id]
}

resource "terraform_data" "db" {
  triggers_replace = "b"
  input            = "primary"
}

resource "terraform_data" "app" {
  input = terraform_data.db.id
}

resource "terraform_data" "creds" {
  input = var.secret
}

removed {
  from = terraform_data.legacy
  lifecycle {
    destroy = false
  }
}

resource "terraform_data" "user_data" {
  input = base64encode(join("\n", concat(["#!/bin/bash"], local.user_data_lines)))
}

module "network_core" {
  source = "./modules/network"
  name   = "core"
}

moved {
  from = module.network
  to   = module.network_core
}
