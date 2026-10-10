terraform {
  required_version = ">= 1.7"
}

resource "terraform_data" "web" {
  input = "web"
}

resource "terraform_data" "db" {
  input = "db"
}

resource "terraform_data" "shard" {
  count = 2
  input = "shard-${count.index}"
}
