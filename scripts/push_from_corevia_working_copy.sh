#!/usr/bin/env bash
# 将 redkit-protos/corevia 的契约树同步到本仓（初始化/应急）。
# 日常路径请在 redkit-protos 编辑，再在 corevia backend 执行 make proto-pull。
# 用法: ./scripts/push_from_corevia_working_copy.sh [corevia-repo-root]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
COREVIA_ROOT="${1:-${COREVIA_ROOT:-/Users/feyman/code/corevia}}"
bash "${SCRIPT_DIR}/sync_from_corevia.sh" "$COREVIA_ROOT"
