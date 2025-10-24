# SQL Studio - Production Deployment Guide

Complete step-by-step guide for deploying SQL Studio to production infrastructure.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Initial Setup](#initial-setup)
- [Kubernetes Deployment](#kubernetes-deployment)
- [DNS and SSL Configuration](#dns-and-ssl-configuration)
- [CDN Setup](#cdn-setup)
- [Monitoring](#monitoring)
- [Verification](#verification)
- [Troubleshooting](#troubleshooting)

## Overview

SQL Studio production deployment architecture:

```
Internet
   |
   ├─> Cloudflare CDN (Edge caching, DDoS protection)
   |      |
   |      ├─> Static Assets (cached 1 year)
   |      └─> Dynamic Content (no cache)
   |
   └─> nginx Ingress Controller (TLS termination, routing)
          |
          ├─> Frontend Service (2 replicas)
          |      └─> nginx pods (React SPA)
          |
          └─> Backend Service (2-10 replicas, auto-scaling)
                 └─> Go API pods
                        └─> Turso Database (libSQL, global replicas)
```

### Technology Stack

- **Container Orchestration**: Kubernetes 1.24+
- **Ingress**: nginx Ingress Controller
- **Certificate Management**: cert-manager with Let's Encrypt
- **CDN**: Cloudflare
- **Database**: Turso (libSQL) with global replication
- **Monitoring**: Prometheus + Grafana
- **CI/CD**: GitHub Actions

## Prerequisites

### Required Tools

```bash
# kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
kubectl version --client

# helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
helm version

# gcloud CLI (for GKE)
curl https://sdk.cloud.google.com | bash
gcloud init

# Optional but recommended
brew install k9s      # Terminal UI for Kubernetes
brew install kubectx  # Easy context switching
```

### Required Accounts

- **Cloud Provider**: GCP, AWS, or Azure account
- **Domain**: Registered domain (e.g., sql-studio.app)
- **Cloudflare**: Free or Pro account
- **Turso**: Account with production database
- **GitHub**: For CI/CD
- **Resend**: For email service (optional)

### Required Information

Before starting, gather:

- [ ] Cloud project ID
- [ ] Domain name
- [ ] Turso database URL and auth token
- [ ] JWT secret (64+ characters)
- [ ] Email service credentials
- [ ] Docker registry credentials

## Initial Setup

### Step 1: Create Kubernetes Cluster

#### GCP (GKE)

```bash
# Set variables
export PROJECT_ID="your-project-id"
export CLUSTER_NAME="sql-studio-cluster"
export REGION="us-central1"
export ZONE="us-central1-a"

# Create cluster
gcloud container clusters create $CLUSTER_NAME \
  --project=$PROJECT_ID \
  --region=$REGION \
  --node-locations=$ZONE \
  --machine-type=e2-medium \
  --num-nodes=2 \
  --enable-autoscaling \
  --min-nodes=2 \
  --max-nodes=10 \
  --enable-autorepair \
  --enable-autoupgrade \
  --enable-ip-alias \
  --network="default" \
  --subnetwork="default" \
  --enable-stackdriver-kubernetes \
  --addons=HorizontalPodAutoscaling,HttpLoadBalancing,GcePersistentDiskCsiDriver

# Get credentials
gcloud container clusters get-credentials $CLUSTER_NAME \
  --region=$REGION \
  --project=$PROJECT_ID

# Verify connection
kubectl get nodes
```

#### AWS (EKS)

```bash
# Install eksctl
brew install eksctl

# Create cluster
eksctl create cluster \
  --name=sql-studio-cluster \
  --region=us-east-1 \
  --nodegroup-name=standard-workers \
  --node-type=t3.medium \
  --nodes=2 \
  --nodes-min=2 \
  --nodes-max=10 \
  --managed

# Get credentials
aws eks update-kubeconfig --name sql-studio-cluster --region us-east-1
```

### Step 2: Install Cluster Prerequisites

```bash
# Create namespace
kubectl apply -f infrastructure/kubernetes/namespace.yaml

# Install metrics-server
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

# Install nginx-ingress
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo update

helm install ingress-nginx ingress-nginx/ingress-nginx \
  --namespace ingress-nginx \
  --create-namespace \
  --set controller.metrics.enabled=true \
  --set controller.service.type=LoadBalancer

# Wait for load balancer
kubectl get service ingress-nginx-controller -n ingress-nginx --watch

# Get load balancer IP
export INGRESS_IP=$(kubectl get service ingress-nginx-controller \
  -n ingress-nginx \
  -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

echo "Ingress IP: $INGRESS_IP"

# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Wait for cert-manager
kubectl wait --for=condition=available --timeout=300s \
  deployment/cert-manager -n cert-manager
```

### Step 3: Configure DNS

Point your domain to the ingress load balancer:

```bash
# Create DNS A records:
# sql-studio.app → $INGRESS_IP
# api.sql-studio.app → $INGRESS_IP
# www.sql-studio.app → $INGRESS_IP
```

**Cloudflare DNS Setup:**

1. Log in to Cloudflare Dashboard
2. Select your domain
3. Go to DNS settings
4. Add A records:
   - Type: A, Name: @, Content: $INGRESS_IP, Proxy: On
   - Type: A, Name: api, Content: $INGRESS_IP, Proxy: On
   - Type: CNAME, Name: www, Content: sql-studio.app, Proxy: On

Wait for DNS propagation (1-5 minutes with Cloudflare):

```bash
# Test DNS resolution
dig sql-studio.app
dig api.sql-studio.app
```

### Step 4: Configure Secrets

```bash
# Generate strong JWT secret
export JWT_SECRET=$(openssl rand -base64 64)

# Create Turso database if not exists
turso db create sql-studio-production --location ord

# Get Turso credentials
export TURSO_URL=$(turso db show sql-studio-production --url)
export TURSO_TOKEN=$(turso db tokens create sql-studio-production --expiration 90d)

# Create Kubernetes secrets
kubectl create secret generic sql-studio-secrets \
  --namespace=sql-studio \
  --from-literal=turso-url="$TURSO_URL" \
  --from-literal=turso-auth-token="$TURSO_TOKEN" \
  --from-literal=jwt-secret="$JWT_SECRET" \
  --from-literal=resend-api-key="${RESEND_API_KEY:-}" \
  --from-literal=resend-from-email="${RESEND_FROM_EMAIL:-noreply@sql-studio.app}"

# Verify secrets
kubectl get secrets -n sql-studio
kubectl describe secret sql-studio-secrets -n sql-studio
```

## Kubernetes Deployment

### Step 5: Update Configuration

```bash
# Clone or navigate to project
cd sql-studio

# Update image references in deployments
export PROJECT_ID="your-project-id"

sed -i "s/YOUR_PROJECT_ID/$PROJECT_ID/g" infrastructure/kubernetes/backend-deployment.yaml
sed -i "s/YOUR_PROJECT_ID/$PROJECT_ID/g" infrastructure/kubernetes/frontend-deployment.yaml

# Update domain in ingress
sed -i 's/sql-studio.app/your-domain.com/g' infrastructure/kubernetes/ingress.yaml
sed -i 's/sql-studio.app/your-domain.com/g' infrastructure/kubernetes/configmap.yaml
```

### Step 6: Build and Push Images

```bash
# Configure Docker for GCR
gcloud auth configure-docker gcr.io

# Build backend
cd backend-go
docker build \
  -t gcr.io/$PROJECT_ID/sql-studio-backend:v1.0.0 \
  -f ../infrastructure/docker/backend.Dockerfile \
  --build-arg VERSION=1.0.0 \
  --build-arg BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  --build-arg GIT_COMMIT=$(git rev-parse --short HEAD) \
  .

docker push gcr.io/$PROJECT_ID/sql-studio-backend:v1.0.0
cd ..

# Build frontend
cd frontend
docker build \
  -t gcr.io/$PROJECT_ID/sql-studio-frontend:v1.0.0 \
  -f ../infrastructure/docker/frontend.Dockerfile \
  .

docker push gcr.io/$PROJECT_ID/sql-studio-frontend:v1.0.0
cd ..
```

### Step 7: Deploy Application

```bash
# Apply configurations
kubectl apply -f infrastructure/kubernetes/configmap.yaml
kubectl apply -f infrastructure/kubernetes/backend-deployment.yaml
kubectl apply -f infrastructure/kubernetes/frontend-deployment.yaml
kubectl apply -f infrastructure/kubernetes/service.yaml
kubectl apply -f infrastructure/kubernetes/hpa.yaml
kubectl apply -f infrastructure/kubernetes/network-policy.yaml

# Wait for pods to be ready
kubectl wait --for=condition=ready pod \
  -l app=sql-studio \
  -n sql-studio \
  --timeout=300s

# Check status
kubectl get all -n sql-studio
```

### Step 8: Run Database Migrations

```bash
# Update migration job image
sed -i "s/YOUR_PROJECT_ID/$PROJECT_ID/g" infrastructure/database/migration-runner.yaml

# Run migrations
kubectl apply -f infrastructure/database/migration-runner.yaml

# Watch migration progress
kubectl logs -f job/sql-studio-db-migrate -n sql-studio

# Verify migration completed
kubectl get job sql-studio-db-migrate -n sql-studio
```

## DNS and SSL Configuration

### Step 9: Setup SSL Certificates

```bash
# Update email in certificate configuration
sed -i 's/admin@sql-studio.app/your-email@domain.com/g' infrastructure/security/ssl-certificates.yaml

# Apply cert-manager configuration
kubectl apply -f infrastructure/security/ssl-certificates.yaml

# Wait for certificates to be issued
kubectl wait --for=condition=ready \
  certificate/sql-studio-cert \
  -n sql-studio \
  --timeout=300s

# Check certificate status
kubectl get certificate -n sql-studio
kubectl describe certificate sql-studio-cert -n sql-studio

# Verify certificate secret created
kubectl get secret sql-studio-tls -n sql-studio
```

### Step 10: Deploy Ingress

```bash
# Apply ingress configuration
kubectl apply -f infrastructure/kubernetes/ingress.yaml

# Check ingress status
kubectl get ingress -n sql-studio
kubectl describe ingress sql-studio -n sql-studio

# Test HTTPS
curl -I https://sql-studio.app
curl -I https://api.sql-studio.app/health
```

## CDN Setup

### Step 11: Configure Cloudflare

1. **Enable Proxy (orange cloud)** for all DNS records
2. **SSL/TLS Settings**:
   - Mode: Full (strict)
   - Always Use HTTPS: On
   - Automatic HTTPS Rewrites: On
   - Minimum TLS Version: 1.2

3. **Caching**:
   - Browser Cache TTL: 4 hours
   - Always Online: On

4. **Speed**:
   - Auto Minify: CSS, JS, HTML
   - Brotli: On
   - Early Hints: On
   - HTTP/2: On
   - HTTP/3 (with QUIC): On

5. **Security**:
   - Security Level: Medium
   - Bot Fight Mode: On
   - Challenge Passage: 30 minutes

6. **Page Rules** (see `infrastructure/cdn/cloudflare-config.yaml`):
   - API endpoints: Bypass cache
   - Static assets: Cache everything, 1 year
   - HTML: No cache

## Monitoring

### Step 12: Setup Monitoring (Optional)

```bash
# Install Prometheus
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace

# Install Grafana dashboards
kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80

# Access Grafana at http://localhost:3000
# Default credentials: admin / prom-operator
```

## Verification

### Step 13: Verify Deployment

```bash
# Check all pods are running
kubectl get pods -n sql-studio

# Check services
kubectl get svc -n sql-studio

# Check ingress
kubectl get ingress -n sql-studio

# Check HPA
kubectl get hpa -n sql-studio

# Test health endpoints
curl https://api.sql-studio.app/health
curl https://sql-studio.app/health

# Test API
curl https://api.sql-studio.app/api/v1/version

# Check logs
kubectl logs -f deployment/sql-studio-backend -n sql-studio
kubectl logs -f deployment/sql-studio-frontend -n sql-studio

# Check resource usage
kubectl top pods -n sql-studio
kubectl top nodes
```

### Step 14: Setup CI/CD

```bash
# Configure GitHub Secrets
# Go to GitHub repo → Settings → Secrets and variables → Actions

# Add secrets:
# - GCP_PROJECT_ID
# - GCP_SA_KEY (service account JSON)
# - GKE_CLUSTER_NAME
# - TURSO_URL
# - TURSO_AUTH_TOKEN
# - JWT_SECRET
# - RESEND_API_KEY (optional)
# - RESEND_FROM_EMAIL (optional)

# Create service account for GitHub Actions
gcloud iam service-accounts create github-actions \
  --display-name="GitHub Actions"

# Grant permissions
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:github-actions@$PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/container.developer"

# Create and download key
gcloud iam service-accounts keys create key.json \
  --iam-account=github-actions@$PROJECT_ID.iam.gserviceaccount.com

# Add key.json contents to GitHub secret GCP_SA_KEY
cat key.json  # Copy this to GitHub Secrets

# Test workflow
git tag v1.0.0
git push origin v1.0.0
# Or trigger manually in GitHub Actions UI
```

## Troubleshooting

### Pods Not Starting

```bash
# Check pod events
kubectl describe pod POD_NAME -n sql-studio

# Check logs
kubectl logs POD_NAME -n sql-studio

# Common issues:
# - Image pull errors → Check image exists and credentials
# - CrashLoopBackOff → Check application logs
# - Pending → Check resource availability
```

### Certificate Issues

```bash
# Check certificate status
kubectl describe certificate sql-studio-cert -n sql-studio

# Check cert-manager logs
kubectl logs -n cert-manager deployment/cert-manager

# Check challenge status
kubectl get challenge -n sql-studio
kubectl describe challenge -n sql-studio

# Force certificate renewal
kubectl delete secret sql-studio-tls -n sql-studio
```

### Ingress Issues

```bash
# Check ingress status
kubectl describe ingress sql-studio -n sql-studio

# Check ingress controller logs
kubectl logs -n ingress-nginx deployment/ingress-nginx-controller

# Check ingress controller service
kubectl get svc -n ingress-nginx
```

### Database Connection Issues

```bash
# Test Turso connection
turso db shell sql-studio-production

# Check secrets
kubectl get secret sql-studio-secrets -n sql-studio -o yaml

# Test from pod
kubectl exec -it deployment/sql-studio-backend -n sql-studio -- \
  curl -v $TURSO_URL
```

## Next Steps

- [ ] Setup monitoring dashboards
- [ ] Configure alerts
- [ ] Setup log aggregation
- [ ] Plan backup and disaster recovery
- [ ] Load test the application
- [ ] Setup staging environment
- [ ] Document runbook procedures

## Additional Resources

- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [nginx Ingress Controller](https://kubernetes.github.io/ingress-nginx/)
- [cert-manager Documentation](https://cert-manager.io/docs/)
- [Turso Documentation](https://docs.turso.tech/)
- [Cloudflare Documentation](https://developers.cloudflare.com/)

## Support

For deployment issues:
- Check [RUNBOOK.md](./RUNBOOK.md) for operational procedures
- Review [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) for common issues
- Open GitHub issue: https://github.com/your-org/sql-studio/issues
