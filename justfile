
# Commands for mcp-tf-provider-docs
default:
  @just --list
# Build mcp-tf-provider-docs with Go
build:
  go build ./...

# Run tests for mcp-tf-provider-docs with Go
test:
  go clean -testcache
  go test ./...