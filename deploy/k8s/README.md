# Kubernetes Deployment

This directory contains Kubernetes manifests for deploying the Fast Ad Bidder in production.

## Prerequisites

- Kubernetes 1.24+
- kubectl configured with cluster access
- Sufficient cluster resources (3+ nodes recommended)
- PostgreSQL database (external or in-cluster)
- InfluxDB instance (external or in-cluster)
- TLS certificates for mTLS authentication

## Quick Start

### 1. Update Configuration

Edit the following files with your environment-specific values:

**`secret.yaml`** - Replace base64-encoded secrets:
```bash
# Generate base64-encoded secrets
echo -n "your-db-password" | base64
echo -n "your-influx-token" | base64
```

**`configmap.yaml`** - Update service endpoints:
- `DB_HOST`: PostgreSQL hostname
- `INFLUX_URL`: InfluxDB URL

**`ingress.yaml`** - Update domain:
- Replace `bidder.example.com` with your actual domain

**`kustomization.yaml`** - Update image registry:
- Replace `your-registry.io/fast-ad-bidder` with your registry

### 2. Create TLS Secret

Create a Kubernetes secret with your mTLS certificates:

```bash
kubectl create secret tls bidder-tls \
  --cert=path/to/server.crt \
  --key=path/to/server.key \
  --namespace=fast-ad-bidder

# Add CA certificate for client verification
kubectl create secret generic bidder-ca-cert \
  --from-file=ca.crt=path/to/ca.crt \
  --namespace=fast-ad-bidder
```

### 3. Deploy Using kubectl

```bash
# Create namespace first
kubectl apply -f namespace.yaml

# Apply all manifests
kubectl apply -f .

# Or use kustomize
kubectl apply -k .
```

### 4. Verify Deployment

```bash
# Check pod status
kubectl get pods -n fast-ad-bidder

# Check service
kubectl get svc -n fast-ad-bidder

# Check ingress
kubectl get ingress -n fast-ad-bidder

# View logs
kubectl logs -f -n fast-ad-bidder -l app.kubernetes.io/component=bidder

# Check HPA status
kubectl get hpa -n fast-ad-bidder
```

## Manifest Overview

| File | Description |
|------|-------------|
| `namespace.yaml` | Creates `fast-ad-bidder` namespace |
| `configmap.yaml` | Non-sensitive configuration |
| `secret.yaml` | Sensitive configuration (DB credentials, tokens) |
| `deployment.yaml` | Bidder pod deployment with 3 replicas |
| `service.yaml` | ClusterIP service exposing bidder |
| `ingress.yaml` | External ingress with mTLS passthrough |
| `hpa.yaml` | Auto-scaling from 3 to 20 pods based on CPU/memory |
| `pdb.yaml` | Ensures min 2 pods during disruptions |
| `servicemonitor.yaml` | Prometheus Operator metrics scraping |
| `kustomization.yaml` | Kustomize configuration for easy deployment |

## Resource Requirements

**Per Pod:**
- CPU Request: 500m
- CPU Limit: 2000m
- Memory Request: 512Mi
- Memory Limit: 2Gi

**Cluster Minimum (3 replicas):**
- CPU: 1.5 cores
- Memory: 1.5 GB

**Cluster Recommended (10 replicas at peak):**
- CPU: 5 cores
- Memory: 5 GB

## Auto-Scaling

The HPA is configured to:
- **Min replicas:** 3 (for HA)
- **Max replicas:** 20 (for traffic spikes)
- **Scale up:** Aggressive (100% increase every 30s if needed)
- **Scale down:** Conservative (50% decrease after 5min stabilization)
- **Triggers:** CPU >70% or Memory >80%

## High Availability

- **Pod Anti-Affinity:** Spreads pods across nodes
- **PodDisruptionBudget:** Ensures min 2 pods during voluntary disruptions
- **Rolling Updates:** Zero-downtime deployments with maxUnavailable=0
- **Health Checks:** Liveness, readiness, and startup probes configured

## Monitoring

Metrics are exposed at `/metrics` and automatically scraped by Prometheus if:
1. Using Prometheus Operator → `servicemonitor.yaml` configures scraping
2. Using standalone Prometheus → Add job to `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'bidder'
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names:
            - fast-ad-bidder
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
        regex: bidder
        action: keep
```

## Security

- **Non-root user:** Runs as UID 1000
- **Read-only root filesystem:** Prevents runtime modifications
- **Dropped capabilities:** All Linux capabilities dropped
- **Seccomp profile:** Runtime default applied
- **mTLS authentication:** Required for all incoming requests
- **Network policies:** (Add `networkpolicy.yaml` if needed)

## Troubleshooting

### Pods not starting

```bash
# Check pod events
kubectl describe pod -n fast-ad-bidder <pod-name>

# Check init container logs
kubectl logs -n fast-ad-bidder <pod-name> -c init-container

# Check resource constraints
kubectl top pods -n fast-ad-bidder
```

### Health check failures

```bash
# Test health endpoint from within cluster
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -n fast-ad-bidder \
  -- curl -vk https://bidder-service:443/health

# Check TLS certificate validity
kubectl get secret bidder-tls -n fast-ad-bidder -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -text -noout
```

### Database connection issues

```bash
# Test PostgreSQL connectivity
kubectl run -it --rm psql --image=postgres:14-alpine --restart=Never -n fast-ad-bidder \
  -- psql -h <DB_HOST> -U <DB_USER> -d <DB_NAME>

# Check connection from bidder pod
kubectl exec -it -n fast-ad-bidder <pod-name> -- /bin/sh
# Then test connection manually
```

### High latency

```bash
# Check resource usage
kubectl top pods -n fast-ad-bidder

# Check HPA status
kubectl get hpa -n fast-ad-bidder

# Force scale up
kubectl scale deployment bidder -n fast-ad-bidder --replicas=10

# Check database performance
# Review slow query logs, check connection pool settings
```

## Rolling Updates

```bash
# Update image
kubectl set image deployment/bidder bidder=your-registry.io/fast-ad-bidder:v2.0.0 -n fast-ad-bidder

# Watch rollout
kubectl rollout status deployment/bidder -n fast-ad-bidder

# Rollback if needed
kubectl rollout undo deployment/bidder -n fast-ad-bidder

# View rollout history
kubectl rollout history deployment/bidder -n fast-ad-bidder
```

## Backup and Restore

### Backup Configuration

```bash
# Backup all manifests
kubectl get all,configmap,secret,ingress,hpa,pdb -n fast-ad-bidder -o yaml > backup.yaml

# Backup specific resources
kubectl get configmap bidder-config -n fast-ad-bidder -o yaml > configmap-backup.yaml
```

### Database Backup

Ensure regular PostgreSQL backups are configured separately (e.g., using CronJob with pg_dump).

## Production Checklist

- [ ] Update all placeholder secrets in `secret.yaml`
- [ ] Configure actual DB and InfluxDB endpoints in `configmap.yaml`
- [ ] Update domain in `ingress.yaml`
- [ ] Update image registry in `kustomization.yaml`
- [ ] Create TLS secrets for mTLS
- [ ] Configure resource limits based on load testing
- [ ] Set up monitoring alerts (see `config/prometheus/alerts.yml`)
- [ ] Configure backup strategy for PostgreSQL
- [ ] Review and adjust HPA thresholds
- [ ] Set up log aggregation (e.g., ELK, Loki)
- [ ] Configure NetworkPolicy for additional security
- [ ] Test disaster recovery procedures

## Clean Up

```bash
# Delete all resources
kubectl delete -k .

# Or delete namespace (deletes everything in it)
kubectl delete namespace fast-ad-bidder
```
