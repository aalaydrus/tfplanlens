# Out-of-band changes: delete one managed file, tamper with another.
set -euo pipefail
rm config.txt
echo "tampered outside terraform" > keep.txt
