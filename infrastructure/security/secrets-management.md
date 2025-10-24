# SQL Studio - Secrets Management Guide

Comprehensive guide for managing secrets securely in production.

## Table of Contents

- [Overview](#overview)
- [Kubernetes Secrets](#kubernetes-secrets)
- [External Secrets Operator](#external-secrets-operator)
- [Secret Rotation](#secret-rotation)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

SQL Studio uses multiple secret management approaches depending on the deployment environment:

1. **Kubernetes Secrets** - Built-in Kubernetes secret management
2. **GCP Secret Manager** - Cloud-native secret management (recommended for GCP)
3. **External Secrets Operator** - Sync secrets from external providers
4. **HashiCorp Vault** - Enterprise secret management (advanced)

## Kubernetes Secrets

### Creating Secrets

```bash
# From literal values
kubectl create secret generic sql-studio-secrets \
  --namespace=sql-studio \
  --from-literal=turso-url='libsql://your-db.turso.io' \
  --from-literal=turso-auth-token='your-token' \
  --from-literal=jwt-secret='your-64-char-secret' \
  --from-literal=resend-api-key='re_your_key' \
  --from-literal=resend-from-email='noreply@sql-studio.app'

# From .env file
kubectl create secret generic sql-studio-secrets \
  --namespace=sql-studio \
  --from-env-file=.env.production

# From individual files
kubectl create secret generic sql-studio-secrets \
  --namespace=sql-studio \
  --from-file=turso-url=./secrets/turso-url.txt \
  --from-file=turso-auth-token=./secrets/turso-token.txt \
  --from-file=jwt-secret=./secrets/jwt-secret.txt

# From YAML (base64 encoded)
kubectl apply -f secrets.yaml
```

### Viewing Secrets

```bash
# List secrets
kubectl get secrets -n sql-studio

# Describe secret (without values)
kubectl describe secret sql-studio-secrets -n sql-studio

# View secret values (CAUTION in production)
kubectl get secret sql-studio-secrets -n sql-studio -o json | \
  jq '.data | map_values(@base64d)'

# Get specific key
kubectl get secret sql-studio-secrets -n sql-studio \
  -o jsonpath='{.data.turso-url}' | base64 -d
```

### Updating Secrets

```bash
# Update entire secret
kubectl create secret generic sql-studio-secrets \
  --namespace=sql-studio \
  --from-literal=jwt-secret='new-secret' \
  --dry-run=client -o yaml | kubectl apply -f -

# Update specific key using patch
kubectl patch secret sql-studio-secrets -n sql-studio \
  --type='json' \
  -p='[{"op": "replace", "path": "/data/jwt-secret", "value": "'$(echo -n "new-secret" | base64)'"}]'

# Delete and recreate (triggers pod restart if using env)
kubectl delete secret sql-studio-secrets -n sql-studio
kubectl create secret generic sql-studio-secrets ...
```

### Using Secrets in Pods

```yaml
# As environment variables
env:
  - name: TURSO_URL
    valueFrom:
      secretKeyRef:
        name: sql-studio-secrets
        key: turso-url

# As volume mounts (recommended - more secure)
volumes:
  - name: secrets
    secret:
      secretName: sql-studio-secrets
      defaultMode: 0400

volumeMounts:
  - name: secrets
    mountPath: /secrets
    readOnly: true

# Access in application:
# Read from /secrets/turso-url file
```

## External Secrets Operator

### Installation

```bash
# Install External Secrets Operator
helm repo add external-secrets https://charts.external-secrets.io
helm install external-secrets \
  external-secrets/external-secrets \
  --namespace external-secrets-system \
  --create-namespace
```

### GCP Secret Manager Integration

```yaml
# SecretStore (namespace-scoped)
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: gcpsm-secret-store
  namespace: sql-studio
spec:
  provider:
    gcpsm:
      projectID: "your-project-id"
      auth:
        workloadIdentity:
          clusterLocation: us-central1
          clusterName: your-cluster
          serviceAccountRef:
            name: sql-studio-backend

---
# ExternalSecret (syncs from GCP to K8s)
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: sql-studio-secrets
  namespace: sql-studio
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: gcpsm-secret-store
    kind: SecretStore

  target:
    name: sql-studio-secrets
    creationPolicy: Owner

  data:
    - secretKey: turso-url
      remoteRef:
        key: turso-url

    - secretKey: turso-auth-token
      remoteRef:
        key: turso-auth-token

    - secretKey: jwt-secret
      remoteRef:
        key: jwt-secret

    - secretKey: resend-api-key
      remoteRef:
        key: resend-api-key

    - secretKey: resend-from-email
      remoteRef:
        key: resend-from-email
```

### AWS Secrets Manager Integration

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: aws-secrets-manager
  namespace: sql-studio
spec:
  provider:
    aws:
      service: SecretsManager
      region: us-east-1
      auth:
        jwt:
          serviceAccountRef:
            name: sql-studio-backend

---
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: sql-studio-secrets
  namespace: sql-studio
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secrets-manager
    kind: SecretStore

  target:
    name: sql-studio-secrets

  dataFrom:
    - extract:
        key: sql-studio/production  # AWS secret name
```

### HashiCorp Vault Integration

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: vault-backend
  namespace: sql-studio
spec:
  provider:
    vault:
      server: "https://vault.sql-studio.app"
      path: "secret"
      version: "v2"
      auth:
        kubernetes:
          mountPath: "kubernetes"
          role: "sql-studio-role"
          serviceAccountRef:
            name: sql-studio-backend

---
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: sql-studio-secrets
  namespace: sql-studio
spec:
  refreshInterval: 15m
  secretStoreRef:
    name: vault-backend
    kind: SecretStore

  target:
    name: sql-studio-secrets

  data:
    - secretKey: turso-url
      remoteRef:
        key: sql-studio/production
        property: turso-url
```

## Secret Rotation

### Manual Rotation Process

1. **Generate new secret**:
   ```bash
   # JWT secret
   NEW_JWT_SECRET=$(openssl rand -base64 64)

   # Turso auth token
   NEW_TURSO_TOKEN=$(turso db tokens create sql-studio-production --expiration 90d)
   ```

2. **Update in secret manager**:
   ```bash
   # GCP Secret Manager
   echo -n "$NEW_JWT_SECRET" | gcloud secrets versions add jwt-secret --data-file=-

   # Kubernetes directly
   kubectl create secret generic sql-studio-secrets \
     --namespace=sql-studio \
     --from-literal=jwt-secret="$NEW_JWT_SECRET" \
     --dry-run=client -o yaml | kubectl apply -f -
   ```

3. **Trigger pod restart** (if using env vars):
   ```bash
   kubectl rollout restart deployment/sql-studio-backend -n sql-studio
   kubectl rollout status deployment/sql-studio-backend -n sql-studio
   ```

4. **Verify application**:
   ```bash
   kubectl logs -f deployment/sql-studio-backend -n sql-studio
   curl -f https://api.sql-studio.app/health
   ```

### Automated Rotation with External Secrets

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: sql-studio-secrets
  namespace: sql-studio
spec:
  refreshInterval: 1h  # Check for changes every hour

  target:
    name: sql-studio-secrets
    creationPolicy: Owner

    # Template for combining multiple secrets
    template:
      engineVersion: v2
      data:
        jwt-secret: "{{ .jwt_secret }}"
        turso-url: "{{ .turso_url }}"
        turso-auth-token: "{{ .turso_token }}"

  dataFrom:
    - extract:
        key: sql-studio/production
```

### Rotation Schedule

| Secret | Rotation Frequency | Method |
|--------|-------------------|--------|
| JWT Secret | Every 90 days | Manual/Automated |
| Turso Auth Token | Every 90 days | Manual (Turso CLI) |
| Resend API Key | Every 180 days | Manual (Resend Dashboard) |
| TLS Certificates | Auto-renewed 15 days before expiry | cert-manager |
| Database Credentials | Every 90 days | Manual |

## Best Practices

### 1. Never Commit Secrets to Git

```bash
# Add to .gitignore
echo "secrets.yaml" >> .gitignore
echo ".env.production" >> .gitignore
echo "*.key" >> .gitignore
echo "*.pem" >> .gitignore
```

### 2. Use Strong Secrets

```bash
# Generate strong secrets
openssl rand -base64 64  # JWT secret (64 bytes)
openssl rand -hex 32     # API key (32 bytes)
uuidgen                  # Unique identifier
```

### 3. Encrypt Secrets at Rest

```bash
# Enable encryption in kube-apiserver
# Add to kube-apiserver flags:
--encryption-provider-config=/etc/kubernetes/encryption-config.yaml
```

### 4. Limit Secret Access

```yaml
# RBAC policy for secrets
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: secret-reader
  namespace: sql-studio
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get"]
    resourceNames:
      - "sql-studio-secrets"  # Only specific secret
```

### 5. Audit Secret Access

```yaml
# Audit policy for secrets
apiVersion: audit.k8s.io/v1
kind: Policy
rules:
  - level: Metadata
    resources:
      - group: ""
        resources: ["secrets"]
    verbs: ["get", "list", "watch"]
```

### 6. Use Volume Mounts Over Environment Variables

```yaml
# GOOD: Volume mount (more secure)
volumes:
  - name: secrets
    secret:
      secretName: sql-studio-secrets
volumeMounts:
  - name: secrets
    mountPath: /secrets
    readOnly: true

# LESS SECURE: Environment variables (visible in ps, logs)
env:
  - name: JWT_SECRET
    valueFrom:
      secretKeyRef:
        name: sql-studio-secrets
        key: jwt-secret
```

### 7. Separate Secrets by Environment

```bash
# Development
kubectl create secret generic sql-studio-secrets \
  --namespace=sql-studio-dev \
  --from-literal=...

# Staging
kubectl create secret generic sql-studio-secrets \
  --namespace=sql-studio-staging \
  --from-literal=...

# Production
kubectl create secret generic sql-studio-secrets \
  --namespace=sql-studio \
  --from-literal=...
```

### 8. Monitor Secret Usage

```bash
# Check which pods are using secrets
kubectl get pods -n sql-studio -o json | \
  jq '.items[] | select(.spec.volumes[]?.secret.secretName=="sql-studio-secrets") | .metadata.name'

# Audit secret access
kubectl logs -n kube-system deployment/kube-apiserver | \
  grep "secret.*sql-studio-secrets"
```

## Troubleshooting

### Secret Not Found

```bash
# Check if secret exists
kubectl get secret sql-studio-secrets -n sql-studio

# If not, create it
kubectl create secret generic sql-studio-secrets ...
```

### Permission Denied

```bash
# Check RBAC permissions
kubectl auth can-i get secrets --namespace=sql-studio --as=system:serviceaccount:sql-studio:sql-studio-backend

# Check service account
kubectl describe serviceaccount sql-studio-backend -n sql-studio

# Check role binding
kubectl describe rolebinding sql-studio-backend-binding -n sql-studio
```

### Secret Value Empty

```bash
# Check secret data
kubectl get secret sql-studio-secrets -n sql-studio -o yaml

# Verify base64 encoding
echo "your-value" | base64
```

### Pod Not Getting Secret Updates

```bash
# Secrets used as env vars require pod restart
kubectl rollout restart deployment/sql-studio-backend -n sql-studio

# Secrets mounted as volumes update automatically (may take 60s)
kubectl exec -it deployment/sql-studio-backend -n sql-studio -- cat /secrets/jwt-secret
```

### External Secrets Not Syncing

```bash
# Check ExternalSecret status
kubectl describe externalsecret sql-studio-secrets -n sql-studio

# Check SecretStore status
kubectl describe secretstore gcpsm-secret-store -n sql-studio

# Check operator logs
kubectl logs -n external-secrets-system deployment/external-secrets
```

## Security Checklist

- [ ] Never commit secrets to version control
- [ ] Use strong, randomly generated secrets
- [ ] Rotate secrets regularly (90-day schedule)
- [ ] Enable encryption at rest
- [ ] Use RBAC to limit access
- [ ] Audit secret access
- [ ] Use volume mounts instead of env vars
- [ ] Separate secrets by environment
- [ ] Use external secret management for production
- [ ] Document secret rotation procedures
- [ ] Test secret rotation in staging first
- [ ] Monitor for secret expiration
- [ ] Have rollback plan for failed rotation

## Additional Resources

- [Kubernetes Secrets](https://kubernetes.io/docs/concepts/configuration/secret/)
- [External Secrets Operator](https://external-secrets.io/)
- [GCP Secret Manager](https://cloud.google.com/secret-manager/docs)
- [AWS Secrets Manager](https://aws.amazon.com/secrets-manager/)
- [HashiCorp Vault](https://www.vaultproject.io/)
- [cert-manager](https://cert-manager.io/)
