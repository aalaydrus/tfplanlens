terraform {
  required_version = ">= 1.7"
}

resource "terraform_data" "trigger" {
  input = "t2"
}

resource "terraform_data" "follower" {
  input = "follower"
  lifecycle {
    replace_triggered_by = [terraform_data.trigger]
  }
}

resource "terraform_data" "tainted" {
  input = "will-be-tainted"
}

resource "terraform_data" "requested" {
  input = "replace-me"
}

resource "terraform_data" "single" {
  for_each = toset(["a"])
  input    = "was-single"
}

resource "terraform_data" "counted" {
  count = 1
  input = "was-single"
}

module "mods" {
  for_each = toset(["a"])
  source   = "./modules/gone"
}
