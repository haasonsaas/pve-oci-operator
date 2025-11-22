# pve-oci-operator

A lightweight Go controller that keeps Proxmox VE LXC containers in sync with declarative service specs and OCI image tags/digests. It resolves desired images from a registry, compares them with the currently running CTs, and performs safe rollouts (with optional rollback) via the Proxmox `pct` CLI.

## Features

- Watches YAML service specs for desired node, CTID, resources, mounts, networks, and rollout policy
- Resolves image tags to immutable digests through the OCI registry API (GHCR ready)
- Tracks actual container state and digests via a simple file-backed store
- Supports recreate rollouts with health checks and configurable auto-rollback
- Provides a ticker-based reconcile loop with dry-run support to preview actions

## Requirements

- Go 1.24+
- Proxmox VE host accessible with the `pct` CLI
- OCI registry credentials (if the registry is private)

## Configuration

Create a `config.yaml`:

```yaml
registry:
  username: $GHCR_USER
  password: $GHCR_PAT
pve:
  mode: cli
  pctPath: /usr/sbin/pct
  statePath: /var/lib/pve-oci-operator/state
  dryRun: false
runner:
  servicesPath: ./services
  interval: 10s
```

## Service Specs

Each service is a YAML file inside `services/`:

```yaml
apiVersion: pve.evalops/v1
kind: Service
metadata:
  name: composer-web
spec:
  node: hephaestus-2
  ctid: 160
  image: ghcr.io/evalops/composer-web
  tag: main
  pullPolicy: digest
  rollout:
    strategy: recreate
    autoRollback: true
```

## Running

```bash
go build ./cmd/pve-oci-operator
./pve-oci-operator --config config.yaml
```

Place new or updated service spec files into the configured directory; the reconcile loop will detect the changes and roll out the specified images.

## Development

```bash
go test ./...
```

Pull requests and issues are welcome.
