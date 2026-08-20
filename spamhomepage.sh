#!/usr/bin/env bash

set -euo pipefail

app_url="${LINKO_URL:-http://localhost:${LINKO_PORT:-8899}}"
request_count="${REQUEST_COUNT:-3500}"

if [[ ! "$request_count" =~ ^[1-9][0-9]*$ ]]; then
  echo "REQUEST_COUNT must be a positive integer" >&2
  exit 2
fi

for ((i = 1; i <= request_count; i++)); do
  curl -fsS "$app_url" > /dev/null
  if (( i % 100 == 0 || i == request_count )); then
    echo "Completed $i requests"
  fi
done
