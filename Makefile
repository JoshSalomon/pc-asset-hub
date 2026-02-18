.PHONY: build test lint coverage clean \
       docker-build-api docker-build-ui docker-build-operator docker-build-all \
       docker-compose-up docker-compose-down test-postgres \
       kind-create kind-delete kind-load kind-deploy-all kind-undeploy-all \
       e2e-test

CONTAINER_ENGINE ?= $(shell docker info >/dev/null 2>&1 && echo docker || echo podman)

build:
	go build ./...

test:
	go test ./... -v

lint:
	golangci-lint run ./...

coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

clean:
	rm -f coverage.out

# Docker targets
docker-build-api:
	$(CONTAINER_ENGINE) build -f build/api-server/Dockerfile -t assethub/api-server:latest .

docker-build-ui:
	$(CONTAINER_ENGINE) build -f build/ui/Dockerfile -t assethub/ui:latest .

docker-build-operator:
	$(CONTAINER_ENGINE) build -f build/operator/Dockerfile -t assethub/operator:latest .

docker-build-all: docker-build-api docker-build-ui docker-build-operator

# Docker Compose targets
docker-compose-up:
	$(CONTAINER_ENGINE) compose up -d

docker-compose-down:
	$(CONTAINER_ENGINE) compose down -v

# PostgreSQL integration tests
test-postgres:
	go test -tags postgres_integration ./internal/infrastructure/gorm/repository/ -v

# kind targets
kind-create:
	kind create cluster --config deploy/kind/kind-config.yaml --name assethub

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
	kubectl apply -f deploy/k8s/namespace.yaml
	kubectl apply -f deploy/k8s/postgres/
	kubectl apply -f deploy/k8s/api-server/
	kubectl apply -f deploy/k8s/ui/
	kubectl apply -f deploy/k8s/operator/crd.yaml
	kubectl apply -f deploy/k8s/operator/

kind-undeploy-all:
	kubectl delete -f deploy/k8s/operator/ --ignore-not-found
	kubectl delete -f deploy/k8s/ui/ --ignore-not-found
	kubectl delete -f deploy/k8s/api-server/ --ignore-not-found
	kubectl delete -f deploy/k8s/postgres/ --ignore-not-found
	kubectl delete -f deploy/k8s/namespace.yaml --ignore-not-found

# E2E tests
e2e-test:
	go test -tags e2e ./test/e2e/ -v -timeout 300s
