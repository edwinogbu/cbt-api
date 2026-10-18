#!/usr/bin/env bash
# Generates a fresh JWT_SECRET/JWT_REFRESH_SECRET/DB_PASSWORD for one
# School CBT Node. Run this once per node and paste the output into that
# node's .env.school-node - never reuse the same output across two nodes.
set -euo pipefail

random_secret() {
  openssl rand -hex 32
}

echo "# Paste these into this node's .env.school-node (do not reuse across nodes):"
echo "DB_PASSWORD=$(random_secret)"
echo "JWT_SECRET=$(random_secret)"
echo "JWT_REFRESH_SECRET=$(random_secret)"
