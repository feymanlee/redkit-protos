#!/usr/bin/env bash
# 生成 Admin TypeScript 客户端到 dist/admin-ts。
# 依赖 corevia 前端仓中的 protoc-gen-typescript-http-string-uint64.mjs。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
COREVIA_ROOT="${COREVIA_ROOT:-/Users/feyman/code/corevia}"
PLUGIN="${COREVIA_ROOT}/frontend/admin/scripts/protoc-gen-typescript-http-string-uint64.mjs"

if [[ ! -f "$PLUGIN" ]]; then
  echo "TS plugin not found: $PLUGIN" >&2
  exit 1
fi

TMP_TPL="$(mktemp "${TMPDIR:-/tmp}/buf.gen.admin.ts.XXXXXX.yaml")"
cleanup() { rm -f "$TMP_TPL"; }
trap cleanup EXIT

sed "s|PLUGIN_PATH_PLACEHOLDER|${PLUGIN}|" "${ROOT}/buf.gen.admin.ts.yaml" >"$TMP_TPL"

mkdir -p "${ROOT}/dist/admin-ts"
cd "${ROOT}/corevia"
buf generate --template "$TMP_TPL" --output "${ROOT}/dist/admin-ts"
echo "ts → ${ROOT}/dist/admin-ts"
