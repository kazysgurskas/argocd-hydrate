VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -X github.com/kazysgurskas/argocd-hydrate/internal/cmd.getVersion.Version=$(VERSION) \
           -X github.com/kazysgurskas/argocd-hydrate/internal/cmd.getVersion.GitCommit=$(GIT_COMMIT) \
           -X github.com/kazysgurskas/argocd-hydrate/internal/cmd.getVersion.BuildDate=$(BUILD_DATE)

.PHONY: build
build:
	go build -ldflags "$(LDFLAGS)" -o bin/argocd-hydrate cmd/argocd-hydrate/main.go

.PHONY: install
install:
	go install -ldflags "$(LDFLAGS)" ./cmd/argocd-hydrate

.PHONY: test
test:
	go test -v ./...

.PHONY: clean
clean:
	rm -rf bin/

.PHONY: docker
docker:
	docker build -t argocd-hydrate:$(VERSION) \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		.
