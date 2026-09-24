# redkit-protos Makefile — Protobuf 唯一事实源 + Go SDK

.PHONY: help check lint build example gen-go openapi ts grpc-docs publish-artifacts clean

GO_MODULE := github.com/feymanlee/redkit-protos
COREVIA_ROOT ?= /Users/feyman/code/corevia

help:
	@echo "make gen-go    - generate gen/go SDK ($(GO_MODULE))"
	@echo "make openapi   - admin OpenAPI under dist/admin-openapi"
	@echo "make ts        - admin TypeScript under dist/admin-ts"
	@echo "make grpc-docs - markdown docs under dist/grpc-docs"
	@echo "make publish-artifacts - copy OpenAPI/TS/api_catalog into corevia"
	@echo "COREVIA_ROOT=$(COREVIA_ROOT)"

check: check-single-app
	@bash scripts/check_export_policy.sh
	@cd corevia && buf lint
	@cd corevia && buf build

# 正向守卫：契约中不得再出现平台 App 身份（ADR 0075）
check-single-app:
	@tmp="$$(mktemp -d)"; trap 'rm -rf "$$tmp"' EXIT; \
	cd corevia && buf build --as-file-descriptor-set -o "$$tmp/descriptor-set.bin"; \
	cd "$(CURDIR)" && go run ./scripts/check_single_app_policy -descriptor-set "$$tmp/descriptor-set.bin"

build: check
	@cd examples/consumer/their && buf build

example: build
	@echo "consumer import ok"

gen-go:
	@rm -rf gen/go
	@cd corevia && buf generate --template ../buf.gen.go.yaml --output ..
	@test -d gen/go
	@echo "generated $(GO_MODULE)/gen/go"
	@if command -v go >/dev/null 2>&1; then go mod tidy; fi

openapi:
	@mkdir -p dist/admin-openapi
	@cd corevia && buf generate --template ../buf.gen.admin.openapi.yaml --output ../dist/admin-openapi
	@echo "openapi → dist/admin-openapi"

ts:
	@COREVIA_ROOT="$(COREVIA_ROOT)" bash scripts/generate-admin-ts.sh

grpc-docs:
	@mkdir -p dist/grpc-docs
	@cd corevia && buf generate --template ../buf.gen.grpc-doc.yaml --output ../dist/grpc-docs
	@echo "grpc-docs → dist/grpc-docs"

# 权限目录 api_catalog 拷入 corevia（OpenAPI yaml 留在本仓 dist，不进 corevia）
publish-artifacts: openapi ts
	@set -e; \
	CV="$(COREVIA_ROOT)"; \
	test -d "$$CV/backend"; \
	CAT="$$(find dist -name api_catalog.yaml 2>/dev/null | head -1)"; \
	if [ -n "$$CAT" ]; then \
		cp "$$CAT" "$$CV/backend/app/admin/cmd/server/assets/api_catalog.yaml"; \
		echo "copied api_catalog.yaml"; \
	else \
		echo "warning: api_catalog.yaml not found under dist/" >&2; \
	fi; \
	if [ -d dist/admin-ts ]; then \
		mkdir -p "$$CV/frontend/admin/apps/admin/src/api/generated"; \
		TS_SRC="$$(find dist/admin-ts -type d -name generated 2>/dev/null | head -1)"; \
		if [ -n "$$TS_SRC" ]; then \
			cp -R "$$TS_SRC/." "$$CV/frontend/admin/apps/admin/src/api/generated/"; \
			echo "copied admin TS client"; \
		else \
			cp -R dist/admin-ts/. "$$CV/frontend/admin/apps/admin/src/api/generated/" || true; \
		fi; \
	fi

clean:
	@rm -rf examples/consumer/their/.cache dist
