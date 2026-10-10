terraform {
  required_version = ">= 1.7"
  required_providers {
    local = {
      source = "hashicorp/local"
    }
  }
}

resource "local_file" "config" {
  filename = "${path.module}/config.txt"
  content  = "v2"
}

resource "local_file" "keep" {
  filename = "${path.module}/keep.txt"
  content  = "keep"
}

resource "terraform_data" "app" {
  input = local_file.config.content
}
