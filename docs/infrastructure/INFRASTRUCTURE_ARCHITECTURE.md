# Howlerops - Infrastructure Architecture

Comprehensive overview of the production infrastructure architecture and design decisions.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                           Internet                              │
└────────────┬────────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Cloudflare CDN                              │
│  ┌──────────────┐  ┌───────────────┐  ┌─────────────────┐     │
│  │ Edge Caching │  │ DDoS Protection│  │ WAF & Bot Shield│     │
│  │ Static Assets│  │ Rate Limiting  │  │ Security Rules  │     │
│  └──────────────┘  └───────────────┘  └─────────────────┘     │
└────────────┬────────────────────────────────────────────────────┘
             │ TLS 1.3
             ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Kubernetes Cluster (GKE)                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              nginx Ingress Controller                     │  │
│  │  - TLS Termination    - Path Routing                     │  │
│  │  - Load Balancing     - SSL Certificates                 │  │
│  └───────┬────────────────────────┬─────────────────────────┘  │
│          │                        │                             │
│          ▼                        ▼                             │
│  ┌───────────────┐        ┌────────────────┐                  │
│  │   Frontend    │        │    Backend     │                  │
│  │   Service     │        │    Service     │                  │
│  │   ClusterIP   │        │    ClusterIP   │                  │
│  └───────┬───────┘        └───────┬────────┘                  │
│          │                        │                             │
│          ▼                        ▼                             │
│  ┌───────────────┐        ┌────────────────┐                  │
│  │   Frontend    │        │    Backend     │                  │
│  │  Deployment   │        │   Deployment   │                  │
│  │               │        │                │                  │
│  │  ┌─────────┐  │        │  ┌──────────┐  │                  │
│  │  │ nginx   │  │        │  │ Go API   │  │                  │
│  │  │ Pod 1   │  │        │  │  Pod 1   │  │                  │
│  │  └─────────┘  │        │  └──────────┘  │                  │
│  │  ┌─────────┐  │        │  ┌──────────┐  │                  │
│  │  │ nginx   │  │        │  │ Go API   │  │                  │
│  │  │ Pod 2   │  │        │  │  Pod 2   │  │                  │
│  │  └─────────┘  │        │  └──────────┘  │                  │
│  │               │        │  ┌──────────┐  │                  │
│  │ (2 replicas)  │        │  │   HPA    │  │                  │
│  │               │        │  │ 2-10 pods│  │                  │
│  └───────────────┘        │  └──────────┘  │                  │
│                           └───────┬────────┘                  │
└───────────────────────────────────┼───────────────────────────┘
                                    │
                                    ▼
                    ┌───────────────────────────┐
                    │     Turso Database        │
                    │      (libSQL/SQLite)      │
                    │                           │
                    │  ┌─────────────────────┐  │
                    │  │   Primary (ORD)     │  │
                    │  │   Chicago           │  │
                    │  └──────────┬──────────┘  │
                    │             │              │
                    │    ┌────────┴────────┐     │
                    │    │    Replicas     │     │
                    │  ┌─▼─┐  ┌─▼─┐  ┌─▼─┐ │     │
                    │  │IAD│  │SJC│  │FRA│ │     │
                    │  │USA│  │USA│  │EUR│ │     │
                    │  └───┘  └───┘  └───┘ │     │
                    │                       │     │
                    └───────────────────────┘     │
