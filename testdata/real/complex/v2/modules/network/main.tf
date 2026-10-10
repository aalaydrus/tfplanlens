variable "name" {
  type = string
}

resource "terraform_data" "vpc" {
  input = "vpc-${var.name}"
}

resource "terraform_data" "route_table" {
  input = terraform_data.vpc.id
}
