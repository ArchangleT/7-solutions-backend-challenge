BIN_DIR   := $(shell go env GOPATH)/bin
OAPI      := $(BIN_DIR)/oapi-codegen
MOCKGEN   := $(BIN_DIR)/mockgen
COVERAGE  := coverage.out
COVERAGE_HTML := coverage.html

.PHONY: run test test-integration tools codegen mock docsgen clean help

## run: start the API server (add APP_ENV=prod for production config)
run:
	go run ./cmd/api

## test: run the full test suite with coverage profile (generated code excluded)
test:
	go test ./... -coverprofile=$(COVERAGE)
	grep -v -E '/(internal/(generated|mocks))/' $(COVERAGE) > $(COVERAGE).filtered
	go tool cover -func=$(COVERAGE).filtered
	rm -f $(COVERAGE).filtered

## test-integration: run the opt-in live MongoDB connectivity test
test-integration:
	TEST_MONGO_URI="$${TEST_MONGO_URI:?set TEST_MONGO_URI to a live MongoDB URI}" \
		go test ./internal/infrastructure/db/ -run TestConnect_Live -count=1 -v

## tools: install oapi-codegen and mockgen into $(go env GOPATH)/bin
tools: $(OAPI) $(MOCKGEN)

$(OAPI):
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

$(MOCKGEN):
	go install go.uber.org/mock/mockgen@latest

## codegen: regenerate the OpenAPI server interfaces and models
codegen: $(OAPI)
	$(OAPI) --config api/codegen.yaml api/openapi.yaml
	gofmt -w internal/generated/api/openapi.gen.go

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
	npx --yes @redocly/cli bundle api/openapi.yaml -o docs/openapi.yaml --ext yaml

## clean: remove generated artifacts
clean:
	rm -rf $(COVERAGE) $(COVERAGE_HTML) docs

## help: list available targets
help:
	@echo "Targets:"
	@awk '/^[a-zA-Z0-9_-]+:/ {help=0} /^## / {help=1} help {sub(/^## /, ""); print $$0}' $(MAKEFILE_LIST)