```

## Component Details

### 1. CDN Layer (Cloudflare)

**Purpose**: Global content delivery, DDoS protection, and edge caching

**Features**:
- **Edge Caching**: Static assets cached at 200+ locations worldwide
- **DDoS Protection**: Automatic mitigation of Layer 3/4/7 attacks
- **WAF (Web Application Firewall)**: Protection against OWASP Top 10
- **Bot Management**: Block malicious bots, allow good bots
- **Rate Limiting**: Protect against abuse and API flooding
- **SSL/TLS**: Automatic certificate provisioning and renewal
- **Performance**: HTTP/2, HTTP/3 (QUIC), Early Hints, Brotli compression

**Cache Strategy**:
```
Static Assets (/assets/*):    Cache-Control: max-age=31536000, immutable
Images/Fonts:                 Cache-Control: max-age=2592000
HTML Files:                   Cache-Control: no-cache, no-store
API Endpoints:                Bypass cache
```

**Cost**: $0/month (Free tier) or $20/month (Pro tier)

### 2. Ingress Layer (nginx Ingress Controller)

**Purpose**: Kubernetes ingress, TLS termination, routing

**Features**:
- **Path-based Routing**: Route traffic to frontend or backend based on path
- **TLS Termination**: Decrypt HTTPS traffic, forward HTTP internally
- **Load Balancing**: Distribute traffic across backend pods
- **Rate Limiting**: Application-level rate limiting
- **CORS Handling**: Cross-origin resource sharing
- **WebSocket Support**: Upgrade HTTP connections for real-time features

**Configuration**:
```yaml
Routing Rules:
  /           → Frontend Service (React SPA)
  /api/*      → Backend Service (Go API)
  /health     → Backend Service (Health check)

TLS:
  Certificates: Managed by cert-manager
  Renewal: Automatic via Let's Encrypt
  Protocol: TLS 1.2, TLS 1.3 only
```

**Cost**: Included in cluster cost

### 3. Application Layer

#### Frontend (React + nginx)

**Deployment**:
- **Replicas**: 2 (fixed)
- **Resources**: 50m CPU, 128Mi memory (request), 200m CPU, 512Mi memory (limit)
- **Image**: nginx:1.25-alpine + React build
- **Size**: ~100MB compressed

**Features**:
- React 18 SPA with Vite build
- Static asset versioning (cache busting)
- nginx with gzip/brotli compression
- Security headers (CSP, HSTS, X-Frame-Options)
- Health check endpoint

**Scaling**: Fixed replicas (no HPA needed for static content)

#### Backend (Go API)

**Deployment**:
- **Replicas**: 2-10 (auto-scaling based on CPU)
- **Resources**: 100m CPU, 256Mi memory (request), 500m CPU, 1Gi memory (limit)
- **Image**: Alpine Linux + Go binary
- **Size**: ~25MB compressed

**Features**:
- RESTful API with Chi router
- JWT authentication
- Database connection pooling
- Structured JSON logging
- Prometheus metrics endpoint
- Graceful shutdown handling

**Scaling**: Horizontal Pod Autoscaler (HPA)
- Target CPU: 70%
- Scale up: Add pod when CPU > 70% for 30s
- Scale down: Remove pod when CPU < 30% for 5min
- Min replicas: 2
- Max replicas: 10

### 4. Data Layer

#### Turso Database (libSQL)

**Configuration**:
- **Primary Region**: ORD (Chicago) - closest to us-central1
- **Read Replicas**: IAD (East US), SJC (West US), FRA (Europe), SYD (Asia-Pacific)
- **Sync Mode**: Asynchronous (5s interval)
- **Connection Pool**: 25 max open connections, 10 max idle

**Features**:
- **Global Distribution**: Low-latency reads from nearest replica
- **Automatic Backups**: Hourly backups with 7-day retention
- **Point-in-Time Recovery**: Restore to any point within 7 days
- **Encryption**: At rest and in transit (TLS 1.3)
- **High Availability**: Multi-region replication

**Cost**: $29/month (Pro tier)

### 5. Security Layer

#### Network Security

**Network Policies**:
```
Default: Deny all ingress/egress
Allow:
  - Frontend → Backend (port 8500)
  - Backend → Turso (port 443)
  - Backend → DNS (port 53)
  - Ingress → Frontend (port 80)
  - Ingress → Backend (port 8500)
  - Prometheus → Backend (port 9100)
```

**Pod Security**:
- Run as non-root (UID 1001)
- Read-only root filesystem
- Drop all capabilities
- No privilege escalation
- Security context constraints

**RBAC**:
- Service accounts per component
- Least privilege access
- No default service account token mounting

#### Secrets Management

**Kubernetes Secrets**:
- Encrypted at rest (via etcd encryption)
- Mounted as volumes (not env vars)
- Access controlled via RBAC
- Audited access logs

**External Secrets** (optional):
- GCP Secret Manager integration
- Automatic rotation
- Centralized management

**Secret Rotation**:
- JWT Secret: Every 90 days
- Turso Token: Every 90 days
- TLS Certificates: Auto-renewed 15 days before expiry

#### SSL/TLS

**Certificate Management**:
- Provider: Let's Encrypt (via cert-manager)
- Renewal: Automatic (15 days before expiry)
- Protocol: TLS 1.2, TLS 1.3 only
- Ciphers: Modern ciphers only (no weak ciphers)

**Security Headers**:
```
X-Frame-Options: DENY
X-Content-Type-Options: nosniff
X-XSS-Protection: 1; mode=block
Strict-Transport-Security: max-age=31536000
Referrer-Policy: strict-origin-when-cross-origin
Content-Security-Policy: (customized)
```

### 6. Monitoring and Observability

#### Metrics (Prometheus)

**Collected Metrics**:
- **System**: CPU, memory, disk, network
- **Application**: Request rate, latency, errors
- **Database**: Connection pool, query performance
- **Business**: User signups, API usage

**Retention**: 30 days

#### Logs (Structured JSON)

**Log Levels**:
- **Production**: INFO, WARN, ERROR
- **Development**: DEBUG

**Log Aggregation**:
- stdout/stderr → Kubernetes logs
- Optional: Fluent Bit → Cloud Logging/Elasticsearch

#### Alerts

**Critical Alerts**:
- Pod not ready for 5 minutes
- High error rate (>5%)
- High latency (p95 > 1s)
- Database connection failures
- Certificate expiring in 7 days

**Warning Alerts**:
- High CPU usage (>80%)
- High memory usage (>90%)
- Scaling at max capacity
- Slow queries (>500ms)

### 7. CI/CD Pipeline

**Workflow**: GitHub Actions

**Stages**:
1. **Validate**: Check secrets, lint, test
2. **Build**: Docker images for backend/frontend
3. **Scan**: Security scan with Trivy
4. **Migrate**: Run database migrations
5. **Deploy**: Update Kubernetes deployments
6. **Smoke Test**: Verify health endpoints
7. **Rollback**: Automatic rollback on failure

**Deployment Strategy**: Rolling update
- Max surge: 1 (can have 1 extra pod)
- Max unavailable: 0 (zero downtime)

**Triggers**:
- Release published (recommended)
- Manual workflow dispatch
- (Optional) Push to main

## Traffic Flow

### User Request Flow

```
1. User requests https://sql-studio.app
   ↓
2. DNS resolves to Cloudflare edge server
   ↓
3. Cloudflare checks cache:
   - Static asset? → Serve from cache
   - Dynamic content? → Forward to origin
   ↓
4. Request forwarded to Ingress Load Balancer IP
   ↓
5. nginx Ingress Controller:
   - Terminates TLS
   - Checks path: / → Frontend, /api/* → Backend
   - Applies rate limiting
   ↓
6. Request forwarded to appropriate Service
   ↓
7. Service load balances across pods
   ↓
8. Pod processes request:
   - Frontend: Serve static files
   - Backend: Process API request, query database
   ↓
9. Response flows back through same path
   ↓
10. Cloudflare caches if appropriate, adds security headers
    ↓
11. User receives response
```

### Data Flow

```
1. User submits data via API
   ↓
2. Backend validates and sanitizes input
   ↓
3. Backend authenticates JWT token
   ↓
4. Backend queries/writes to Turso
   ↓
5. Turso writes to primary database (ORD)
   ↓
6. Turso asynchronously replicates to read replicas
   ↓
7. Backend returns response
   ↓
8. Frontend updates UI
```

## Scaling Strategy

### Horizontal Scaling

**Frontend**: Fixed 2 replicas (no auto-scaling needed)

**Backend**: Auto-scales 2-10 replicas
- **Scale Up**: CPU > 70% for 30s → Add 1-2 pods
- **Scale Down**: CPU < 30% for 5min → Remove 1 pod
- **Cooldown**: 60s scale-up, 300s scale-down

**Database**: Global read replicas for geographic distribution

### Vertical Scaling

**When to scale vertically**:
- Consistently hitting resource limits
- Horizontal scaling not effective
- Memory-intensive workloads

**How to scale**:
1. Update resource requests/limits in deployment
2. Apply changes (triggers rolling update)
3. Monitor performance

### Load Testing

**Before production**:
- Target: 1000 req/s sustained
- Duration: 30 minutes
- Ramp-up: Gradual increase over 5 minutes
- Monitor: CPU, memory, latency, errors

**Tools**: k6, Apache Bench, Vegeta

## Disaster Recovery

### Backup Strategy

**Database**:
- Automatic hourly backups (Turso managed)
- Point-in-time recovery (7 days)
- Manual backups before major changes

**Kubernetes Configurations**:
- All manifests in Git (version controlled)
- Helm charts for complex deployments
- Velero for cluster backups (optional)

### Recovery Procedures

**RTO (Recovery Time Objective)**: 1 hour
**RPO (Recovery Point Objective)**: 1 hour

**Scenarios**:

1. **Pod Failure**: Automatic (Kubernetes restarts)
2. **Node Failure**: Automatic (pods rescheduled)
3. **Deployment Failure**: Automatic rollback
4. **Database Failure**: Restore from backup
5. **Complete Cluster Failure**: Rebuild from Git + restore data

### High Availability

**Component Availability**:
- Frontend: 99.9% (2 replicas across zones)
- Backend: 99.9% (2-10 replicas across zones)
- Database: 99.95% (Turso SLA)
- Ingress: 99.9% (GCP Load Balancer SLA)
- CDN: 100% (Cloudflare SLA)

**Overall Target**: 99.9% uptime (8.76 hours/year downtime)

## Cost Optimization

### Current Estimated Costs

**Monthly Breakdown**:
- GKE Cluster: $150-300 (2-10 nodes, e2-medium)
- Load Balancer: $18
- Turso Database: $29 (Pro tier)
- Cloudflare: $0-20 (Free or Pro)
- Storage: $10 (persistent volumes, images)
- **Total**: $207-377/month

**Cost Reduction Strategies**:
1. Use Spot/Preemptible instances (30% savings)
2. Right-size node types
3. Enable cluster autoscaling
4. Use committed use discounts (Google)
5. Optimize image sizes
6. Review and clean up unused resources

### Scaling Costs

**At 1000 users**:
- Backend: 2-3 pods average
- Storage: ~10GB
- Database: Pro tier
- **Est. Cost**: $250/month

**At 10,000 users**:
- Backend: 5-7 pods average
- Storage: ~100GB
- Database: Pro tier (may need higher)
- **Est. Cost**: $400/month

**At 100,000 users**:
- Backend: 10+ pods (may need more nodes)
- Storage: ~1TB
- Database: Scale or Enterprise tier
- CDN: Pro tier recommended
- **Est. Cost**: $1000-1500/month

## Performance Targets

**Response Times** (p95):
- Static assets: < 50ms (cached)
- API endpoints: < 200ms
- Database queries: < 50ms

**Throughput**:
- Frontend: 10,000 req/s (CDN cached)
- Backend: 1,000 req/s (per instance)
- Database: 10,000 req/s (reads), 1,000 req/s (writes)

**Availability**: 99.9% uptime

## Security Considerations

**Threat Model**:
- DDoS attacks → Mitigated by Cloudflare
- SQL injection → Input validation + prepared statements
- XSS attacks → CSP headers + sanitization
- CSRF attacks → JWT tokens + SameSite cookies
- Credential stuffing → Rate limiting + MFA
- Data breaches → Encryption + access controls

**Compliance**:
- GDPR: User data rights, encryption, audit logs
- SOC 2: Access controls, monitoring, incident response
- PCI DSS: (If handling payments) Stripe handles

## Future Improvements

**Short-term** (1-3 months):
- [ ] Add Grafana dashboards
- [ ] Implement distributed tracing (Jaeger)
- [ ] Add end-to-end tests
- [ ] Document runbooks

**Medium-term** (3-6 months):
- [ ] Multi-region deployment
- [ ] Blue-green deployments
- [ ] Chaos engineering tests
- [ ] Advanced auto-scaling (custom metrics)

**Long-term** (6-12 months):
- [ ] Service mesh (Istio)
- [ ] Multi-cloud strategy
- [ ] Edge computing (Cloudflare Workers)
- [ ] AI/ML model serving

## References

- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [nginx Ingress Controller](https://kubernetes.github.io/ingress-nginx/)
- [Turso Documentation](https://docs.turso.tech/)
- [Cloudflare Documentation](https://developers.cloudflare.com/)
- [12-Factor App](https://12factor.net/)
