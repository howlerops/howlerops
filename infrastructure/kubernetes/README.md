# Howlerops - Kubernetes Deployment Guide

Complete guide for deploying Howlerops to Kubernetes clusters in production.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [File Overview](#file-overview)
- [Deployment Steps](#deployment-steps)
- [Configuration](#configuration)
- [Monitoring](#monitoring)
- [Troubleshooting](#troubleshooting)
- [Rollback](#rollback)

## Prerequisites

### Required Tools

```bash
# kubectl (Kubernetes CLI)
kubectl version --client

# helm (Package manager for Kubernetes)
helm version

# Docker (for building images)
docker --version

# Optional but recommended
k9s  # Terminal UI for Kubernetes
```

### Cluster Requirements

- **Kubernetes version**: 1.24+
- **Node resources**: Minimum 2 nodes with 2 CPU / 4GB RAM each
- **Network plugin**: Must support Network Policies (Calico, Cilium, Weave)
- **Ingress controller**: nginx-ingress or equivalent
- **Cert-manager**: For automatic TLS certificate management
- **Metrics server**: For HPA (Horizontal Pod Autoscaler)

### Install Prerequisites

```bash
# Install metrics-server
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

# Install nginx-ingress
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm install ingress-nginx ingress-nginx/ingress-nginx \
  --namespace ingress-nginx \
  --create-namespace

# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Verify installations
kubectl get pods -n kube-system | grep metrics-server
kubectl get pods -n ingress-nginx
kubectl get pods -n cert-manager
```

## Quick Start

```bash
# 1. Create namespace
kubectl apply -f namespace.yaml

# 2. Create secrets (copy template first)
cp secrets.yaml.template secrets.yaml
# Edit secrets.yaml with your values (base64 encoded)
kubectl apply -f secrets.yaml

# 3. Apply configuration
kubectl apply -f configmap.yaml

# 4. Deploy applications
kubectl apply -f backend-deployment.yaml
kubectl apply -f frontend-deployment.yaml

# 5. Create services
kubectl apply -f service.yaml

# 6. Setup auto-scaling
kubectl apply -f hpa.yaml

# 7. Apply network policies
kubectl apply -f network-policy.yaml

# 8. Setup ingress (after updating domains)
kubectl apply -f ingress.yaml

# 9. Verify deployment
kubectl get pods -n sql-studio
kubectl get svc -n sql-studio
kubectl get ingress -n sql-studio
```

## File Overview

### Core Manifests

| File | Purpose | Priority |
|------|---------|----------|
| `namespace.yaml` | Namespace with resource quotas | **Required** |
| `secrets.yaml.template` | Secret templates (DO NOT commit) | **Required** |
| `configmap.yaml` | Non-sensitive configuration | **Required** |
| `backend-deployment.yaml` | Backend API deployment | **Required** |
| `frontend-deployment.yaml` | Frontend web deployment | **Required** |
| `service.yaml` | ClusterIP services | **Required** |
| `ingress.yaml` | External access & TLS | **Required** |
| `hpa.yaml` | Horizontal Pod Autoscaler | Recommended |
| `network-policy.yaml` | Network security policies | Recommended |

### Deployment Order

1. **Namespace** → Resource boundaries
2. **Secrets** → Sensitive configuration
3. **ConfigMaps** → Non-sensitive configuration
4. **Deployments** → Application pods
5. **Services** → Internal networking
6. **Ingress** → External access
7. **HPA** → Auto-scaling
8. **Network Policies** → Security

## Deployment Steps

### Step 1: Configure Secrets

```bash
# Copy the template
cp secrets.yaml.template secrets.yaml

# Generate secrets
export TURSO_URL="libsql://your-database.turso.io"
export TURSO_TOKEN="your-turso-token"
export JWT_SECRET=$(openssl rand -base64 64)
export RESEND_KEY="re_your_resend_key"
export FROM_EMAIL="noreply@sql-studio.app"

# Base64 encode and update secrets.yaml
echo -n "$TURSO_URL" | base64
echo -n "$TURSO_TOKEN" | base64
echo -n "$JWT_SECRET" | base64
echo -n "$RESEND_KEY" | base64
echo -n "$FROM_EMAIL" | base64

# Or create directly with kubectl
kubectl create secret generic sql-studio-secrets \
  --namespace=sql-studio \
  --from-literal=turso-url="$TURSO_URL" \
  --from-literal=turso-auth-token="$TURSO_TOKEN" \
  --from-literal=jwt-secret="$JWT_SECRET" \
  --from-literal=resend-api-key="$RESEND_KEY" \
  --from-literal=resend-from-email="$FROM_EMAIL"
```

### Step 2: Update Configuration

Edit `configmap.yaml` and `ingress.yaml`:

```bash
# Update domains in ingress.yaml
sed -i 's/sql-studio.app/your-domain.com/g' ingress.yaml

# Update backend API URL in configmap.yaml
sed -i 's/api.sql-studio.app/api.your-domain.com/g' configmap.yaml
```

### Step 3: Build and Push Images

```bash
# Backend
cd backend-go
docker build -t gcr.io/YOUR_PROJECT_ID/sql-studio-backend:v1.0.0 .
docker push gcr.io/YOUR_PROJECT_ID/sql-studio-backend:v1.0.0

# Frontend
cd ../frontend
docker build -t gcr.io/YOUR_PROJECT_ID/sql-studio-frontend:v1.0.0 .
docker push gcr.io/YOUR_PROJECT_ID/sql-studio-frontend:v1.0.0

# Update image references in deployment files
sed -i 's/YOUR_PROJECT_ID/your-actual-project-id/g' backend-deployment.yaml
sed -i 's/YOUR_PROJECT_ID/your-actual-project-id/g' frontend-deployment.yaml
```

### Step 4: Deploy to Cluster

```bash
# Apply all manifests in order
kubectl apply -f namespace.yaml
kubectl apply -f secrets.yaml
kubectl apply -f configmap.yaml
kubectl apply -f backend-deployment.yaml
kubectl apply -f frontend-deployment.yaml
kubectl apply -f service.yaml
kubectl apply -f hpa.yaml
kubectl apply -f network-policy.yaml
kubectl apply -f ingress.yaml

# Wait for pods to be ready
kubectl wait --for=condition=ready pod \
  -l app=sql-studio \
  -n sql-studio \
  --timeout=300s

# Check status
kubectl get all -n sql-studio
```

### Step 5: Configure DNS

Point your domains to the ingress load balancer:

```bash
# Get ingress IP
kubectl get ingress sql-studio -n sql-studio

# Create DNS A records:
# sql-studio.app → INGRESS_IP
# api.sql-studio.app → INGRESS_IP
# www.sql-studio.app → INGRESS_IP
```

### Step 6: Verify Deployment

```bash
# Check pods are running
kubectl get pods -n sql-studio

# Check services
kubectl get svc -n sql-studio

# Check ingress
kubectl get ingress -n sql-studio

# Test health endpoints
curl -k https://api.sql-studio.app/health
curl -k https://sql-studio.app/health

# View logs
kubectl logs -f deployment/sql-studio-backend -n sql-studio
kubectl logs -f deployment/sql-studio-frontend -n sql-studio

# Check HPA status
kubectl get hpa -n sql-studio

# Check resource usage
kubectl top pods -n sql-studio
```

## Configuration

### Environment Variables

Backend configuration is in `configmap.yaml`:

```yaml
# Server ports
SERVER_HTTP_PORT: "8500"
SERVER_GRPC_PORT: "9500"
METRICS_PORT: "9100"

# Environment
ENVIRONMENT: "production"

# Logging
LOG_LEVEL: "info"
LOG_FORMAT: "json"

# Database connection pool
DATABASE_MAX_OPEN_CONNS: "25"
DATABASE_MAX_IDLE_CONNS: "10"

# JWT expiration
JWT_EXPIRATION: "24h"
JWT_REFRESH_EXPIRATION: "168h"
```

### Resource Limits

Adjust resources in deployment files based on your needs:

```yaml
resources:
  requests:
    cpu: 100m      # Minimum guaranteed
    memory: 256Mi
  limits:
    cpu: 500m      # Maximum allowed
    memory: 1Gi
```

### Auto-scaling

Modify HPA settings in `hpa.yaml`:

```yaml
minReplicas: 2    # Minimum pods
maxReplicas: 10   # Maximum pods

metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70  # Target CPU %
```

## Monitoring

### View Logs

```bash
# Stream backend logs
kubectl logs -f deployment/sql-studio-backend -n sql-studio

# Stream frontend logs
kubectl logs -f deployment/sql-studio-frontend -n sql-studio

# View last 100 lines
kubectl logs --tail=100 deployment/sql-studio-backend -n sql-studio

# View logs from all pods
kubectl logs -l app=sql-studio -n sql-studio --all-containers=true

# View previous container logs (after restart)
kubectl logs -p deployment/sql-studio-backend -n sql-studio
```

### Check Resource Usage

```bash
# Pod resource usage
kubectl top pods -n sql-studio

# Node resource usage
kubectl top nodes

# Detailed pod info
kubectl describe pod -l app=sql-studio -n sql-studio

# Resource quotas
kubectl describe resourcequota -n sql-studio
```

### Monitor Auto-scaling

```bash
# Watch HPA status
kubectl get hpa -n sql-studio --watch

# HPA events
kubectl describe hpa sql-studio-backend-hpa -n sql-studio

# Current metrics
kubectl get hpa sql-studio-backend-hpa -n sql-studio -o yaml
```

## Troubleshooting

### Pods Not Starting

```bash
# Check pod status
kubectl get pods -n sql-studio

# View pod events
kubectl describe pod POD_NAME -n sql-studio

# Check logs
kubectl logs POD_NAME -n sql-studio

# Common issues:
# - Image pull errors → Check image exists and registry auth
# - CrashLoopBackOff → Check application logs
# - Pending → Check resource availability
```

### Connection Issues

```bash
# Test service connectivity
kubectl run test-pod --image=busybox -n sql-studio -- sleep 3600
kubectl exec -it test-pod -n sql-studio -- wget -O- http://sql-studio-backend:8500/health

# Check network policies
kubectl get networkpolicies -n sql-studio
kubectl describe networkpolicy backend-ingress -n sql-studio

# Test DNS resolution
kubectl exec -it test-pod -n sql-studio -- nslookup sql-studio-backend
```

### Ingress Issues

```bash
# Check ingress status
kubectl describe ingress sql-studio -n sql-studio

# Check ingress controller logs
kubectl logs -n ingress-nginx -l app.kubernetes.io/component=controller

# Verify cert-manager
kubectl get certificate -n sql-studio
kubectl describe certificate sql-studio-tls -n sql-studio
```

### Performance Issues

```bash
# Check if HPA is working
kubectl get hpa -n sql-studio

# Check resource usage
kubectl top pods -n sql-studio

# Check if hitting resource limits
kubectl describe pod POD_NAME -n sql-studio | grep -A 5 "Limits"

# Check for throttling
kubectl describe pod POD_NAME -n sql-studio | grep -i throttl
```

## Rollback

### Rollback Deployment

```bash
# View revision history
kubectl rollout history deployment/sql-studio-backend -n sql-studio

# Rollback to previous version
kubectl rollout undo deployment/sql-studio-backend -n sql-studio

# Rollback to specific revision
kubectl rollout undo deployment/sql-studio-backend -n sql-studio --to-revision=2

# Check rollback status
kubectl rollout status deployment/sql-studio-backend -n sql-studio
```

### Emergency Procedures

```bash
# Scale down to zero (emergency stop)
kubectl scale deployment sql-studio-backend --replicas=0 -n sql-studio

# Scale back up
kubectl scale deployment sql-studio-backend --replicas=2 -n sql-studio

# Restart all pods
kubectl rollout restart deployment/sql-studio-backend -n sql-studio
kubectl rollout restart deployment/sql-studio-frontend -n sql-studio

# Delete and recreate (last resort)
kubectl delete deployment sql-studio-backend -n sql-studio
kubectl apply -f backend-deployment.yaml
```

## Useful Commands

```bash
# Get everything in namespace
kubectl get all -n sql-studio

# Watch pod status
kubectl get pods -n sql-studio --watch

# Port forward for local testing
kubectl port-forward svc/sql-studio-backend 8500:8500 -n sql-studio

# Execute command in pod
kubectl exec -it POD_NAME -n sql-studio -- /bin/sh

# Copy files from pod
kubectl cp sql-studio/POD_NAME:/app/logs/app.log ./app.log

# View cluster events
kubectl get events -n sql-studio --sort-by='.lastTimestamp'

# Dry run (test without applying)
kubectl apply -f backend-deployment.yaml --dry-run=client

# Validate YAML
kubectl apply -f backend-deployment.yaml --dry-run=server --validate=true
```

## Security Best Practices

1. **Use secrets for sensitive data** - Never commit secrets to git
2. **Enable network policies** - Isolate pod communication
3. **Run as non-root** - All containers use non-root users
4. **Read-only filesystem** - Containers use read-only root filesystem
5. **Resource limits** - Set CPU/memory limits on all containers
6. **RBAC** - Use service accounts with minimal permissions
7. **Security contexts** - Drop unnecessary capabilities
8. **Regular updates** - Keep images and dependencies updated

## Additional Resources

- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Kubectl Cheat Sheet](https://kubernetes.io/docs/reference/kubectl/cheatsheet/)
- [Howlerops Docs](../docs/infrastructure/)
- [Deployment Guide](../docs/infrastructure/DEPLOYMENT_GUIDE.md)
- [Runbook](../docs/infrastructure/RUNBOOK.md)

## Support

For issues or questions:
- GitHub Issues: https://github.com/your-org/sql-studio/issues
- Documentation: https://docs.sql-studio.app
- Email: support@sql-studio.app
