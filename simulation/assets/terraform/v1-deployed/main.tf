# v1: what is deployed in production today.
terraform {
  required_version = ">= 1.7"
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
  }
}

provider "aws" {
  region              = "us-east-1"
  allowed_account_ids = ["123456789012"] # the workload account

  default_tags {
    tags = {
      app        = "orders"
      managed_by = "terraform"
    }
  }
}

variable "db_password" {
  type      = string
  sensitive = true
}

locals {
  # queue name => visibility timeout (seconds)
  queues = {
    standard = 30
    priority = 30
    bulk     = 120
  }
}

# --- Storage ---------------------------------------------------------------

resource "aws_s3_bucket" "receipts" {
  bucket = "orders-receipts-prod"
}

resource "aws_s3_bucket_versioning" "receipts" {
  bucket = aws_s3_bucket.receipts.id
  versioning_configuration {
    status = "Enabled"
  }
}


resource "aws_dynamodb_table" "orders" {
  name         = "orders"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "order_id"

  attribute {
    name = "order_id"
    type = "S"
  }
}

# --- Messaging -------------------------------------------------------------

resource "aws_sqs_queue" "dlq" {
  name = "orders-dlq"
}

resource "aws_sqs_queue" "orders" {
  for_each = local.queues

  name                       = "orders-${each.key}"
  visibility_timeout_seconds = each.value
  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.dlq.arn
    maxReceiveCount     = 5
  })
}

resource "aws_sns_topic" "events" {
  name = "order-events"
}

resource "aws_sns_topic_subscription" "standard" {
  topic_arn = aws_sns_topic.events.arn
  protocol  = "sqs"
  endpoint  = aws_sqs_queue.orders["standard"].arn
}

# --- Config and access -----------------------------------------------------

resource "aws_ssm_parameter" "flags" {
  name  = "/orders/feature-flags"
  type  = "String"
  value = jsonencode({ express_checkout = false })
}

resource "aws_secretsmanager_secret" "db" {
  name                    = "orders/db-credentials"
  description             = "Database credentials for the orders service"
  recovery_window_in_days = 7
}

# Every change to secret_string creates a new version of the secret (the old
# one is kept as AWSPREVIOUS). Terraform shows that as an in-place update, even
# though a consumer reading the secret sees a new value.
resource "aws_secretsmanager_secret_version" "db" {
  secret_id = aws_secretsmanager_secret.db.id
  secret_string = jsonencode({
    username = "orders_app"
    password = var.db_password
  })
}

resource "aws_iam_role" "worker" {
  name = "orders-worker"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "lambda.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy" "worker" {
  name = "orders-worker"
  role = aws_iam_role.worker.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["sqs:ReceiveMessage", "sqs:DeleteMessage"]
        Resource = [for q in aws_sqs_queue.orders : q.arn]
      },
      {
        Effect   = "Allow"
        Action   = ["secretsmanager:GetSecretValue"]
        Resource = aws_secretsmanager_secret.db.arn
      },
      {
        Effect   = "Allow"
        Action   = ["dynamodb:PutItem", "dynamodb:GetItem"]
        Resource = aws_dynamodb_table.orders.arn
      },
    ]
  })
}
