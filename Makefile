BINARY := ovaimporter
OSFLAG := $(shell go env GOHOSTOS)
GIT_COMMIT:=$(shell git rev-parse --short HEAD)
TIME:=$(shell date '+%FT%TZ')
REPO:= github.com/jacobweinstock

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[32m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: test
test: ## Run unit tests
	go test -v -covermode=count ./...

.PHONY: cover
cover: ## Run unit tests with coverage report
	go test -coverprofile=cover.out ./... || true
	go tool cover -func=cover.out
	rm -rf cover.out

.PHONY: lint
lint:  ## Run linting
	@echo be sure golangci-lint is installed: https://golangci-lint.run/usage/install/
	golangci-lint run

.PHONY: linux
linux: ## Compile for linux
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags '-s -w -X ${REPO}/${BINARY}/cmd.buildTime=${TIME} -X ${REPO}/${BINARY}/cmd.gitCommit=${GIT_COMMIT} -extldflags "-static"' -o bin/${BINARY}-linux main.go

.PHONY: darwin
darwin: ## Compile for darwin
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X ${REPO}/${BINARY}/cmd.buildTime=${TIME} -X ${REPO}/${BINARY}/cmd.gitCommit=${GIT_COMMIT} -extldflags '-static'" -o bin/${BINARY}-darwin main.go

.PHONY: build
build: ## Compile the binary for the native OS
ifeq (${OSFLAG},linux)
	@$(MAKE) linux
else
	@$(MAKE) darwin
endif

.PHONY: shell
shell: ## Drop into a shell with code mounted in
	docker run -it --rm -v ${PWD}:/code -w /code golang || true

.PHONY: image
image: ## Build container image
	docker build --rm -t ${BINARY} .