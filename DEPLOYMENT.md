# Deploying AI Asset Hub on Kubernetes (kind)

This guide walks through deploying the full AI Asset Hub stack on a local
[kind](https://kind.sigs.k8s.io/) (Kubernetes in Docker) cluster.

## Architecture

```
┌──────────────────────────────────────────────────────┐
│                    kind cluster                      │
│                                                      │
│  ┌────────────┐   ┌─────────────┐   ┌────────────┐   │
│  │ PostgreSQL │◄──│ API Server  │◄──│     UI     │   │
│  │ StatefulSet│   │ Deployment  │   │ Deployment │   │
│  │ port 5432  │   │ port 8080   │   │ port 80    │   │
│  └────────────┘   └──────┬──────┘   └─────┬──────┘   │
│                   NodePort 30080    NodePort 30000   │
│                          │                           │
│                          │ on promote/demote         │
│                          ▼                           │
│                  CatalogVersion CRs                  │
│                          │                           │
│  ┌─────────────────────┐ │ watches                   │
│  │ AssetHub Operator   │◄┘                           │
│  │ Deployment          │──▶ owner refs, status       │
│  │ watches AssetHub +  │                             │
│  │ CatalogVersion CRs  │                             │
│  └─────────────────────┘                             │
└──────────────────────────────────────────────────────┘
        │                        │
   localhost:30080          localhost:30000
    (API server)               (UI)
```

The deployment is **operator-driven**. Only PostgreSQL and the operator
itself are deployed directly from manifests. The operator reconciles the
`AssetHub` CR and creates the API server and UI Deployments, Services,
and ConfigMap automatically.

**Components:**

| Component | Image | Description |
|-----------|-------|-------------|
| API Server | `assethub/api-server` | Go REST API with RBAC, health probes, PostgreSQL backend. Manages `CatalogVersion` CRs when running in K8s. Created by the operator. |
| UI | `assethub/ui` | React/PatternFly SPA served by nginx. Created by the operator. |
| Operator | `assethub/operator` | Kubernetes operator that reconciles `AssetHub` CRs (creates API server + UI) and `CatalogVersion` CRs (sets owner refs, updates status). |
| PostgreSQL | `postgres:16-alpine` | Database backend (StatefulSet with persistent volume). Deployed directly from manifests. |

## Prerequisites

Install the following tools before proceeding:

| Tool | Minimum Version | Install |
|------|----------------|---------|
| **Go** | 1.25.7 | [go.dev/dl](https://go.dev/dl/) |
| **Node.js** | 22 | [nodejs.org](https://nodejs.org/) |
| **Docker** or **Podman** | Docker 24+ / Podman 4+ | [docker.com](https://docs.docker.com/get-docker/) or [podman.io](https://podman.io/getting-started/installation) |
| **kind** | 0.20+ | [kind.sigs.k8s.io](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) |
| **kubectl** | 1.28+ | [kubernetes.io](https://kubernetes.io/docs/tasks/tools/) |
| **curl** | any | typically pre-installed |

Verify prerequisites:

```bash
go version          # go1.25.7 or later
node --version      # v22.x
docker version      # or: podman version
kind version        # v0.20+
kubectl version --client
```

## Quick Start

The fastest way to deploy is the all-in-one script:

```bash
./scripts/kind-deploy.sh
```

This single command will:

1. Build all three container images (API server, UI, operator)
2. Create a kind cluster with port mappings
3. Load images into the cluster
4. Deploy PostgreSQL
5. Deploy CRDs (AssetHub + CatalogVersion), operator RBAC, operator, and AssetHub CR
6. Deploy API server RBAC (ServiceAccount for CatalogVersion CR management)
7. Wait for the operator to create and reconcile the API server and UI deployments
8. Verify the health endpoint

Once complete, the services are available at:

- **API server:** http://localhost:30080
- **UI:** http://localhost:30000

### Script Commands

The deploy script accepts an optional command argument:

```bash
./scripts/kind-deploy.sh [command]
```

| Command | Description | Builds Images | Creates Cluster | Deploys |
|---------|-------------|:---:|:---:|:---:|
| `deploy` (default) | Full setup from scratch | Yes | Yes | Yes |
| `up` | Deploy using existing images (skip build) | No | Yes | Yes |
| `rebuild` | Rebuild images and redeploy into existing cluster | Yes | No | Yes (restart) |
| `teardown` | Delete the cluster and all resources | - | Deletes | - |

**When to use each:**

- **`deploy`** — First-time setup or starting fresh after a teardown.
- **`up`** — Images are already built (e.g., after a previous `deploy`). Faster startup when only the cluster was deleted.
- **`rebuild`** — Code changed and you want to update the running cluster without recreating it. Rebuilds images, reloads into kind, and restarts all deployments.
- **`teardown`** — Done working. Removes the kind cluster, all pods, volumes, and loaded images.

## Step-by-Step Deployment

If you prefer to understand each step, follow this manual walkthrough.

### 1. Build Container Images

From the project root:

```bash
# Build all three images (auto-detects docker or podman)
make docker-build-all
```

Or build individually:

```bash
make docker-build-api       # API server (Go, distroless, ~20MB)
make docker-build-ui        # UI (React build + nginx, ~30MB)
make docker-build-operator  # Operator (Go, distroless, ~45MB)
```

Verify the images exist:

```bash
docker images | grep assethub   # or: podman images | grep assethub
```

Expected output:

```
assethub/api-server   latest   ...   ...   ~20MB
assethub/ui           latest   ...   ...   ~30MB
assethub/operator     latest   ...   ...   ~45MB
```

### 2. Create the kind Cluster

```bash
make kind-create
```

This creates a single-node cluster named `assethub` with two host port
mappings:

| Host Port | Container Port | Service |
|-----------|---------------|---------|
| 30080 | 30080 | API server (NodePort) |
| 30000 | 30000 | UI (NodePort) |

Verify the cluster is running:

```bash
kubectl --context=kind-assethub cluster-info
kubectl --context=kind-assethub get nodes
```

If using **podman**, set this before running kind:

```bash
export KIND_EXPERIMENTAL_PROVIDER=podman
```

### 3. Load Images into kind

kind runs its own container registry. Images built on the host must be
loaded explicitly:

```bash
make kind-load
```

This loads all three `assethub/*:latest` images into the cluster. The
operator sets `imagePullPolicy: Never` in development mode so the
cluster uses these local images directly.

### 4. Deploy the Stack

```bash
make kind-deploy-all
```

This applies Kubernetes manifests in dependency order:

1. **Namespace** (`assethub`)
2. **PostgreSQL** — Secret, Service, StatefulSet
3. **CRDs** — AssetHub CRD + CatalogVersion CRD
4. **Operator** — ServiceAccount, Role (incl. catalogversions permissions), RoleBinding, Deployment, sample AssetHub CR
5. **API Server RBAC** — ServiceAccount, Role (manage CatalogVersion CRs), RoleBinding

The operator then reconciles the AssetHub CR and creates:

- **API server** — ConfigMap (`api-server-config`), Deployment (`assethub-api`), Service (`assethub-api-svc`)
- **UI** — Deployment (`assethub-ui`), Service (`assethub-ui-svc`)

The ConfigMap includes environment-driven settings (RBAC mode, DB
connection, CORS origins, log level) and the `CLUSTER_ROLE` value.

### 5. Wait for Pods

Watch the pods come up:

```bash
kubectl --context=kind-assethub -n assethub get pods -w
```

All pods should reach `Running` and `1/1 Ready` status. The API server
won't become ready until PostgreSQL passes its readiness probe.

Expected output:

```
NAME                                 READY   STATUS    RESTARTS   AGE
assethub-api-xxxxx-xxxxx             1/1     Running   0          30s
assethub-operator-xxxxx-xxxxx        1/1     Running   0          45s
assethub-ui-xxxxx-xxxxx              1/1     Running   0          30s
postgres-0                           1/1     Running   0          60s
```

### 6. Verify the Deployment

Health check:

```bash
curl http://localhost:30080/healthz
# {"status":"ok"}

curl http://localhost:30080/readyz
# {"status":"ready"}
```

Verify the CatalogVersion CRD is registered:

```bash
kubectl --context=kind-assethub -n assethub get crd catalogversions.assethub.project-catalyst.io
```

Verify the operator reconciled the ConfigMap with `CLUSTER_ROLE`:

```bash
kubectl --context=kind-assethub -n assethub get configmap api-server-config -o jsonpath='{.data.CLUSTER_ROLE}'
# development
```

List entity types (empty initially):

```bash
curl -s http://localhost:30080/api/meta/v1/entity-types \
  -H 'X-User-Role: Admin' | jq .
# {"items":[],"total":0}
```

Open the UI in a browser:

```
http://localhost:30000
```

## Deploying the Operator Custom Resource

The operator watches for `AssetHub` custom resources and creates
Deployments, Services, and ConfigMaps based on the spec. The sample CR
is applied automatically by the deploy script, but you can re-apply or
modify it:

```bash
kubectl --context=kind-assethub apply -f deploy/k8s/operator/sample-cr.yaml
```

The sample CR spec:

```yaml
spec:
  replicas: 1
  dbConnection: "host=postgres user=assethub password=assethub dbname=assethub port=5432 sslmode=disable"
  uiReplicas: 1
  environment: development     # development | openshift
  apiNodePort: 30080
  uiNodePort: 30000
  logLevel: info
  clusterRole: development     # development | testing | production
```

The `clusterRole` field controls which catalog version lifecycle stages
the API server exposes:

| clusterRole | API serves |
|-------------|-----------|
| `development` (default) | development + testing + production catalog versions |
| `testing` | testing + production catalog versions |
| `production` | production catalog versions only |

The `environment` field controls infrastructure behavior:

| environment | Service type | Image pull | RBAC mode | Routes |
|-------------|-------------|-----------|-----------|--------|
| `development` (default) | NodePort | Never | header | none |
| `openshift` | ClusterIP | IfNotPresent | token | TLS edge |

Verify the operator reconciled:

```bash
kubectl --context=kind-assethub -n assethub get assethub
kubectl --context=kind-assethub -n assethub get deployments
```

## Using the API

All API requests require the `X-User-Role` header (development mode).
Available roles (in ascending privilege order): `RO`, `RW`, `Admin`,
`SuperAdmin`.

**Note:** Header-based RBAC is a development convenience. The API server
logs a warning on startup: `WARNING: RBAC is using header-based mode
(X-User-Role). This is an insecure development configuration.` In
production (`environment: openshift`), token-based authentication is
used.

### Create an Entity Type

```bash
curl -s -X POST http://localhost:30080/api/meta/v1/entity-types \
  -H 'Content-Type: application/json' \
  -H 'X-User-Role: Admin' \
  -d '{"name": "MLModel", "description": "Machine learning model"}' | jq .
```

### Create a Catalog Version

```bash
# Use the entity_type_version ID from the previous response
curl -s -X POST http://localhost:30080/api/meta/v1/catalog-versions \
  -H 'Content-Type: application/json' \
  -H 'X-User-Role: Admin' \
  -d '{
    "version_label": "v1.0",
    "pins": [{"entity_type_version_id": "<VERSION_ID>"}]
  }' | jq .
```

### Promote a Catalog Version

```bash
curl -s -X POST http://localhost:30080/api/meta/v1/catalog-versions/<CV_ID>/promote \
  -H 'X-User-Role: Admin' | jq .
```

On promotion to testing, a `CatalogVersion` CR is created in K8s:

```bash
kubectl --context=kind-assethub -n assethub get cv
# NAME    AGE
# v1-0    5s
```

### Create an Instance

```bash
curl -s -X POST http://localhost:30080/api/data/v1/<CV_ID>/MLModel \
  -H 'Content-Type: application/json' \
  -H 'X-User-Role: Admin' \
  -d '{"name": "llama-3-70b", "description": "Meta Llama 3 70B"}' | jq .
```

## Updating After Code Changes

After modifying source code, rebuild and redeploy without recreating the
cluster:

```bash
./scripts/kind-deploy.sh rebuild
```

This rebuilds all images, reloads them into kind, and restarts the
deployments (operator, API server, UI).

If CRDs or RBAC manifests changed, re-apply them before restarting:

```bash
kubectl --context=kind-assethub apply -f deploy/k8s/operator/crd.yaml
kubectl --context=kind-assethub apply -f deploy/k8s/operator/catalogversion-crd.yaml
kubectl --context=kind-assethub apply -f deploy/k8s/operator/role.yaml
kubectl --context=kind-assethub apply -f deploy/k8s/api-server/rbac.yaml
kubectl --context=kind-assethub apply -f deploy/k8s/operator/sample-cr.yaml
```

## Teardown

Delete the entire cluster:

```bash
./scripts/kind-deploy.sh teardown
```

Or using make:

```bash
make kind-delete
```

This removes the cluster, all pods, volumes, and loaded images.

## Alternative: Docker Compose (without Kubernetes)

For a simpler local setup without Kubernetes, use docker-compose:

```bash
# Build images first
make docker-build-all

# Start the stack (PostgreSQL + API + UI)
make docker-compose-up
```

Services:

| Service | URL |
|---------|-----|
| API server | http://localhost:8080 |
| UI | http://localhost:3000 |
| PostgreSQL | localhost:5432 |

Stop and remove volumes:

```bash
make docker-compose-down
```

In docker-compose mode, the API server runs without a K8s client.
CatalogVersion CR management is disabled — promotion and demotion still
work but only update the database.

## Troubleshooting

### Pods stuck in `Pending` or `ImagePullBackOff`

Images were not loaded into kind, or `imagePullPolicy` is not set to
`Never`. The operator sets `imagePullPolicy: Never` when
`environment: development` is set on the AssetHub CR. Verify:

```bash
kubectl --context=kind-assethub -n assethub get deployment assethub-api \
  -o jsonpath='{.spec.template.spec.containers[0].imagePullPolicy}'
# Should print: Never
```

If not, re-apply the sample CR (which includes `environment: development`):

```bash
kubectl --context=kind-assethub apply -f deploy/k8s/operator/sample-cr.yaml
```

Also ensure images are loaded:

```bash
make kind-load
```

### API server `CrashLoopBackOff`

PostgreSQL may not be ready yet. Check its status:

```bash
kubectl --context=kind-assethub -n assethub get pods -l app=postgres
kubectl --context=kind-assethub -n assethub logs postgres-0
```

The API server depends on PostgreSQL being reachable. Its readiness probe
will fail until the database connection succeeds.

### CRD validation errors on `kubectl apply`

If applying the sample CR fails with `unknown field` errors, the CRD
in the cluster is outdated. Re-apply the CRDs:

```bash
kubectl --context=kind-assethub apply -f deploy/k8s/operator/crd.yaml
kubectl --context=kind-assethub apply -f deploy/k8s/operator/catalogversion-crd.yaml
```

Then re-apply the CR:

```bash
kubectl --context=kind-assethub apply -f deploy/k8s/operator/sample-cr.yaml
```

### Services are ClusterIP instead of NodePort (ports unreachable)

If `localhost:30080` or `localhost:30000` is unreachable after a rebuild,
the operator-managed services may be stale. This happens when services
were originally created by an older operator build (e.g., before
`environment: development` was set on the CR). Kubernetes does not allow
changing a service's type from ClusterIP to NodePort via update — the
services must be deleted and recreated.

Check the service types:

```bash
kubectl --context=kind-assethub -n assethub get svc
```

If `assethub-api-svc` or `assethub-ui-svc` show `ClusterIP` instead of
`NodePort`, delete them and restart the operator:

```bash
kubectl --context=kind-assethub -n assethub delete svc assethub-api-svc assethub-ui-svc
kubectl --context=kind-assethub -n assethub rollout restart deployment/assethub-operator
```

The operator will recreate the services with the correct NodePort type.

### Port already in use

If port 30080 or 30000 is already bound:

```bash
# Find the process using the port
ss -tlnp | grep 30080
```

Either stop the conflicting process or edit
`deploy/kind/kind-config.yaml` to change the `hostPort` values.

### Podman-specific issues

Set the kind provider before creating the cluster:

```bash
export KIND_EXPERIMENTAL_PROVIDER=podman
```

If `kind load docker-image` fails, ensure the podman socket is running:

```bash
systemctl --user start podman.socket
```

### Viewing logs

```bash
# API server logs
kubectl --context=kind-assethub -n assethub logs -f deployment/assethub-api

# Operator logs
kubectl --context=kind-assethub -n assethub logs -f deployment/assethub-operator

# PostgreSQL logs
kubectl --context=kind-assethub -n assethub logs -f postgres-0
```

### Running E2E tests against the cluster

With the cluster running:

```bash
make e2e-test
```

This runs the E2E test suite in `test/e2e/` against `http://localhost:30080`.

## Manifest Reference

```
deploy/
├── kind/
│   └── kind-config.yaml              # kind cluster configuration
└── k8s/
    ├── namespace.yaml                # assethub namespace
    ├── postgres/
    │   ├── secret.yaml               # database credentials
    │   ├── service.yaml              # headless service
    │   └── statefulset.yaml          # PostgreSQL with PVC
    ├── api-server/
    │   ├── rbac.yaml                 # ServiceAccount + Role + RoleBinding for CR management
    │   ├── configmap.yaml            # reference: operator creates this from AssetHub CR
    │   ├── deployment.yaml           # reference: operator creates this from AssetHub CR
    │   └── service.yaml              # reference: operator creates this from AssetHub CR
    ├── ui/
    │   ├── deployment.yaml           # reference: operator creates this from AssetHub CR
    │   └── service.yaml              # reference: operator creates this from AssetHub CR
    └── operator/
        ├── crd.yaml                  # AssetHub CRD (incl. clusterRole, environment fields)
        ├── catalogversion-crd.yaml   # CatalogVersion CRD (discovery CRs)
        ├── serviceaccount.yaml       # operator identity
        ├── role.yaml                 # namespace-scoped permissions (incl. catalogversions)
        ├── rolebinding.yaml          # binds role to service account
        ├── deployment.yaml           # operator pod
        └── sample-cr.yaml            # AssetHub CR (replicas, environment, clusterRole, etc.)
```

**Note:** The `api-server/` and `ui/` manifests (except `rbac.yaml`) are
reference files showing what the operator creates. They are not applied
directly during deployment — the operator generates these resources from
the AssetHub CR spec.
```
