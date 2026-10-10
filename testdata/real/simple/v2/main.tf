terraform {
  required_version = ">= 1.7"
}

resource "terraform_data" "web" {
  input = "v2"
}

resource "terraform_data" "db" {
  triggers_replace = "b"
  input            = "data"
}

resource "terraform_data" "cache" {
  input = "new"
}
