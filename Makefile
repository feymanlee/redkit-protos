# redkit-protos Makefile

.PHONY: help check lint build example gen-go clean

GO_MODULE := github.com/feymanlee/redkit-protos

help:
	@echo "make gen-go   - generate gen/go from corevia/"
	@echo "make check    - export policy + buf lint/build"
	@echo "make example  - consumer import verification"

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
