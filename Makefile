.PHONY: build test lint coverage clean \
       docker-build-api docker-build-ui docker-build-operator docker-build-all \
       docker-compose-up docker-compose-down test-postgres \
       kind-create kind-delete kind-load kind-deploy-all kind-undeploy-all \
       e2e-test deploy test-backend test-browser test-system test-live \
       test-all coverage-backend coverage-browser

# Resolve project root from Makefile location (works from any directory via make -C or absolute path)
PROJECT_ROOT := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
CONTAINER_ENGINE ?= $(shell docker info >/dev/null 2>&1 && echo docker || echo podman)
KUBE_CMD ?= kubectl --context kind-assethub

# === Quick commands (work from any directory) ===

deploy:
	"$(PROJECT_ROOT)scripts/kind-deploy.sh" rebuild "$(KUBE_CMD)"

test-backend:
	cd "$(PROJECT_ROOT)" && go test ./internal/... -count=1

test-browser:
	cd "$(PROJECT_ROOT)ui" && npx vitest run --config vitest.browser.config.ts

test-system:
	cd "$(PROJECT_ROOT)ui" && npx vitest run --config vitest.system.config.ts

test-live:
	"$(PROJECT_ROOT)scripts/test-containment-links.sh"
	"$(PROJECT_ROOT)scripts/test-data-viewer.sh"
	"$(PROJECT_ROOT)scripts/test-validation.sh"
	"$(PROJECT_ROOT)scripts/test-publishing.sh"
	"$(PROJECT_ROOT)scripts/test-copy-replace.sh"

test-all: test-backend test-browser test-system test-live

coverage-backend:
	cd "$(PROJECT_ROOT)" && go test ./internal/... -count=1 -coverprofile=coverage.out
	cd "$(PROJECT_ROOT)" && go tool cover -func=coverage.out | tail -1

coverage-browser:
	cd "$(PROJECT_ROOT)ui" && npx vitest run --config vitest.browser.config.ts --coverage

# === Original targets ===

build:
	cd "$(PROJECT_ROOT)" && go build ./...

test:
	cd "$(PROJECT_ROOT)" && go test ./... -v

lint:
	cd "$(PROJECT_ROOT)" && golangci-lint run ./...

coverage:
	cd "$(PROJECT_ROOT)" && go test ./... -coverprofile=coverage.out
	cd "$(PROJECT_ROOT)" && go tool cover -func=coverage.out

clean:
	rm -f "$(PROJECT_ROOT)coverage.out"

# Docker targets
docker-build-api:
	$(CONTAINER_ENGINE) build -f "$(PROJECT_ROOT)build/api-server/Dockerfile" -t assethub/api-server:latest "$(PROJECT_ROOT)"

docker-build-ui:
	$(CONTAINER_ENGINE) build -f "$(PROJECT_ROOT)build/ui/Dockerfile" -t assethub/ui:latest "$(PROJECT_ROOT)"

docker-build-operator:
	$(CONTAINER_ENGINE) build -f "$(PROJECT_ROOT)build/operator/Dockerfile" -t assethub/operator:latest "$(PROJECT_ROOT)"

docker-build-all: docker-build-api docker-build-ui docker-build-operator

# Docker Compose targets
docker-compose-up:
	cd "$(PROJECT_ROOT)" && $(CONTAINER_ENGINE) compose up -d

docker-compose-down:
	cd "$(PROJECT_ROOT)" && $(CONTAINER_ENGINE) compose down -v

# PostgreSQL integration tests
test-postgres:
	cd "$(PROJECT_ROOT)" && go test -tags postgres_integration ./internal/infrastructure/gorm/repository/ -v

# kind targets
kind-create:
	kind create cluster --config "$(PROJECT_ROOT)deploy/kind/kind-config.yaml" --name assethub

kind-delete:
	kind delete cluster --name assethub

kind-load: docker-build-all
ifeq ($(CONTAINER_ENGINE),podman)
	@echo "Using podman: retagging and saving images for kind"
	podman tag localhost/assethub/api-server:latest docker.io/assethub/api-server:latest
	podman save docker.io/assethub/api-server:latest -o /tmp/assethub-api-server.tar
	kind load image-archive /tmp/assethub-api-server.tar --name assethub
	rm -f /tmp/assethub-api-server.tar
	podman tag localhost/assethub/ui:latest docker.io/assethub/ui:latest
	podman save docker.io/assethub/ui:latest -o /tmp/assethub-ui.tar
	kind load image-archive /tmp/assethub-ui.tar --name assethub
	rm -f /tmp/assethub-ui.tar
	podman tag localhost/assethub/operator:latest docker.io/assethub/operator:latest
	podman save docker.io/assethub/operator:latest -o /tmp/assethub-operator.tar
	kind load image-archive /tmp/assethub-operator.tar --name assethub
	rm -f /tmp/assethub-operator.tar
else
	kind load docker-image assethub/api-server:latest --name assethub
	kind load docker-image assethub/ui:latest --name assethub
	kind load docker-image assethub/operator:latest --name assethub
endif

kind-deploy-all:
	$(KUBE_CMD) apply -f "$(PROJECT_ROOT)deploy/k8s/namespace.yaml"
	$(KUBE_CMD) apply -f "$(PROJECT_ROOT)deploy/k8s/postgres/"
	$(KUBE_CMD) apply -f "$(PROJECT_ROOT)deploy/k8s/api-server/"
	$(KUBE_CMD) apply -f "$(PROJECT_ROOT)deploy/k8s/ui/"
	$(KUBE_CMD) apply -f "$(PROJECT_ROOT)deploy/k8s/operator/crd.yaml"
	$(KUBE_CMD) apply -f "$(PROJECT_ROOT)deploy/k8s/operator/"

kind-undeploy-all:
	$(KUBE_CMD) delete -f "$(PROJECT_ROOT)deploy/k8s/operator/" --ignore-not-found
	$(KUBE_CMD) delete -f "$(PROJECT_ROOT)deploy/k8s/ui/" --ignore-not-found
	$(KUBE_CMD) delete -f "$(PROJECT_ROOT)deploy/k8s/api-server/" --ignore-not-found
	$(KUBE_CMD) delete -f "$(PROJECT_ROOT)deploy/k8s/postgres/" --ignore-not-found
	$(KUBE_CMD) delete -f "$(PROJECT_ROOT)deploy/k8s/namespace.yaml" --ignore-not-found

# E2E tests
e2e-test:
	cd "$(PROJECT_ROOT)" && go test -tags e2e ./test/e2e/ -v -timeout 300s
