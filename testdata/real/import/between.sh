# Forget terraform_data.imp so the v2 import block has something to adopt.
set -euo pipefail
id=$(terraform output -raw imp_id)
terraform state rm terraform_data.imp
printf 'import_id = "%s"\n' "$id" > terraform.tfvars
