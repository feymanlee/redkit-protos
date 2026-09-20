# Consumer 示例（仓外项目 import corevia proto）

本目录演示：在**不自建 proto 源**的前提下，如何用 workspace 引用 `redkit-protos/corevia`。

## 布局（submodule 场景）

```text
your-repo/
├── buf.work.yaml
├── proto/
│   ├── buf.yaml
│   └── your/v1/demo.proto
└── third_party/redkit-protos/    # git@github.com:feymanlee/redkit-protos.git
    └── corevia/                  # 契约 module
```

```yaml
# your-repo/buf.work.yaml
version: v1
directories:
  - third_party/redkit-protos/corevia
  - proto
```

本示例在 redkit-protos 仓内已用 workspace 连接 `corevia/` 与 `examples/consumer/their`，等价于 submodule 路径写法。

## import

```proto
import "common/v1/common.proto";
```

仅允许 `policy/export-policy.yaml` 的 `external.packages`。

## 本地验证（在 redkit-protos 根）

```bash
make example
```

## 你仓库里的验证

```bash
cd proto && buf build
```

完整步骤见 [../docs/consumer-onboarding.md](../docs/consumer-onboarding.md)。
