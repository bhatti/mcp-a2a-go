# Kubernetes Deployment Guide

Production-ready Kubernetes manifests for MCP & A2A servers with observability, high availability, and auto-scaling.

## Quick Start

```bash
# 1. Update secrets in k8s/base/secret.yaml
kubectl apply -k k8s/base

# 2. Wait for services to be ready
kubectl wait --for=condition=ready pod -l app=postgres -n mcp-a2a --timeout=300s
kubectl wait --for=condition=ready pod -l app=mcp-server -n mcp-a2a --timeout=300s

# 3. Get ingress IP
kubectl get ingress -n mcp-a2a
```

## Prerequisites

- Kubernetes 1.25+
- kubectl configured
- Ingress controller (nginx recommended)
- Cert-manager for TLS
- Storage class configured

## Architecture

```
┌─────────────────────────────────────────┐
│            Ingress (TLS)                │
│  mcp.example.com | a2a.example.com      │
└────────┬───────────────┬────────────────┘
         │               │
    ┌────▼────┐     ┌────▼────┐
    │MCP Pods │     │A2A Pods │
    │ (3-10)  │     │ (3-10)  │
    └────┬────┘     └────┬────┘
         │               │
    ┌────▼───────────────▼────┐
    │   Redis  │  PostgreSQL   │
    │  (Cache) │  (+ pgvector) │
    └──────────┴───────────────┘
```

## Components

### Deployments

- **mcp-server**: 3-10 replicas (HPA)
- **a2a-server**: 3-10 replicas (HPA)
- **postgres**: 1 replica (StatefulSet recommended for prod)
- **redis**: 1 replica (Redis Cluster for prod)

### Storage

- PostgreSQL: 10Gi PVC
- Redis: 5Gi PVC
- Prometheus: 20Gi PVC
- Grafana: 5Gi PVC

### Auto-Scaling

Both MCP and A2A servers have HPA configured:
- Min replicas: 3
- Max replicas: 10
- CPU target: 70%
- Memory target: 80%

## Configuration

### Secrets

Update `k8s/base/secret.yaml` with:
- Database credentials
- JWT keys (generate with `openssl genrsa`)
- OpenAI API key
- LangFuse keys

### ConfigMaps

- `mcp-server-config`: MCP server configuration
- `a2a-server-config`: A2A server configuration
- `postgres-init`: Database initialization script

### Environment-Specific Overlays

```bash
# Development
kubectl apply -k k8s/overlays/dev

# Production
kubectl apply -k k8s/overlays/prod
```

## Deployment Steps

### 1. Prepare Secrets

```bash
# Generate JWT keys
openssl genrsa -out private.pem 2048
openssl rsa -in private.pem -pubout -out public.pem

# Edit secret.yaml with real values
vim k8s/base/secret.yaml
```

### 2. Build and Push Images

```bash
# Build MCP server
cd mcp-server
docker build -t your-registry/mcp-server:v1.0.0 .
docker push your-registry/mcp-server:v1.0.0

# Build A2A server
cd ../a2a-server
docker build -t your-registry/a2a-server:v1.0.0 .
docker push your-registry/a2a-server:v1.0.0

# Update image tags in deployment YAMLs
```

### 3. Deploy

```bash
# Apply all manifests
kubectl apply -k k8s/base

# Check status
kubectl get pods -n mcp-a2a
kubectl get svc -n mcp-a2a
kubectl get ing -n mcp-a2a
```

### 4. Verify

```bash
# Check MCP server
kubectl port-forward svc/mcp-server-service 8080:8080 -n mcp-a2a
curl http://localhost:8080/health

# Check A2A server
kubectl port-forward svc/a2a-server-service 8081:8081 -n mcp-a2a
curl http://localhost:8081/health
```

## Monitoring

### Logs

```bash
# MCP server logs
kubectl logs -f deployment/mcp-server -n mcp-a2a

# A2A server logs
kubectl logs -f deployment/a2a-server -n mcp-a2a

# Database logs
kubectl logs -f deployment/postgres -n mcp-a2a
```

### Metrics

```bash
# Get HPA status
kubectl get hpa -n mcp-a2a

# Top pods
kubectl top pods -n mcp-a2a

# Describe pod
kubectl describe pod <pod-name> -n mcp-a2a
```

## Scaling

### Manual Scaling

```bash
# Scale MCP server
kubectl scale deployment mcp-server --replicas=5 -n mcp-a2a

# Scale A2A server
kubectl scale deployment a2a-server --replicas=5 -n mcp-a2a
```

### Auto-Scaling (HPA)

HPA automatically scales based on CPU/memory:
```bash
# View HPA status
kubectl get hpa -n mcp-a2a

# Edit HPA
kubectl edit hpa mcp-server-hpa -n mcp-a2a
```

## Backup & Recovery

### PostgreSQL Backup

```bash
# Create backup
kubectl exec -it deployment/postgres -n mcp-a2a -- pg_dump -U mcp_user mcp_db > backup.sql

# Restore
kubectl exec -i deployment/postgres -n mcp-a2a -- psql -U mcp_user mcp_db < backup.sql
```

### Redis Backup

```bash
# Create backup
kubectl exec -it deployment/redis -n mcp-a2a -- redis-cli SAVE
kubectl cp mcp-a2a/redis-pod:/data/dump.rdb ./redis-backup.rdb
```

## Troubleshooting

### Pod Not Starting

```bash
kubectl describe pod <pod-name> -n mcp-a2a
kubectl logs <pod-name> -n mcp-a2a
```

### Database Connection Issues

```bash
# Check postgres is ready
kubectl get pods -l app=postgres -n mcp-a2a

# Test connection
kubectl exec -it deployment/postgres -n mcp-a2a -- psql -U mcp_user -d mcp_db -c "SELECT 1"
```

### Ingress Not Working

```bash
# Check ingress
kubectl describe ingress mcp-a2a-ingress -n mcp-a2a

# Check ingress controller logs
kubectl logs -n ingress-nginx deployment/ingress-nginx-controller
```

## Production Considerations

### High Availability

- Use StatefulSet for PostgreSQL with replication
- Deploy Redis Cluster (3+ nodes)
- Multi-AZ deployment for cloud providers
- Pod anti-affinity rules

### Security

- Enable Network Policies
- Use Pod Security Policies/Standards
- Rotate secrets regularly
- Enable audit logging
- Use external secret management (Vault, AWS Secrets Manager)

### Performance

- Adjust resource requests/limits based on load
- Use node affinity for database pods
- Enable persistent connections
- Configure connection pooling

### Cost Optimization

- Set appropriate resource limits
- Use HPA to scale down during low usage
- Use spot/preemptible instances for non-critical workloads
- Monitor and optimize PVC sizes

## Cleanup

```bash
# Delete everything
kubectl delete -k k8s/base

# Delete namespace (and all resources)
kubectl delete namespace mcp-a2a
```
