terraform {
  required_version = ">= 1.7"
}

resource "terraform_data" "db" {
  triggers_replace = "a"
  input            = "data"
}

resource "terraform_data" "app" {
  input = "v1"
}

resource "terraform_data" "legacy" {
  input = "old"
}

resource "terraform_data" "cache" {
  count = 2
  input = "c"
}
