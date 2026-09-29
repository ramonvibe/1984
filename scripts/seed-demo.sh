#!/usr/bin/env bash
set -euo pipefail

docker compose exec -T db psql -U trackline -d trackline < scripts/seed-demo.sql
echo 'Demo pronto: demo@1984.local / DemoSenha123!'
