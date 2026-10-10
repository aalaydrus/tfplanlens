terraform {
  required_version = ">= 1.7"
}

resource "terraform_data" "web" {
  input = "v1"
}

resource "terraform_data" "db" {
  triggers_replace = "a"
  input            = "data"
}

resource "terraform_data" "old" {
  input = "to-be-removed"
}
