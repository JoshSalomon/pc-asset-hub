---
id: "mem_cbea494d"
topic: "Deploying to kind with podman - image loading workaround"
tags:
  - kubernetes
  - kind
  - podman
  - docker
  - deployment
  - containers
phase: 0
difficulty: 0.7
created_at: "2026-02-16T12:46:23.261893+00:00"
created_session: 3
---
## Problem

`kind load docker-image` does not work with the podman provider. It fails with `ERROR: image: "..." not present locally` even when the image exists in podman's store.

Two separate issues must be solved:

### Issue 1: Engine detection

`command -v docker` returns success if the docker binary is installed, even when the Docker daemon is not running. Use `docker info` / `podman info` to verify the engine is actually functional:

```bash
if command -v docker &>/dev/null && docker info &>/dev/null; then
    ENGINE="docker"
elif command -v podman &>/dev/null && podman info &>/dev/null; then
    ENGINE="podman"
fi
```

For Makefiles:
```makefile
CONTAINER_ENGINE ?= $(shell docker info >/dev/null 2>&1 && echo docker || echo podman)
```

### Issue 2: Loading images into kind

`kind load docker-image` is broken with podman (see https://github.com/kubernetes-sigs/kind/issues/3105). Workaround: use `podman save` + `kind load image-archive`.

Additionally, podman tags images with a `localhost/` prefix (e.g., `localhost/assethub/api-server:latest`), but Kubernetes manifests reference them without the prefix (`assethub/api-server:latest`). When `kind load image-archive` imports the tar, it preserves the image name from the archive. If the archive contains `localhost/assethub/...`, the K8s pods will fail with `ErrImageNeverPull` because they look for `assethub/...` (no prefix).

**Solution**: Retag with `docker.io/` prefix before saving, so the archive contains the unprefixed name that Kubernetes expects:

```bash
for img in assethub/api-server assethub/ui assethub/operator; do
    podman tag "localhost/$img:latest" "docker.io/$img:latest"
    podman save "docker.io/$img:latest" -o "/tmp/$img.tar"
    kind load image-archive "/tmp/$img.tar" --name "$CLUSTER_NAME"
    rm -f "/tmp/$img.tar"
done
```

### Issue 3: KIND_EXPERIMENTAL_PROVIDER

When using podman, set `export KIND_EXPERIMENTAL_PROVIDER=podman` before any `kind` command.

### K8s manifest requirement

All deployment manifests must use `imagePullPolicy: Never` for locally loaded images. This is already set in the project's manifests under `deploy/k8s/`.

### Project-specific files

- Deploy script: `scripts/kind-deploy.sh` (handles all of the above automatically)
- Kind config: `deploy/kind/kind-config.yaml` (ports 30080, 30000)
- Makefile targets: `kind-create`, `kind-load`, `kind-deploy-all`, `kind-delete`

