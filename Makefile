.PHONY: build test lint fmt check tidy run install

build: ## Build the binary
	go build -o ./dist/mi .

test: ## Run all tests
	go test ./...

lint: ## Run golangci-lint
	golangci-lint run ./...

fmt: ## Run golangci-lint fmt
	golangci-lint fmt

check: ## Run all checks (tidy, lint, test)
	$(MAKE) tidy
	$(MAKE) lint
	$(MAKE) test

tidy: ## Run go mod tidy
	go mod tidy

run: ## Run the application
	go run .

install: ## Build and install to ~/go/bin
	go build -o ~/go/bin/mi .
