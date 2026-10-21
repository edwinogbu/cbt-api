#!/usr/bin/env bash
# Generates a fresh JWT_SECRET/JWT_REFRESH_SECRET/DB_PASSWORD/
# ENCRYPTION_MASTER_KEY for one School CBT Node. Run this once per node
# and paste the output into that node's .env.school-node - never reuse
# the same output across two nodes.
set -euo pipefail

random_secret() {
  openssl rand -hex 32
}

# ENCRYPTION_MASTER_KEY must be EXACTLY 32 characters (AES-256 key, used
# as raw bytes, not hex/base64-decoded - see pkg/utils/encryption.go).
# 24 random bytes base64-encodes to exactly 32 characters with no
# padding (24 is divisible by 3), unlike random_secret()'s hex output
# above, which is sized for JWT and would be the wrong length here.
random_encryption_key() {
  openssl rand -base64 24
}

echo "# Paste these into this node's .env.school-node (do not reuse across nodes):"
echo "DB_PASSWORD=$(random_secret)"
echo "JWT_SECRET=$(random_secret)"
echo "JWT_REFRESH_SECRET=$(random_secret)"
echo "ENCRYPTION_MASTER_KEY=$(random_encryption_key)"
