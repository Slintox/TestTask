.PHONY:
.DEFAULT_GOAL := help

# Not finished

# HELP =================================================================================================================
# This will output the help for each task
# thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

clean:
	rm -f ./.bin/TestTask.exe

build: clean
	go build -o ./.bin/TestTask.exe ./cmd/

run: build ## Run app
	./.bin/TestTask.exe -wbs 1 -rbs 1 -neg

test: ## Run test
	go test -v ./...

# Not finished
bench: ## Run bench
	go test -benchmem -bench ./cmd

race: ## Run test with race detection
	go test -v -race ./...

cover: ## Run coverage
	go test --short -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out
	rm coverage.out

# Not finished
profile: ## Run profiler
	go test -cpuprofile=profile.out -bench ./cmd/
