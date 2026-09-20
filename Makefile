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

check:
	@bash scripts/check_export_policy.sh
	@cd corevia && buf lint
	@cd corevia && buf build

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

# 将 Admin OpenAPI / TS / api_catalog 拷入 corevia 既有消费路径
publish-artifacts: openapi ts
	@set -e; \
	CV="$(COREVIA_ROOT)"; \
	test -d "$$CV/backend"; \
	mkdir -p "$$CV/backend/api/gen/openapi/admin"; \
	if [ -d dist/admin-openapi/gen/openapi/admin ]; then \
		cp -R dist/admin-openapi/gen/openapi/admin/. "$$CV/backend/api/gen/openapi/admin/"; \
	else \
		find dist/admin-openapi -type f \( -name '*.yaml' -o -name '*.yml' \) -exec cp {} "$$CV/backend/api/gen/openapi/admin/" \; 2>/dev/null || true; \
	fi; \
	if [ -f dist/admin-assets/api_catalog.yaml ]; then \
		cp dist/admin-assets/api_catalog.yaml "$$CV/backend/app/admin/cmd/server/assets/api_catalog.yaml"; \
	elif [ -f dist/admin-openapi/dist/admin-assets/api_catalog.yaml ]; then \
		cp dist/admin-openapi/dist/admin-assets/api_catalog.yaml "$$CV/backend/app/admin/cmd/server/assets/api_catalog.yaml"; \
	else \
		CAT="$$(find dist -name api_catalog.yaml 2>/dev/null | head -1)"; \
		if [ -n "$$CAT" ]; then cp "$$CAT" "$$CV/backend/app/admin/cmd/server/assets/api_catalog.yaml"; fi; \
	fi; \
	if [ -d dist/admin-ts ]; then \
		mkdir -p "$$CV/frontend/admin/apps/admin/src/api/generated"; \
		TS_SRC="$$(find dist/admin-ts -type d -name generated 2>/dev/null | head -1)"; \
		if [ -n "$$TS_SRC" ]; then \
			cp -R "$$TS_SRC/." "$$CV/frontend/admin/apps/admin/src/api/generated/"; \
		else \
			cp -R dist/admin-ts/. "$$CV/frontend/admin/apps/admin/src/api/generated/" || true; \
		fi; \
	fi; \
	echo "published openapi/ts artifacts into $$CV"

clean:
	@rm -rf examples/consumer/their/.cache dist
