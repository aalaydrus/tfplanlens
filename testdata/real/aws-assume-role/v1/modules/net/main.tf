terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
  }
}

variable "suffix" {
  type = string
}

# Runs on whatever provider the caller passes in `providers = { aws = ... }`.
resource "aws_sqs_queue" "inner" {
  name = "net-inner-${var.suffix}"
}
