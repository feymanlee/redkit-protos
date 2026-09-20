# redkit-protos — Protobuf 唯一事实源 + Go SDK

所有项目的 Protobuf 与生成逻辑都在本仓。业务项目通过 **`go get github.com/feymanlee/redkit-protos`** 引用 Go SDK。

- Remote: `git@github.com:feymanlee/redkit-protos.git`
- Go module: `github.com/feymanlee/redkit-protos`
- 契约树: `corevia/`
- Go SDK: `gen/go/`（提交进仓库；当前 tag `v0.2.0`）

## Go 接入

```bash
go get github.com/feymanlee/redkit-protos@v0.2.0
```

```go
import (
    commonv1 "github.com/feymanlee/redkit-protos/gen/go/common/v1"
    walletv1 "github.com/feymanlee/redkit-protos/gen/go/wallet/v1"
)
```

禁止使用旧路径 `corevia/api/gen/go/...`。Go module 版本 tag 必须是 `vX.Y.Z`（不要用 `protos/vX.Y.Z` 作为 go get 版本）。

## 改契约 / 发版

```bash
# 1) 编辑 corevia/**/*.proto
make gen-go
git add -A && git commit -m "feat(contracts): ..."
git tag v0.3.0 && git push origin main v0.3.0
# 2) 业务仓
go get github.com/feymanlee/redkit-protos@v0.3.0
```

## 本仓命令

```bash
make gen-go     # 生成 gen/go
make check      # export policy + buf lint/build
make example    # 仓外 proto import 验证
```

Admin OpenAPI/TS：在本仓基于 `corevia/` 生成，再同步到 corevia 前端/资产目录（见 `buf.gen.admin.openapi.yaml`）。

## 非 Go / proto import

见 `docs/consumer-onboarding.md` 与 `examples/consumer/`。

## 约束

- 手写 proto 只改 `corevia/**.proto`。
- 仓外 proto import 仅限 `policy/export-policy.yaml` 的 `external.packages`。
- 内部 gRPC 仍在私网。
