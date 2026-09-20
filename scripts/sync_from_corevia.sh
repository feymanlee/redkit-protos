#!/usr/bin/env bash
# 从 corevia 仓全量同步 backend/api 契约目录到 redkit-protos/corevia。
# Phase 1：生成事实源仍是 corevia/backend/api；本仓是**唯一**跨项目 proto 管理入口。
# 用法: ./scripts/sync_from_corevia.sh [path-to-corevia-repo]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
COREVIA_ROOT="${1:-${COREVIA_ROOT:-/Users/feyman/code/corevia}}"
API_DIR="${COREVIA_ROOT}/backend/api"
POLICY_FILE="${REPO_ROOT}/policy/export-policy.yaml"
DEST="${REPO_ROOT}/corevia"

if [[ ! -d "$API_DIR" ]]; then
  echo "corevia api dir not found: $API_DIR" >&2
  exit 1
fi
if [[ ! -f "$POLICY_FILE" ]]; then
  echo "missing $POLICY_FILE" >&2
  exit 1
fi

# 全量契约 top-level（与 backend/api 手写契约目录一致；不含 gen/、scripts/、生成模板）。
SYNC_TOPS=(admin common core gift ops payment support user wallet)

for top in "${SYNC_TOPS[@]}"; do
  src="${API_DIR}/${top}"
  if [[ ! -d "$src" ]]; then
    echo "missing source tree: $src" >&2
    exit 1
  fi
  rm -rf "${DEST:?}/${top}"
  cp -R "$src" "${DEST}/${top}"
  echo "synced ${top}/"
done

if [[ -f "${API_DIR}/buf.lock" ]]; then
  # corevia 的 buf.lock 可能是 v2；本仓 corevia/buf.yaml 为 v1 workspace 成员，需本地重解析 deps。
  cp "${API_DIR}/buf.lock" "${DEST}/buf.lock"
fi

# 确保 lock 与本仓 v1 buf.yaml 匹配。
(
  cd "$DEST"
  rm -f buf.lock
  buf dep update
)

# 写入同步元数据。
cat >"${DEST}/SYNC.json" <<EOF
{
  "source": "${API_DIR}",
  "source_commit": "$(git -C "$COREVIA_ROOT" rev-parse HEAD 2>/dev/null || echo unknown)",
  "synced_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "mode": "full-api-trees",
  "packages": $(printf '%s\n' "${SYNC_TOPS[@]}" | awk 'BEGIN{printf "["} {printf "%s\"%s\"", (NR>1?",":""), $0} END{printf "]"}')
}
EOF

echo "sync complete → ${DEST} (full contract trees)"
