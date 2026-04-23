#!/usr/bin/env bash
#
# Deploy AI Asset Hub to a Kubernetes cluster.
#
# Usage:
#   ./scripts/kind-deploy.sh [--source-dir DIR] [command] [kube-cmd]
#
# Options:
#   --source-dir DIR  Use DIR as Docker build context instead of this repo.
#                     Must contain go.mod and ui/. Useful for worktree builds.
#
# Commands:
#   deploy   - build images, create cluster, deploy everything (full setup)
#   up       - create cluster and deploy using existing images (skip build)
#   rebuild  - rebuild images and redeploy (keep cluster)
#   teardown - delete the cluster
#
# kube-cmd (optional):
#   The kubectl/oc command (with flags) to use for all cluster operations.
#   Defaults to "oc". Examples:
#     oc
#     kubectl --context kind-assethub
#     kubectl --context my-cluster --namespace my-ns
#
# Examples:
#   ./scripts/kind-deploy.sh deploy "kubectl --context kind-assethub"
#   ./scripts/kind-deploy.sh rebuild "kubectl --context kind-assethub"
#   ./scripts/kind-deploy.sh --source-dir .worktrees/my-branch rebuild "kubectl --context kind-assethub"
#   ./scripts/kind-deploy.sh teardown
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CLUSTER_NAME="assethub"

# Source directory for Docker build context — defaults to PROJECT_ROOT.
# Override with --source-dir to build from a worktree.
SOURCE_DIR="$PROJECT_ROOT"

# Parse --source-dir option (must come before positional args)
while [[ "${1:-}" == --* ]]; do
    case "$1" in
        --source-dir)
            shift
            candidate="$(realpath -e "$1" 2>/dev/null)" || { echo "ERROR: --source-dir: directory does not exist: $1" >&2; exit 1; }
            [[ -f "$candidate/go.mod" && -d "$candidate/ui" ]] || { echo "ERROR: --source-dir: not a valid project root: $candidate" >&2; exit 1; }
            SOURCE_DIR="$candidate"
            shift
            ;;
        *) echo "ERROR: unknown option: $1" >&2; exit 1 ;;
    esac
done

# Kubernetes CLI command — second positional argument, defaults to "oc"
KUBE_CMD="${2:-oc}"

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
echo "==> Using kube command: $KUBE_CMD"
if [ "$SOURCE_DIR" != "$PROJECT_ROOT" ]; then
    echo "==> Using source dir: $SOURCE_DIR (worktree)"
fi

log() { echo "==> $*"; }
err() { echo "ERROR: $*" >&2; exit 1; }

# ──────────────────────────────────────────────
# Phase 1: Build container images
# ──────────────────────────────────────────────
build_images() {
    log "Building API server image..."
    $ENGINE build -f "$SOURCE_DIR/build/api-server/Dockerfile" \
        -t assethub/api-server:latest "$SOURCE_DIR"

    log "Building UI image..."
    $ENGINE build -f "$SOURCE_DIR/build/ui/Dockerfile" \
        -t assethub/ui:latest "$SOURCE_DIR"

    log "Building operator image..."
    $ENGINE build -f "$SOURCE_DIR/build/operator/Dockerfile" \
        -t assethub/operator:latest "$SOURCE_DIR"

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
    $KUBE_CMD wait --for=condition=Ready node --all --timeout=120s
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
    $KUBE_CMD apply -f "$PROJECT_ROOT/deploy/k8s/namespace.yaml"

    log "Deploying PostgreSQL..."
    $KUBE_CMD apply -f "$PROJECT_ROOT/deploy/k8s/postgres/"

    log "Waiting for PostgreSQL to be ready..."
    $KUBE_CMD -n assethub rollout status statefulset/postgres --timeout=120s

    log "Deploying operator CRDs and resources (operator manages API server and UI)..."
    $KUBE_CMD apply -f "$PROJECT_ROOT/deploy/k8s/operator/crd.yaml"
    $KUBE_CMD apply -f "$PROJECT_ROOT/deploy/k8s/operator/catalogversion-crd.yaml"
    $KUBE_CMD apply -f "$PROJECT_ROOT/deploy/k8s/operator/catalog-crd.yaml"
    $KUBE_CMD apply -f "$PROJECT_ROOT/deploy/k8s/operator/"

    log "Deploying API server RBAC..."
    $KUBE_CMD apply -f "$PROJECT_ROOT/deploy/k8s/api-server/rbac.yaml"

    log "Waiting for operator to be ready..."
    $KUBE_CMD -n assethub rollout status deployment/assethub-operator --timeout=120s

    log "Waiting for operator-managed deployments..."
    # The operator creates these deployments from the CR; wait for them to appear
    local retries=30
    local delay=2
    for i in $(seq 1 $retries); do
        if $KUBE_CMD -n assethub get deployment assethub-api &>/dev/null; then
            break
        fi
        if [ "$i" -eq "$retries" ]; then
            err "Operator did not create assethub-api deployment within ${retries} attempts"
        fi
        sleep "$delay"
    done
    $KUBE_CMD -n assethub rollout status deployment/assethub-api --timeout=120s
    $KUBE_CMD -n assethub rollout status deployment/assethub-ui --timeout=120s
}

