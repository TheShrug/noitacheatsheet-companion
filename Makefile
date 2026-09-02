# The repo's interface. Same verbs as every other repo in the fleet — build,
# test, run — so moving between them costs nothing.
#
# There is no `make database` here. This app has no database: it runs on a
# player's machine and its only state is a local queue file.

.DEFAULT_GOAL := help

VERSION ?= dev
GOFLAGS ?=

# -trimpath and the ldflags match the Dockerfile's release build, so a binary
# you build by hand behaves like the one CI ships.
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: help
help: ## List the targets
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build ./dist/companion for this machine
	@mkdir -p dist
	go build $(GOFLAGS) -trimpath -ldflags "$(LDFLAGS)" -o dist/ ./cmd/...

.PHONY: test
test: ## Run the suite, plus gofmt and vet
	@test -z "$$(gofmt -l . | tee /dev/stderr)" || { echo "gofmt: files need formatting (run make fmt)"; exit 1; }
	go vet ./...
	go test ./... -race -count=1

.PHONY: run
run: build ## Report where Noita's files are on this machine
	@# Not a server yet. When the review queue lands this becomes the
	@# background process and prints http://127.0.0.1:7331 as its last line,
	@# matching the fleet's `make run` contract.
	./dist/companion paths

.PHONY: fmt
fmt: ## Format every Go file in place
	gofmt -w .

.PHONY: dist
dist: ## Build the release binaries for every platform, in Docker
	docker build --target dist --build-arg VERSION=$(VERSION) --output dist .
	@ls -l dist

.PHONY: clean
clean: ## Remove build output
	rm -rf dist
