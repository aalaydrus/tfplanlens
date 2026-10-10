terraform {
  required_version = ">= 1.7"
}

variable "import_id" {
  type = string
}

resource "terraform_data" "imp" {
  input = "managed"
}

import {
  to = terraform_data.imp
  id = var.import_id
}

resource "terraform_data" "other" {
  input = "other-changed"
}
