VERSION ?= v$(shell cat VERSION)
COMMIT := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GO ?= go
GORELEASER ?= goreleaser
LDFLAGS := -s -w -X github.com/impishMD/TExec/util.Ver=$(VERSION) -X github.com/impishMD/TExec/util.Commit=$(COMMIT) -X github.com/impishMD/TExec/util.Date=$(BUILD_DATE)

.PHONY: deps build frontend backend test lint docs docs-check check container release-check release-snapshot version-check helm-check
deps:
	$(GO) mod download
	cd web && npm ci --no-audit --no-fund
frontend:
	cd web && npm run build
backend:
	CGO_ENABLED=0 $(GO) build -trimpath -tags netgo -ldflags '$(LDFLAGS)' -o bin/texec ./cli
build: frontend backend
test:
	$(GO) test ./...
	cd web && npm run test:unit
lint:
	$(GO) vet ./...
	cd web && npm run lint -- --no-fix
docs:
	$(GO) run ./tools/docsref -out docs/en/reference/configuration.md
	$(GO) run ./tools/clidocs -out docs/en/reference/cli.md
docs-check: docs
	git diff --exit-code -- docs/en/reference
	python3 tools/check_docs.py
check: lint docs-check version-check
container:
	docker build --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) -t impishmd/texec:$(VERSION) .
version-check:
	python3 tools/check-version.py
helm-check:
	helm lint charts/texec --strict -f charts/texec/ci/test-values.yaml
	python3 tools/check-chart.py
release-check: version-check
	$(GORELEASER) check
release-snapshot:
	TEXEC_SNAPSHOT_VERSION=$(shell cat VERSION)-dev $(GORELEASER) release --snapshot --clean --skip=publish
