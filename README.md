# redkit-protos

**所有项目的 Protobuf 都只住在本仓。** 不存在「其他项目自己再放一份 proto」：Gamoji/Pincp/App 等仓外工程只 **消费** 本仓契约，平台与 corevia 的契约树也收敛在这里。

- 唯一契约 tree：`corevia/`（路径与历史 `backend/api` 对齐，import 仍是 `common/v1/...`）
- 仓外 **可以 import** 的 package：见 `policy/export-policy.yaml` 的 `external.packages`
- `admin/`、`core/` 等内部面**也在本仓**，供平台生成使用；**外部项目不得 import**（`forbidden.packages`）

决策：corevia 仓 [ADR 0066](../corevia/docs/adr/0066-protos-live-in-a-dedicated-redkit-protos-repo.md)、迁移地图 `.scratch/protos-monorepo/`。

## 目录

```text
redkit-protos/
├── buf.work.yaml
├── corevia/                 # 唯一 proto 树：admin core common gift ops payment support user wallet
│   ├── buf.yaml
│   └── SYNC.json
├── policy/export-policy.yaml
├── scripts/sync_from_corevia.sh
├── examples/consumer/       # 仓外项目 import 示例
└── Makefile
```

## 阶段

| 阶段 | 内容 |
| --- | --- |
| **Phase 2（当前写入路径）** | **只在本仓编辑** `.proto`；corevia 执行 `make proto-pull` 后生成。`backend/api` 契约树是工作副本 |
| 初始化/应急 | `make sync` 或 `scripts/sync_from_corevia.sh` 从 corevia 工作副本推回本仓 |
| 远端 | `git init` 已完成；推送到公司 Git 后按 `protos/vX.Y.Z` 打 tag |

corevia 侧命令：

```bash
cd corevia/backend
make proto-pull    # REDKIT_PROTOS_ROOT=/Users/feyman/code/redkit-protos
make proto-check
make api
```

## 仓外项目如何 import

分步指南：[docs/consumer-onboarding.md](./docs/consumer-onboarding.md)

1. submodule/vendor 本仓到你的工程（如 `third_party/redkit-protos`）
2. buf workspace **v1**，目录指向 `corevia/`
3. `import "common/v1/common.proto"`（仅 `external.packages`）

示例：`examples/consumer/`。`make example` 会验证跨 module import。

## 本仓命令

```bash
# 从 corevia 全量同步契约树
make sync COREVIA_ROOT=/path/to/corevia

# 策略 + 构建 + 消费方 import 验证
make example
```

## 约束

- 业务差异用 package/字段与 App Scope 表达，**不**再新建 gamoji/pincp 等平行 proto 仓或目录。
- 运行时 gRPC 仍在私网；引用 proto ≠ 公网暴露。
- Phase 1 改契约：先改 `corevia/backend/api`，再 `make sync`；Phase 2 后只改本仓。