# ──────────────────────────────────────────────
# Phase 5: Verify deployment
# ──────────────────────────────────────────────
verify() {
    log "Verifying deployment..."
    echo ""
    $KUBE_CMD -n assethub get pods
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
            echo "The pods may still be starting. Check with: $KUBE_CMD -n assethub get pods"
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
    echo "    $KUBE_CMD -n assethub get pods       # list pods"
    echo "    $KUBE_CMD -n assethub logs -f <pod>  # stream logs"
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

    # On kind clusters, rollout restart alone doesn't guarantee new images are used
    # because imagePullPolicy=Never and the tag (latest) doesn't change. The pod
    # spec hash stays the same so pods may not be recreated. Deleting pods forces
    # the deployment controller to create new ones from the updated image cache.
    if echo "$KUBE_CMD" | grep -q "kind"; then
        log "Kind cluster detected — deleting pods to force image refresh..."
        $KUBE_CMD -n assethub delete pods -l app=assethub-operator --wait=false 2>/dev/null || true
        $KUBE_CMD -n assethub delete pods -l app=assethub-api --wait=false 2>/dev/null || true
        $KUBE_CMD -n assethub delete pods -l app=assethub-ui --wait=false 2>/dev/null || true
    else
        log "Restarting deployments..."
        $KUBE_CMD -n assethub rollout restart deployment/assethub-operator
        $KUBE_CMD -n assethub rollout restart deployment/assethub-api
        $KUBE_CMD -n assethub rollout restart deployment/assethub-ui
    fi

    $KUBE_CMD -n assethub rollout status deployment/assethub-operator --timeout=120s
    $KUBE_CMD -n assethub rollout status deployment/assethub-api --timeout=120s
    $KUBE_CMD -n assethub rollout status deployment/assethub-ui --timeout=120s

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
        echo "Usage: $0 [--source-dir DIR] {deploy|up|rebuild|teardown} [kube-cmd]"
        echo ""
        echo "  Options:"
        echo "    --source-dir DIR  Build from DIR instead of this repo (e.g., a worktree)"
        echo ""
        echo "  Commands:"
        echo "    deploy   - build images, create cluster, deploy everything (full setup)"
        echo "    up       - create cluster and deploy using existing images (skip build)"
        echo "    rebuild  - rebuild images and redeploy (keep cluster)"
        echo "    teardown - delete the cluster"
        echo ""
        echo "  kube-cmd (optional, default: oc):"
        echo "    The kubectl/oc command to use for cluster operations."
        echo "    Examples:"
        echo "      $0 deploy \"kubectl --context kind-assethub\""
        echo "      $0 --source-dir .worktrees/my-branch rebuild \"kubectl --context kind-assethub\""
        exit 1
        ;;
esac
