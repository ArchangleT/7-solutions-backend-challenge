BIN_DIR   := $(shell go env GOPATH)/bin
OAPI      := $(BIN_DIR)/oapi-codegen
MOCKGEN   := $(BIN_DIR)/mockgen
PROTOC_GEN_GO := $(BIN_DIR)/protoc-gen-go
PROTOC_GEN_GO_GRPC := $(BIN_DIR)/protoc-gen-go-grpc
COVERAGE  := coverage.out
COVERAGE_HTML := coverage.html

.PHONY: build run test test-integration tools codegen proto mock docsgen clean help

## build: compile the API binary into bin/
build:
	go build -o bin/api_user_mgr ./cmd/api

## run: start the API server (add APP_ENV=prod for production config)
run:
	go run ./cmd/api

## test: run the full test suite with coverage profile (generated code excluded)
test:
	go test ./... -coverprofile=$(COVERAGE)
	grep -v -E '/internal/mocks/|server_openapi\.gen\.go|\.pb\.go' $(COVERAGE) > $(COVERAGE).filtered
	go tool cover -func=$(COVERAGE).filtered
	rm -f $(COVERAGE).filtered

## test-integration: run the opt-in live MongoDB connectivity test
test-integration:
	TEST_MONGO_URI="$${TEST_MONGO_URI:?set TEST_MONGO_URI to a live MongoDB URI}" \
		go test ./internal/infrastructure/db/ -run TestConnect_Live -count=1 -v

## tools: install oapi-codegen, mockgen and protoc plugins into $(go env GOPATH)/bin
tools: $(OAPI) $(MOCKGEN) $(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC)

$(OAPI):
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

$(MOCKGEN):
	go install go.uber.org/mock/mockgen@latest

$(PROTOC_GEN_GO):
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

$(PROTOC_GEN_GO_GRPC):
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

## codegen: regenerate the OpenAPI server interfaces and models
codegen: $(OAPI)
	$(OAPI) --config api/codegen.yaml api/openapi.yaml
	gofmt -w internal/adapter/inbound/http/server_openapi.gen.go

## proto: regenerate the gRPC stubs from api/proto/user_service.proto
## (requires protoc: brew install protobuf)
proto: $(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC)
	mkdir -p internal/adapter/inbound/grpc/userv1
	protoc \
		--proto_path=api/proto \
		--go_out=internal/adapter/inbound/grpc/userv1 \
		--go_opt=paths=source_relative \
		--go-grpc_out=internal/adapter/inbound/grpc/userv1 \
		--go-grpc_opt=paths=source_relative \
		user_service.proto
	gofmt -w internal/adapter/inbound/grpc/userv1

## mock: regenerate gomock mocks for all application ports
mock: $(MOCKGEN)
	$(MOCKGEN) -source internal/domain/statusdb/repository.go \
		-destination internal/mocks/statusdb/repository.go \
		-package statusdbmocks
	$(MOCKGEN) -source internal/domain/user/repository.go \
		-destination internal/mocks/userrepo/repository.go \
		-package userreposmocks
	$(MOCKGEN) -source internal/application/ports/hasher.go \
		-destination internal/mocks/ports/hasher.go \
		-package portsmocks
	$(MOCKGEN) -source internal/application/ports/token.go \
		-destination internal/mocks/ports/token.go \
		-package portsmocks
	gofmt -w internal/mocks

## docsgen: bundle the OpenAPI spec into docs/ as JSON and YAML
docsgen:
	mkdir -p docs
	npx --yes @redocly/cli bundle api/openapi.yaml -o docs/swagger.json --ext json
	npx --yes @redocly/cli bundle api/openapi.yaml -o docs/swagger.yaml --ext yaml

## clean: remove generated artifacts
clean:
	rm -rf $(COVERAGE) $(COVERAGE_HTML) docs

## help: list available targets
help:
	@echo "Targets:"
	@awk '/^[a-zA-Z0-9_-]+:/ {help=0} /^## / {help=1} help {sub(/^## /, ""); print $$0}' $(MAKEFILE_LIST)