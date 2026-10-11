terraform {
  required_version = ">= 1.7"
}

# After review: db is no longer replaced, app changes differently, the queue
# is gone, legacy is now deleted, and the cache grows by one instance.
resource "terraform_data" "db" {
  triggers_replace = "a"
  input            = "data"
}

resource "terraform_data" "app" {
  input = "v3"
}

resource "terraform_data" "cache" {
  count = 3
  input = "c"
}
