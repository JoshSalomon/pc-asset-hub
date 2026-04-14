# AI Asset Hub -- Deployment Guide

This guide covers deploying the AI Asset Hub on a local Kind cluster for development and on OpenShift for production.

## Local Development (Kind Cluster)

### Prerequisites

- **Podman** (preferred) or Docker
- **kind** (Kubernetes in Docker/Podman)
- **kubectl**

### Environment Setup

Kind requires an environment variable when using Podman as the container runtime:

```bash
export KIND_EXPERIMENTAL_PROVIDER=podman
```

### Cluster Operations

The `scripts/kind-deploy.sh` script handles all cluster lifecycle operations. It accepts an optional kubectl command (defaults to `oc`):

```bash
KUBE_CMD="kubectl --context kind-assethub"

# Full deploy: build images, create cluster, deploy everything
./scripts/kind-deploy.sh deploy "$KUBE_CMD"

# Rebuild: rebuild images and redeploy (keeps the cluster)
./scripts/kind-deploy.sh rebuild "$KUBE_CMD"

# Quick start: create cluster and deploy using existing images (skip build)
./scripts/kind-deploy.sh up "$KUBE_CMD"

# Teardown: delete the cluster entirely
./scripts/kind-deploy.sh teardown
```

The Makefile also provides a convenient `deploy` target that runs `rebuild`:

```bash
make -f /path/to/pc-asset-hub/Makefile deploy
```

### Exposed Ports

| Service    | Port  | URL                        |
|------------|-------|----------------------------|
| API Server | 30080 | `http://localhost:30080`   |
| UI         | 30000 | `http://localhost:30000`   |

The UI serves both the schema management interface (root path) and the operational catalog data viewer (`/operational` path) from the same port.

## Container Images

Three container images are built from Dockerfiles in the `build/` directory:

| Image                    | Dockerfile                      | Description                          |
|--------------------------|---------------------------------|--------------------------------------|
| `assethub/api-server`    | `build/api-server/Dockerfile`   | Go API server (Echo + GORM)          |
| `assethub/ui`            | `build/ui/Dockerfile`           | React/PatternFly SPA served by nginx |
| `assethub/operator`      | `build/operator/Dockerfile`     | K8s operator (operator-sdk)          |

Build all images with `make docker-build-all`. For Kind clusters, `kind-deploy.sh` handles image loading automatically (including Podman retag/save workarounds).

## Kubernetes Resources

All manifests live in `deploy/k8s/` and are organized by component:

### Namespace

- `namespace.yaml` -- Creates the `assethub` namespace for all resources.

### PostgreSQL (`deploy/k8s/postgres/`)

- `statefulset.yaml` -- PostgreSQL StatefulSet with persistent volume.
- `service.yaml` -- ClusterIP service for database access.
- `secret.yaml` -- Database credentials (username, password, database name).

### API Server (`deploy/k8s/api-server/`)

- `deployment.yaml` -- API server Deployment with environment variable configuration.
- `service.yaml` -- NodePort service exposing port 30080.
- `configmap.yaml` -- Configuration values (DB connection string, CORS origins, RBAC mode).
- `rbac.yaml` -- ServiceAccount, Role, and RoleBinding for K8s API access (CatalogVersion CR management).

### UI (`deploy/k8s/ui/`)

- `deployment.yaml` -- UI Deployment serving the React SPA via nginx.
- `service.yaml` -- NodePort service exposing port 30000.

### Operator (`deploy/k8s/operator/`)

- `crd.yaml` -- AssetHub CustomResourceDefinition.
- `catalogversion-crd.yaml` -- CatalogVersion CustomResourceDefinition.
- `catalog-crd.yaml` -- Catalog CustomResourceDefinition.
- `deployment.yaml` -- Operator Deployment.
- `serviceaccount.yaml`, `role.yaml`, `rolebinding.yaml` -- Operator RBAC.
- `sample-cr.yaml` -- Example AssetHub CR for testing.

## Configuration

The API server reads configuration from environment variables, set via the ConfigMap and Deployment in `deploy/k8s/api-server/`.

| Variable               | Default        | Description                                              |
|------------------------|----------------|----------------------------------------------------------|
| `DB_DRIVER`            | `sqlite`       | Database driver: `sqlite` or `postgres`                  |
| `DB_CONNECTION_STRING` | `assethub.db`  | Database connection string (file path or PG DSN)         |
| `API_PORT`             | `8080`         | Port the API server listens on                           |
| `RBAC_MODE`            | `header`       | Auth mode: `header` (dev) or `sar` (OpenShift SAR)      |
| `CORS_ALLOWED_ORIGINS` | (empty)        | Comma-separated list of allowed CORS origins             |
| `LOG_LEVEL`            | `info`         | Log verbosity level                                      |
| `CLUSTER_ROLE`         | `development`  | Controls visible lifecycle stages: `development`, `testing`, or `production` |

### RBAC Modes

- **`header`** (default, development): The API reads the user's role from request headers. No cluster authentication is performed. Suitable for local development and testing.
- **`sar`** (OpenShift): The API performs SubjectAccessReview requests against the K8s API to determine the caller's role. Requires a valid ServiceAccount token and appropriate RBAC configuration on the cluster.

### Cluster Role

The `CLUSTER_ROLE` setting controls which catalog version lifecycle stages are visible through the API:

| Cluster Role  | Visible Stages                          |
|---------------|-----------------------------------------|
| `development` | development, testing, production        |
| `testing`     | testing, production                     |
| `production`  | production only                         |

## Health Checks

The API server exposes two health endpoints:

- **`/healthz`** -- Liveness probe. Returns 200 if the process is running.
- **`/readyz`** -- Readiness probe. Returns 200 when the server is ready to accept traffic (database connection established).

These are configured as Kubernetes liveness and readiness probes in the API server Deployment manifest.

## Troubleshooting

### Pod CrashLoopBackOff After Restart

After a system restart, start the Kind node and wait for stabilization:

```bash
podman start assethub-control-plane
sleep 10
kubectl --context kind-assethub -n assethub get pods
```

If the API server is in `CrashLoopBackOff`, it started before PostgreSQL was ready. Delete the pod to force a restart:

```bash
kubectl --context kind-assethub -n assethub delete pod -l app=api-server
```

### Image Pull Errors

Kind uses locally loaded images. `ImagePullBackOff` means images were not loaded. Re-run `./scripts/kind-deploy.sh rebuild "kubectl --context kind-assethub"` to rebuild and reload.

### Checking Logs

```bash
kubectl --context kind-assethub -n assethub logs -l app=api-server -f
kubectl --context kind-assethub -n assethub logs -l app=operator -f
kubectl --context kind-assethub -n assethub logs -l app=postgres -f
```
