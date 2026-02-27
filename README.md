# external-dns-inwx-webhook

An [ExternalDNS](https://github.com/kubernetes-sigs/external-dns) webhook provider for [INWX](https://www.inwx.com/) (InterNetworX). This plugin lets ExternalDNS automatically manage DNS records hosted at INWX based on your Kubernetes Ingress, Service, and DNSEndpoint resources.

## How it works

ExternalDNS watches Kubernetes resources (Ingresses, Services, DNSEndpoints) for desired DNS records. When configured with `--provider=webhook`, it delegates record management to an external process over HTTP. This webhook plugin receives those HTTP requests and translates them into INWX API calls via XML-RPC, creating, updating, and deleting DNS records as needed.

The plugin runs as a **sidecar container** alongside ExternalDNS in the same Pod. It exposes two HTTP servers:

- **Webhook server** (`localhost:8888`) — handles ExternalDNS communication (only accessible within the Pod)
- **Metrics server** (`:8080`) — exposes `/healthz` for liveness/readiness probes and `/metrics` for Prometheus scraping

## Installation

### Container image

Pre-built multi-arch images (linux/amd64, linux/arm64) are published to GitHub Container Registry:

```
ghcr.io/orbit-online/external-dns-inwx-webhook
```

Images are tagged with semantic versions (e.g. `v1.0.0`) and `latest` on every push to `main`.

### Build from source

```bash
# requires Go 1.25+
go build -o external-dns-inwx-webhook .
```

Or with Docker:

```bash
docker build -t external-dns-inwx-webhook .
```

## Configuration

All options can be set via CLI flags or environment variables.

| Flag | Environment Variable | Default | Description |
|---|---|---|---|
| `--inwx-username` | `INWX_USERNAME` | *(required)* | INWX account username |
| `--inwx-password` | `INWX_PASSWORD` | *(required)* | INWX account password |
| `--domain-filter` | `INWX_DOMAIN_FILTER` | *(none)* | Restrict to specific domain(s); can be specified multiple times |
| `--listen-address` | `INWX_LISTEN_ADDRESS` | `localhost:8888` | Webhook endpoint listen address |
| `--metrics-listen-address` | `INWX_METRICS_LISTEN_ADDRESS` | `:8080` | Metrics/health endpoint listen address |
| `--inwx-sandbox` | `INWX_SANDBOX` | `false` | Use the INWX sandbox API for testing |
| `--tls-config` | `INWX_TLS_CONFIG` | *(none)* | Path to TLS config file |
| `--log.level` | — | `info` | Log level (`debug`, `info`, `warn`, `error`) |

## Kubernetes deployment

This webhook runs as a sidecar container alongside ExternalDNS. There are two ways to deploy it.

### Using the official ExternalDNS Helm chart (recommended)

The [ExternalDNS Helm chart](https://github.com/kubernetes-sigs/external-dns/tree/master/charts/external-dns) has built-in support for webhook providers. This is the easiest way to get started.

#### 1. Add the Helm repo

```bash
helm repo add external-dns https://kubernetes-sigs.github.io/external-dns/
helm repo update
```

#### 2. Create INWX credentials secret

```bash
kubectl create namespace external-dns

kubectl -n external-dns create secret generic inwx-credentials \
  --from-literal=INWX_USERNAME=your-username \
  --from-literal=INWX_PASSWORD=your-password
```

#### 3. Install with Helm

Create a `values.yaml`:

```yaml
provider:
  name: webhook
  webhook:
    image:
      repository: ghcr.io/orbit-online/external-dns-inwx-webhook
      tag: latest
    env:
    - name: INWX_USERNAME
      valueFrom:
        secretKeyRef:
          name: inwx-credentials
          key: INWX_USERNAME
    - name: INWX_PASSWORD
      valueFrom:
        secretKeyRef:
          name: inwx-credentials
          key: INWX_PASSWORD
    args:
    - --domain-filter=example.com
    - --log.level=debug
```

Then install:

```bash
helm install external-dns external-dns/external-dns \
  --namespace external-dns \
  -f values.yaml
```

#### 4. Verify

```bash
kubectl -n external-dns logs deployment/external-dns -c webhook
```

At startup, the webhook logs all available INWX zones — useful for verifying your domain filter configuration.

### Using raw manifests

A plain Kubernetes manifest (Deployment + RBAC) is provided in [`example/external-dns.yaml`](example/external-dns.yaml) if you prefer not to use Helm.

```bash
kubectl create namespace external-dns

kubectl -n external-dns create secret generic inwx-credentials \
  --from-literal=INWX_USERNAME=your-username \
  --from-literal=INWX_PASSWORD=your-password

kubectl -n external-dns apply -f example/external-dns.yaml
```

## Running locally

```bash
export INWX_USERNAME=your-username
export INWX_PASSWORD=your-password

# Use sandbox mode for testing
./external-dns-inwx-webhook --inwx-sandbox --domain-filter=example.com --log.level=debug
```

The webhook server will be available at `http://localhost:8888` and metrics at `http://localhost:8080`.

## Key behaviors

- **Upsert semantics** — Record creates are idempotent. If an identical record already exists, the create is skipped. If a record with the same name and type but different content exists, it is updated rather than duplicated.
- **Zone caching** — The INWX zone list is cached for 5 minutes to reduce API calls.
- **Pagination** — Zone listing is paginated (100 per page) to support accounts with many domains.
- **Apex domain handling** — Correctly handles ExternalDNS ownership TXT records for apex domains, including edge cases around dot-boundary and hyphen-boundary matching.

## Development

### Running tests

```bash
go test ./provider
```

Tests use an in-memory mock of the INWX API client, covering the full lifecycle of record creation, update, deletion, zone matching, and various edge cases.

### Project structure

```
├── main.go                     # Entrypoint, HTTP server setup
├── provider/
│   ├── inwx.go                 # Core provider logic
│   ├── client_wrapper.go       # INWX API client wrapper with zone caching
│   └── mock_client_wrapper.go  # In-memory mock for tests
├── example/
│   └── external-dns.yaml       # Sample Kubernetes deployment manifest
└── Dockerfile                  # Multi-stage build (Alpine-based)
```

### Dependencies

| Library | Purpose |
|---|---|
| [goinwx](https://github.com/nrdcg/goinwx) | INWX XML-RPC API client |
| [external-dns](https://github.com/kubernetes-sigs/external-dns) | Webhook provider API types and server |
| [kingpin](https://github.com/alecthomas/kingpin) | CLI flag and environment variable parsing |
| [prometheus/client_golang](https://github.com/prometheus/client_golang) | Prometheus metrics |
| [prometheus/exporter-toolkit](https://github.com/prometheus/exporter-toolkit) | TLS-capable HTTP server |

## License

MIT — see [LICENSE](LICENSE) for details.
