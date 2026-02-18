#!/usr/bin/env bash
#
# Deploy AI Asset Hub to a local kind cluster.
#
# Usage:
#   ./scripts/kind-deploy.sh          # full deploy (build + create cluster + deploy)
#   ./scripts/kind-deploy.sh up       # deploy using existing images (skip build)
#   ./scripts/kind-deploy.sh rebuild  # rebuild images and redeploy (keeps cluster)
#   ./scripts/kind-deploy.sh teardown # delete the cluster
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CLUSTER_NAME="assethub"

# Detect container engine — check that the daemon is actually reachable,
# not just that the binary exists on PATH.
ENGINE=""
if command -v docker &>/dev/null && docker info &>/dev/null; then
    ENGINE="docker"
elif command -v podman &>/dev/null && podman info &>/dev/null; then
    ENGINE="podman"
else
    echo "Error: no working container engine found."
    echo "  - docker is $(command -v docker 2>/dev/null && echo 'installed but the daemon is not running' || echo 'not installed')"
    echo "  - podman is $(command -v podman 2>/dev/null && echo 'installed but not responding' || echo 'not installed')"
    exit 1
fi

# Use KIND_EXPERIMENTAL_PROVIDER for podman
if [ "$ENGINE" = "podman" ]; then
    export KIND_EXPERIMENTAL_PROVIDER=podman
fi

echo "==> Using container engine: $ENGINE"

log() { echo "==> $*"; }
err() { echo "ERROR: $*" >&2; exit 1; }

# ──────────────────────────────────────────────
# Phase 1: Build container images
# ──────────────────────────────────────────────
build_images() {
    log "Building API server image..."
    $ENGINE build -f "$PROJECT_ROOT/build/api-server/Dockerfile" \
        -t assethub/api-server:latest "$PROJECT_ROOT"

    log "Building UI image..."
    $ENGINE build -f "$PROJECT_ROOT/build/ui/Dockerfile" \
        -t assethub/ui:latest "$PROJECT_ROOT"

    log "Building operator image..."
    $ENGINE build -f "$PROJECT_ROOT/build/operator/Dockerfile" \
        -t assethub/operator:latest "$PROJECT_ROOT"

    log "All images built successfully."
}

# ──────────────────────────────────────────────
# Phase 2: Create kind cluster
# ──────────────────────────────────────────────
create_cluster() {
    if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
        log "Cluster '$CLUSTER_NAME' already exists, skipping creation."
        return
    fi

    log "Creating kind cluster '$CLUSTER_NAME'..."
    kind create cluster \
        --config "$PROJECT_ROOT/deploy/kind/kind-config.yaml" \
        --name "$CLUSTER_NAME"

    log "Cluster created. Waiting for node to be ready..."
    kubectl wait --for=condition=Ready node --all --timeout=120s
}

# ──────────────────────────────────────────────
# Phase 3: Load images into kind
# ──────────────────────────────────────────────
load_images() {
    log "Loading images into kind cluster..."

    if [ "$ENGINE" = "podman" ]; then
        # kind load docker-image does not work with the podman provider.
        # Workaround: save to a tar archive and load that instead.
        # See https://github.com/kubernetes-sigs/kind/issues/3105
        #
        # Podman tags images as localhost/assethub/... but K8s manifests
        # reference them as assethub/... (no prefix). We retag before
        # saving so the archive contains the unprefixed name.
        local tmpdir
        tmpdir="$(mktemp -d)"
        trap "rm -rf '$tmpdir'" RETURN

        for img in assethub/api-server assethub/ui assethub/operator; do
            local archive="$tmpdir/$(echo "$img" | tr '/' '-').tar"
            # Retag to drop the localhost/ prefix
            podman tag "localhost/$img:latest" "docker.io/$img:latest" 2>/dev/null || true
            log "  Saving $img ..."
            podman save "docker.io/$img:latest" -o "$archive"
            log "  Loading $img into kind ..."
            kind load image-archive "$archive" --name "$CLUSTER_NAME"
            rm -f "$archive"   # free disk space between images
        done
    else
        kind load docker-image assethub/api-server:latest --name "$CLUSTER_NAME"
        kind load docker-image assethub/ui:latest --name "$CLUSTER_NAME"
        kind load docker-image assethub/operator:latest --name "$CLUSTER_NAME"
    fi

    log "Images loaded."
}

