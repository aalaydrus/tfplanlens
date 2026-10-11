terraform {
  required_version = ">= 1.7"
}

# First plan under review: replaces db, updates app, adds a queue.
resource "terraform_data" "db" {
  triggers_replace = "b"
  input            = "data"
}

resource "terraform_data" "app" {
  input = "v2"
}

resource "terraform_data" "legacy" {
  input = "old"
}

resource "terraform_data" "cache" {
  count = 2
  input = "c"
}

resource "terraform_data" "queue" {
  input = "q"
}
