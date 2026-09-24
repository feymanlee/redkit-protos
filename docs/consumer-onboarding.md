# 仓外项目如何引用 redkit-protos

适用：C 端 BFF、App BFF / App Services 等**不自建 proto 源**的工程。

## Go 项目（推荐）

```bash
go get github.com/feymanlee/redkit-protos@vX.Y.Z
```

```go
import (
    commonv1 "github.com/feymanlee/redkit-protos/gen/go/common/v1"
)
```

## 其他语言 / 需要 .proto 源

远端：`git@github.com:feymanlee/redkit-protos.git`（建议 pin `vX.Y.Z`）

```bash
git submodule add git@github.com:feymanlee/redkit-protos.git third_party/redkit-protos
cd third_party/redkit-protos && git checkout v0.2.0
```

契约树：`third_party/redkit-protos/corevia/`。

buf workspace **v1**：

```yaml
version: v1
directories:
  - third_party/redkit-protos/corevia
  - proto
```

```proto
import "common/v1/common.proto";
```

仅允许 `policy/export-policy.yaml` 的 `external.packages`（禁止 `admin/`、`core/` 等）。

## 运行时

内部 gRPC 仅私网可达；引用 proto/SDK ≠ 公网暴露。


## 1. 获取契约

远端：`git@github.com:feymanlee/redkit-protos.git`（建议 pin tag，当前 `protos/v0.1.0`）

**推荐：git submodule**

```bash
# 在你的仓库根
git submodule add git@github.com:feymanlee/redkit-protos.git third_party/redkit-protos
cd third_party/redkit-protos && git checkout protos/v0.1.0 && cd -
git submodule update --init
```

**或：vendor 目录**

```bash
git clone git@github.com:feymanlee/redkit-protos.git third_party/redkit-protos
cd third_party/redkit-protos && git checkout protos/v0.1.0
```

契约树：`third_party/redkit-protos/corevia/`。

## 2. Buf workspace（v1）

```yaml
# your-repo/buf.work.yaml
version: v1
directories:
  - third_party/redkit-protos/corevia
  - proto          # 你自己的 proto module
```

```yaml
# your-repo/proto/buf.yaml
version: v1
lint:
  use:
    - DEFAULT
```

## 3. 在自己的 proto 里 import

```proto
syntax = "proto3";
package your.v1;

import "common/v1/common.proto";

message YourMessage {
  uint64 user_id = 1;
}
```

路径相对 **corevia module 根**，与历史 `backend/api` 一致。

## 4. 允许 import 的范围

以 `redkit-protos/policy/export-policy.yaml` 为准：

| 可 import（external） | 不可 import（forbidden） |
| --- | --- |
| `common/**` | `admin/**` |
| `user/types|consumer|internal/**` | `core/**` |
| `wallet/v1`、`payment/v1` | `user/administration/**` |
| `ops/v1`、`ops/reward/v1` | 其他 policy 列出的内部面 |

## 5. 验证

```bash
cd proto && buf build
# 或 workspace 根：
buf build proto
```

完整可运行示例：`redkit-protos/examples/consumer/`。

## 6. 运行时提醒

内部 gRPC 仍只在私网可达；import proto 不代表服务已对公网开放。metadata（App/User/Session）由 BFF 按已验证身份重建。
