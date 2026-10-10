terraform {
  required_providers {
    local = {
      source = "hashicorp/local"
    }
  }
}

resource "local_file" "inner" {
  filename = "${path.root}/inner.txt"
  content  = "inner-v1"
}
