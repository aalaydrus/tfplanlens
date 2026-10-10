terraform {
  required_version = ">= 1.7"
}

resource "terraform_data" "imp" {
  input = "managed"
}

resource "terraform_data" "other" {
  input = "other"
}

output "imp_id" {
  value = terraform_data.imp.id
}
