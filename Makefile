# redkit-protos Makefile — 跨项目 Protobuf 管理入口

.PHONY: help sync check lint build example clean

COREVIA_ROOT ?= /Users/feyman/code/corevia

help:
	@echo "make sync     - 从 corevia 同步 external 面 proto"
	@echo "make check    - 校验 export policy 与目录"
	@echo "make lint     - buf lint corevia module"
	@echo "make build    - buf build corevia + 消费方示例"
	@echo "make example  - 验证 examples/consumer 可 import corevia proto"
	@echo "COREVIA_ROOT=$(COREVIA_ROOT)"

sync:
	@bash scripts/sync_from_corevia.sh "$(COREVIA_ROOT)"

check:
	@bash scripts/check_export_policy.sh

lint: check
	@cd corevia && buf lint
	@cd examples/consumer/their && buf lint || true

build: check
	@cd corevia && buf build
	@cd examples/consumer/their && buf build
	@echo "modules build ok"

example: build
	@cd examples/consumer/their && buf build -o /dev/null
	@echo "consumer import ok"

clean:
	@rm -rf examples/consumer/their/.cache
