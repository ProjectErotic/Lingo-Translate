.PHONY: all build test clean cross-compile bump install uninstall package

BINARY_NAME=lingo
VERSION ?= $(shell cat VERSION 2>/dev/null || echo "2.0.0")
LDFLAGS=-s -w -X 'main.version=$(VERSION)' -X 'lingo-translate/pkg/version.Version=$(VERSION)'

bump:
	@./scripts/bump.sh $(v)

all: test build

build:
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME) ./cmd/lingo

desktop:
	cd frontend && npm run build
	go build -tags gtk3 -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME)-desktop ./cmd/lingo-desktop

test:
	CGO_ENABLED=0 go test -v ./pkg/...

cross-compile:
	mkdir -p dist
	# Linux amd64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY_NAME)-linux-amd64 ./cmd/lingo
	# Linux arm64
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY_NAME)-linux-arm64 ./cmd/lingo
	# Windows amd64
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY_NAME)-windows-amd64.exe ./cmd/lingo
	# macOS Apple Silicon (arm64)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY_NAME)-darwin-arm64 ./cmd/lingo
	# macOS Intel (amd64)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY_NAME)-darwin-amd64 ./cmd/lingo

clean:
	rm -rf bin dist workspace.nst

install: build
	@./scripts/install.sh

uninstall:
	@./scripts/install.sh --uninstall

package: build
	@mkdir -p dist/staging/lingo-v$(VERSION)-linux-amd64
	@cp bin/$(BINARY_NAME) dist/staging/lingo-v$(VERSION)-linux-amd64/
	@if [ -f bin/$(BINARY_NAME)-desktop ]; then cp bin/$(BINARY_NAME)-desktop dist/staging/lingo-v$(VERSION)-linux-amd64/; fi
	@cp build/appicon.png dist/staging/lingo-v$(VERSION)-linux-amd64/icon.png
	@cp build/linux/Lingo.desktop dist/staging/lingo-v$(VERSION)-linux-amd64/
	@cp scripts/install.sh dist/staging/lingo-v$(VERSION)-linux-amd64/
	@cp LICENSE dist/staging/lingo-v$(VERSION)-linux-amd64/
	@tar -czf dist/lingo-v$(VERSION)-linux-amd64.tar.gz -C dist/staging lingo-v$(VERSION)-linux-amd64
	@rm -rf dist/staging
	@echo "✅ Packaged dist/lingo-v$(VERSION)-linux-amd64.tar.gz"
