# 仓外项目如何引用 redkit-protos 契约

适用：Gamoji BFF、Pincp BFF、App BFF / App Services 等**不自建 proto 源**的工程。

## 1. 获取契约

**推荐：git submodule**

```bash
# 在你的仓库根
git submodule add <REDKIT_PROTOS_GIT_URL> third_party/redkit-protos
git submodule update --init
```

**或：vendor 目录**

```bash
git clone <REDKIT_PROTOS_GIT_URL> third_party/redkit-protos
# CI 用同一路径拉取，并 pin 到 tag，例如 protos/v0.1.0
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
  common.v1.AppId app_id = 1;
  uint64 user_id = 2;
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
