# redkit-protos — Protobuf 唯一事实源 + Go SDK

所有项目的 Protobuf 与生成逻辑都在本仓。业务项目通过 **`go get github.com/feymanlee/redkit-protos`** 引用 Go SDK。

- Remote: `git@github.com:feymanlee/redkit-protos.git`
- Go module: `github.com/feymanlee/redkit-protos`
- 契约树: `corevia/`
- 生成代码: `gen/go/`（提交进仓库）

## Go 接入

```bash
go get github.com/feymanlee/redkit-protos@protos/vX.Y.Z
```

```go
import (
    commonv1 "github.com/feymanlee/redkit-protos/gen/go/common/v1"
    walletv1 "github.com/feymanlee/redkit-protos/gen/go/wallet/v1"
)
```

禁止使用旧路径 `corevia/api/gen/go/...`。

## 本仓命令

```bash
make gen-go
make check
```

发布：

```bash
make gen-go
git add gen/go go.mod go.sum buf.gen.go.yaml
git commit -m "chore(gen): regenerate Go SDK"
git tag protos/vX.Y.Z && git push origin main protos/vX.Y.Z
```

## 非 Go / proto import

见 `docs/consumer-onboarding.md` 与 `examples/consumer/`。

## 约束

- 手写 proto 只改 `corevia/**.proto`。
- 仓外 import 仅限 `policy/export-policy.yaml` 的 `external.packages`。
- 内部 gRPC 仍在私网。
