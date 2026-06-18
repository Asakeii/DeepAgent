#!/usr/bin/env bash
set -euo pipefail

RUN_NAME="deep-agent"

mkdir -p output/bin output/conf
cp -R conf/* output/conf/ 2>/dev/null || true
go build -o "output/bin/${RUN_NAME}" .
"./output/bin/${RUN_NAME}" "$@"

