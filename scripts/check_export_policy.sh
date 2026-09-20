#!/usr/bin/env bash
# 校验：corevia/ 是完整契约树；external 可 import 面存在；external 不与 forbidden 重叠。
# forbidden 允许在仓内存在（内部契约也住在 redkit-protos），只是仓外不得 import。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
POLICY_FILE="${REPO_ROOT}/policy/export-policy.yaml"
MODULE_DIR="${REPO_ROOT}/corevia"
WORK_FILE="${REPO_ROOT}/buf.work.yaml"

fail=0
report() { echo "redkit-protos policy violation: $1" >&2; fail=1; }

[[ -f "$POLICY_FILE" ]] || { report "missing $POLICY_FILE"; exit 1; }
[[ -d "$MODULE_DIR" ]] || { report "missing module dir $MODULE_DIR"; exit 1; }
[[ -f "$MODULE_DIR/buf.yaml" ]] || { report "missing $MODULE_DIR/buf.yaml"; exit 1; }
[[ -f "$WORK_FILE" ]] || { report "missing $WORK_FILE"; exit 1; }

if ! grep -q 'corevia' "$WORK_FILE"; then
  report "buf.work.yaml must include corevia module"
fi

# 唯一 tree：不应再出现 gamoji/pincp 等并列 proto 源目录。
for unexpected in gamoji pincp apps engagment engagement; do
  if [[ -d "${REPO_ROOT}/${unexpected}" ]]; then
    report "unexpected proto tree ${unexpected}/ — all protos live under corevia/ only"
  fi
done

yaml_list() {
  local file="$1" parent="$2" child="$3"
  awk -v parent="$parent" -v child="$child" '
    BEGIN { in_parent=0; in_child=0 }
    $0 ~ "^" parent ":" { in_parent=1; next }
    in_parent && /^[^[:space:]#]/ { in_parent=0; in_child=0 }
    in_parent && $0 ~ "^[[:space:]]+" child ":" { in_child=1; next }
    in_child && /^[[:space:]]+- / {
      line=$0
      sub(/^[[:space:]]+- /, "", line)
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", line)
      gsub(/^["\047]|["\047]$/, "", line)
      print line
      next
    }
    in_child && /^[[:space:]]*[^[:space:]#]/ && $0 !~ /^[[:space:]]+- / { in_child=0 }
  ' "$file"
}

external=()
forbidden=()
while IFS= read -r line; do [[ -n "$line" ]] && external+=("${line%/}"); done <<EOF
$(yaml_list "$POLICY_FILE" external packages)
EOF
while IFS= read -r line; do [[ -n "$line" ]] && forbidden+=("${line%/}"); done <<EOF
$(yaml_list "$POLICY_FILE" forbidden packages)
EOF

[[ ${#external[@]} -gt 0 ]] || report "external.packages empty"
[[ ${#forbidden[@]} -gt 0 ]] || report "forbidden.packages empty"

for req in admin core; do
  found=0
  for item in "${forbidden[@]}"; do
    [[ "${item%/}" == "$req" ]] && found=1
  done
  [[ $found -eq 1 ]] || report "forbidden.packages must include ${req}/ (external must not import it)"
done

for pkg in "${external[@]}"; do
  if [[ ! -d "${MODULE_DIR}/${pkg}" ]]; then
    report "external package not present under corevia/: ${pkg} (run scripts/sync_from_corevia.sh)"
  fi
  for f in "${forbidden[@]}"; do
    fr="${f%/}"
    if [[ "$pkg" == "$fr" || "$pkg" == "$fr/"* ]]; then
      report "external ${pkg} overlaps forbidden ${f}"
    fi
  done
done

# 完整契约树应包含 admin 与 core（住在本仓，但不给外部 import）。
for required in admin core common user wallet payment ops; do
  if [[ ! -d "${MODULE_DIR}/${required}" ]]; then
    report "full proto home missing tree: corevia/${required}/"
  fi
done

if [[ $fail -ne 0 ]]; then
  exit 1
fi
echo "redkit-protos policy ok: single-home=corevia external=${#external[@]} forbidden=${#forbidden[@]} full-trees=present"
