# redkit-protos Makefile

.PHONY: help check lint build example gen-go openapi ts grpc-docs clean

GO_MODULE := github.com/feymanlee/redkit-protos

help:
	@echo "make gen-go   - generate gen/go SDK"
	@echo "make check    - policy + buf lint/build"
	@echo "make openapi  - admin OpenAPI into dist/admin-openapi"
	@echo "make ts       - (from corevia) admin TS via buf"

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
	@echo "openapi under dist/admin-openapi — copy to corevia backend/app/admin/... as needed"

grpc-docs:
	@mkdir -p dist/grpc-docs
	@cd corevia && buf generate --template ../buf.gen.grpc-doc.yaml --output ../dist/grpc-docs

clean:
	@rm -rf examples/consumer/their/.cache
