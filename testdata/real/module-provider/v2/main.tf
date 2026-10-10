terraform {
  required_version = ">= 1.7"
  required_providers {
    local = {
      source = "hashicorp/local"
    }
  }
}

provider "local" {}

provider "local" {
  alias = "alt"
}

# The module's resources run on the aliased provider configuration.
module "net" {
  source = "./modules/net"
  providers = {
    local = local.alt
  }
}

resource "local_file" "root" {
  filename = "${path.module}/root.txt"
  content  = "root-v2"
}

resource "local_file" "alt" {
  provider = local.alt
  filename = "${path.module}/alt.txt"
  content  = "alt-v2"
}
