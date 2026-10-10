terraform {
  required_version = ">= 1.7"
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
  }
}

variable "staging_role_arn" {
  type    = string
  default = "arn:aws:iam::222222222222:role/staging-deployer"
}

# Default provider: runs as the runner's own identity (no assume_role).
provider "aws" {}

# Literal role ARN.
provider "aws" {
  alias = "prod"
  assume_role {
    role_arn = "arn:aws:iam::111111111111:role/prod-deployer"
  }
}

# Role ARN from a variable.
provider "aws" {
  alias = "staging"
  assume_role {
    role_arn = var.staging_role_arn
  }
}

# No assume_role: also the runner's identity.
provider "aws" {
  alias = "sandbox"
}

# No assume_role, but pinned to one account.
provider "aws" {
  alias               = "audit"
  allowed_account_ids = ["999999999999"]
}

data "aws_caller_identity" "current" {}

data "aws_caller_identity" "prod" {
  provider = aws.prod
}

resource "aws_sqs_queue" "runner" {
  name = "runner-queue"
  tags = {
    env = "v2"
  }
}

resource "aws_sns_topic" "prod" {
  provider     = aws.prod
  name         = "prod-topic"
  display_name = "prod v2"
}

resource "aws_sqs_queue" "staging" {
  provider = aws.staging
  name     = "staging-queue"
  tags = {
    env = "v2"
  }
}

resource "aws_ssm_parameter" "sandbox" {
  provider = aws.sandbox
  name     = "/sandbox/setting"
  type     = "String"
  value    = "v2"
}

resource "aws_sqs_queue" "audit" {
  provider                   = aws.audit
  name                       = "audit-queue"
  visibility_timeout_seconds = 60
}

# A module whose resources run on the prod provider.
module "net" {
  source = "./modules/net"
  providers = {
    aws = aws.prod
  }
  suffix = "v2"   # renames the queue: replaced, through the aliased provider
}

# A subscription created by the runner (default provider) to a topic that
# lives in the prod account: a real cross-account reference.
resource "aws_sns_topic_subscription" "cross" {
  topic_arn = aws_sns_topic.prod.arn
  protocol  = "sqs"
  endpoint  = aws_sqs_queue.runner.arn
}
