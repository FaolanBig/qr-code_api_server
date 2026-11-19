# Deployment

## Docker

Build:

```bash
docker build -t qr-code-api:local .
```

Run:

```bash
docker run --rm \
  --name qr-code-api \
  --publish 127.0.0.1:8080:8080 \
  --env API_KEY="replace-with-a-random-key-at-least-16-characters" \
  qr-code-api:local
```

The container image:

- Is built in a multi-stage Docker build.
- Contains Alpine BusyBox `/bin/sh` and `curl`.
- Runs as the non-root `app` user.
- Exposes port `8080`.
- Includes a Docker health check for `/healthz`.
- Is stateless and requires no mounted volumes.

For production, run containers with read-only root filesystems and resource limits where possible.

## Kubernetes

The manifests in `deploy/kubernetes/` are suitable as a starting point:

- `configmap.yaml`: listen and timeout settings
- `secret.example.yaml`: placeholder API-key Secret
- `deployment.yaml`: two replicas, probes, resources, and security context
- `service.yaml`: internal port 80 mapped to container port 8080
- `kustomization.yaml`: apply all resources together

Create the Secret through the Kubernetes CLI or a secret manager:

```bash
kubectl -n default create secret generic qr-code-api \
  --from-literal=api-key="replace-with-a-random-key-at-least-16-characters"
```

Do not apply `secret.example.yaml` with a real cluster. Replace the placeholder before applying it manually or omit it from Kustomize.

Apply configuration and workloads:

```bash
kubectl apply -f deploy/kubernetes/configmap.yaml
kubectl apply -f deploy/kubernetes/deployment.yaml
kubectl apply -f deploy/kubernetes/service.yaml

kubectl -n default rollout status deployment/qr-code-api
kubectl -n default get pods,service
```

Or apply the configured resources:

```bash
kubectl apply -k deploy/kubernetes
```

## Image

The example Deployment references:

```text
ghcr.io/faolanbig/qr-code-api-server:latest
```

Build and push a tagged image into your registry, or change `spec.template.spec.containers[].image` to match an approved image.

## Network security

The example Service is ClusterIP-only. In production:

1. Deploy an ingress or API gateway.
2. Enforce HTTPS/TLS at the ingress.
3. Apply ingress allow-lists or network policies as appropriate.
4. Keep the API key out of source-control manifests.
5. Consider rate limiting at the gateway.

## Probe behavior

- `/healthz`: process liveness
- `/readyz`: readiness signal
- Startup probe: 12 attempts with five-second intervals
- Readiness probe: every 10 seconds
- Liveness probe: every 20 seconds

## Rotation

Rotate the API key by updating the `qr-code-api` Secret:

```bash
kubectl -n default create secret generic qr-code-api \
  --from-literal=api-key="a-new-random-key-at-least-16-characters" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl -n default rollout restart deployment/qr-code-api
```

## Scaling

The service is stateless, so replicas can scale horizontally. Generation is CPU and memory bounded. Add resource limits based on observed payload sizes and request volume.
