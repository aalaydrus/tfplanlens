terraform {
  required_version = ">= 1.7"
}

resource "terraform_data" "trigger" {
  input = "t1"
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

# Becomes for_each in v2: no implicit move exists, so Terraform deletes the
# un-keyed instance (delete_because_wrong_repetition).
resource "terraform_data" "single" {
  input = "was-single"
}

# A plain resource that becomes count = 1 in v2: Terraform treats that as an
# implicit move (previous_address), not a delete.
resource "terraform_data" "counted" {
  input = "was-single"
}

# Module call removed entirely in v2.
module "gone" {
  source = "./modules/gone"
}

# Module for_each key removed in v2.
module "mods" {
  for_each = toset(["a", "b"])
  source   = "./modules/gone"
}
