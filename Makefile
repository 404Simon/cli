.PHONY: build clean install help docs docs-hugo docs-custom

BINARY_NAME=pangolin
OUTPUT_DIR=bin
LDFLAGS=-ldflags="-s -w"

all: clean build

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(OUTPUT_DIR)
	@go build $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME) .
	@echo "Build complete: $(OUTPUT_DIR)/$(BINARY_NAME)"

clean:
	@echo "Cleaning..."
	@rm -rf $(OUTPUT_DIR)
	@echo "Clean complete"

install: build
	@echo "Installing $(BINARY_NAME)..."
	@go install $(LDFLAGS) .

docs:
	@echo "Generating markdown documentation..."
	@go run tools/gendocs/main.go -dir docs
	@echo "Documentation generated in docs/"

docs-hugo:
	@echo "Generating markdown documentation with Hugo front matter..."
	@go run tools/gendocs/main.go -dir docs -frontmatter -baseurl /commands
	@echo "Documentation generated in docs/"

docs-custom:
	@echo "Generating markdown documentation with custom output directory..."
	@if [ -z "$(DIR)" ]; then \
		echo "Usage: make docs-custom DIR=/path/to/output"; \
		exit 1; \
	fi
	@go run tools/gendocs/main.go -dir $(DIR)