# ──────────────────────────────────────────────
# Phase 4: Deploy Kubernetes resources
# ──────────────────────────────────────────────
deploy_resources() {
    log "Deploying namespace..."
    kubectl apply -f "$PROJECT_ROOT/deploy/k8s/namespace.yaml"

    log "Deploying PostgreSQL..."
    kubectl apply -f "$PROJECT_ROOT/deploy/k8s/postgres/"

    log "Waiting for PostgreSQL to be ready..."
    kubectl -n assethub rollout status statefulset/postgres --timeout=120s

    log "Deploying operator CRDs and resources (operator manages API server and UI)..."
    kubectl apply -f "$PROJECT_ROOT/deploy/k8s/operator/crd.yaml"
    kubectl apply -f "$PROJECT_ROOT/deploy/k8s/operator/catalogversion-crd.yaml"
    kubectl apply -f "$PROJECT_ROOT/deploy/k8s/operator/"

    log "Deploying API server RBAC..."
    kubectl apply -f "$PROJECT_ROOT/deploy/k8s/api-server/rbac.yaml"

    log "Waiting for operator to be ready..."
    kubectl -n assethub rollout status deployment/assethub-operator --timeout=120s

    log "Waiting for operator-managed deployments..."
    # The operator creates these deployments from the CR; wait for them to appear
    local retries=30
    local delay=2
    for i in $(seq 1 $retries); do
        if kubectl -n assethub get deployment assethub-api &>/dev/null; then
            break
        fi
        if [ "$i" -eq "$retries" ]; then
            err "Operator did not create assethub-api deployment within ${retries} attempts"
        fi
        sleep "$delay"
    done
    kubectl -n assethub rollout status deployment/assethub-api --timeout=120s
    kubectl -n assethub rollout status deployment/assethub-ui --timeout=120s
}

# ──────────────────────────────────────────────
# Phase 5: Verify deployment
# ──────────────────────────────────────────────
verify() {
    log "Verifying deployment..."
    echo ""
    kubectl -n assethub get pods
    echo ""

    # Health check with retries
    local retries=10
    local delay=3
    for i in $(seq 1 $retries); do
        if curl -sf http://localhost:30080/healthz >/dev/null 2>&1; then
            log "API server health check passed."
            break
        fi
        if [ "$i" -eq "$retries" ]; then
            echo "Warning: API health check did not pass after ${retries} attempts."
            echo "The pods may still be starting. Check with: kubectl -n assethub get pods"
        else
            sleep "$delay"
        fi
    done

    echo ""
    echo "============================================"
    echo "  AI Asset Hub deployed successfully!"
    echo "============================================"
    echo ""
    echo "  API server:  http://localhost:30080"
    echo "  Health:       http://localhost:30080/healthz"
    echo "  Readiness:    http://localhost:30080/readyz"
    echo "  UI:           http://localhost:30000"
    echo ""
    echo "  Example API call:"
    echo "    curl -s http://localhost:30080/api/meta/v1/entity-types \\"
    echo "      -H 'X-User-Role: Admin' | jq ."
    echo ""
    echo "  Useful commands:"
    echo "    kubectl -n assethub get pods       # list pods"
    echo "    kubectl -n assethub logs -f <pod>  # stream logs"
    echo "    ./scripts/kind-deploy.sh teardown  # delete cluster"
    echo ""
}

# ──────────────────────────────────────────────
# Teardown
# ──────────────────────────────────────────────
teardown() {
    log "Deleting kind cluster '$CLUSTER_NAME'..."
    kind delete cluster --name "$CLUSTER_NAME"
    log "Cluster deleted."
}

# ──────────────────────────────────────────────
# Rebuild (keep cluster, rebuild images + redeploy)
# ──────────────────────────────────────────────
rebuild() {
    build_images
    load_images

    log "Restarting deployments..."
    kubectl -n assethub rollout restart deployment/assethub-operator
    kubectl -n assethub rollout restart deployment/assethub-api
    kubectl -n assethub rollout restart deployment/assethub-ui

    kubectl -n assethub rollout status deployment/assethub-operator --timeout=120s
    kubectl -n assethub rollout status deployment/assethub-api --timeout=120s
    kubectl -n assethub rollout status deployment/assethub-ui --timeout=120s

    verify
}

# ──────────────────────────────────────────────
# Main
# ──────────────────────────────────────────────
case "${1:-deploy}" in
    deploy)
        build_images
        create_cluster
        load_images
        deploy_resources
        verify
        ;;
    up)
        create_cluster
        load_images
        deploy_resources
        verify
        ;;
    rebuild)
        rebuild
        ;;
    teardown)
        teardown
        ;;
    *)
        echo "Usage: $0 {deploy|up|rebuild|teardown}"
        echo ""
        echo "  deploy   - build images, create cluster, deploy everything (full setup)"
        echo "  up       - create cluster and deploy using existing images (skip build)"
        echo "  rebuild  - rebuild images and redeploy (keep cluster)"
        echo "  teardown - delete the cluster"
        exit 1
        ;;
esac
