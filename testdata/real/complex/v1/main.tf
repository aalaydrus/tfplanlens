terraform {
  required_version = ">= 1.7"
}

variable "secret" {
  type      = string
  sensitive = true
  default   = "hunter2"
}

locals {
  user_data_lines = [for i in range(400) : "echo step-${i}"]
}

resource "terraform_data" "subnet" {
  for_each = toset(["a", "b", "c"])
  input    = "subnet-${each.key}"
}

resource "terraform_data" "worker" {
  count = 4
  input = "worker-${count.index}"
}

resource "terraform_data" "lb" {
  input = [for s in terraform_data.subnet : s.id]
}

resource "terraform_data" "db" {
  triggers_replace = "a"
  input            = "primary"
}

resource "terraform_data" "app" {
  input = terraform_data.db.id
}

resource "terraform_data" "creds" {
  input = var.secret
}

resource "terraform_data" "legacy" {
  input = "legacy-thing"
}

resource "terraform_data" "user_data" {
  input = base64encode(join("\n", concat(["#!/bin/bash"], local.user_data_lines)))
}

module "network" {
  source = "./modules/network"
  name   = "core"
}
