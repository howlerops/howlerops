# SQL Studio - Infrastructure Cost Estimation

Detailed breakdown of production infrastructure costs and optimization strategies.

## Monthly Cost Summary

| Component | Cost (Low) | Cost (High) | Notes |
|-----------|------------|-------------|-------|
| **Compute** | $150 | $300 | GKE cluster with 2-10 nodes |
| **Load Balancer** | $18 | $18 | Single GCP load balancer |
| **Database** | $29 | $29 | Turso Pro tier |
| **CDN** | $0 | $20 | Cloudflare Free/Pro |
| **Storage** | $10 | $30 | Persistent volumes + images |
| **Bandwidth** | $0 | $50 | First 1TB free on GCP |
| **Monitoring** | $0 | $50 | Optional (Datadog, New Relic) |
| **SSL Certificates** | $0 | $0 | Let's Encrypt (free) |
| **DNS** | $0 | $0 | Cloudflare (free) |
| **Email Service** | $0 | $10 | Resend (free tier) |
| ****TOTAL**/** | **$207** | **$507** | **Monthly** |
| **Annual** | **$2,484** | **$6,084** | **12 months** |

## Detailed Cost Breakdown

### 1. Compute (Kubernetes Cluster)

#### GCP GKE

**Node Configuration**:
- Machine type: e2-medium (2 vCPU, 4GB RAM)
- Region: us-central1
- Autoscaling: 2-10 nodes

**Cost Calculation**:
```
Minimum (2 nodes):
  - e2-medium: $24.67/node/month
  - Total: $24.67 × 2 = $49.34/month

Average (5 nodes):
  - e2-medium: $24.67/node/month
  - Total: $24.67 × 5 = $123.35/month

Maximum (10 nodes):
  - e2-medium: $24.67/node/month
  - Total: $24.67 × 10 = $246.70/month

GKE Management: Free for up to 1 cluster
```

**Optimization**:
- Use Spot VMs: 60-91% cheaper ($5-10/node/month)
- Reserved capacity: 37% discount with 1-year commitment
- Right-size nodes based on actual usage

#### AWS EKS

**Cost Comparison**:
```
EKS Control Plane: $0.10/hour = $73/month
Worker Nodes (t3.medium):
  - On-demand: $30.37/node/month
  - Spot: $9-15/node/month

Minimum (2 nodes): $73 + ($30.37 × 2) = $133.74/month
Average (5 nodes): $73 + ($30.37 × 5) = $224.85/month
Maximum (10 nodes): $73 + ($30.37 × 10) = $376.70/month
```

#### Azure AKS

**Cost Comparison**:
```
AKS Control Plane: Free
Worker Nodes (B2s):
  - On-demand: $30.37/node/month
  - Spot: ~$10/node/month

Minimum (2 nodes): $60.74/month
Average (5 nodes): $151.85/month
Maximum (10 nodes): $303.70/month
```

**Recommendation**: GCP GKE for best price/performance ratio

### 2. Load Balancer

**GCP Load Balancer**:
```
Forwarding rules: $0.025/hour = $18.25/month
Ingress traffic: Free (first 1TB)
Egress traffic: $0.12/GB (over 1TB)

Monthly: ~$18/month (low traffic)
```

**AWS Application Load Balancer**:
```
ALB hours: $0.0225/hour = $16.43/month
Load Balancer Capacity Units (LCUs): ~$0.008/hour = $5.84/month
Total: ~$22.27/month
```

**Azure Load Balancer**:
```
Basic: Free
Standard: $0.025/hour = $18.25/month
```

### 3. Database (Turso)

**Pricing Tiers**:

| Tier | Monthly Cost | Storage | Rows Read | Rows Written |
|------|--------------|---------|-----------|--------------|
| Free | $0 | 500MB | 1B/month | 25M/month |
| Pro | $29 | 5GB | Unlimited | Unlimited |
| Enterprise | Custom | Custom | Unlimited | Unlimited |

**Expected Usage** (1000 users):
- Storage: 500MB-1GB
- Reads: 10M/month
- Writes: 1M/month
- **Recommended**: Pro tier ($29/month)

**Expected Usage** (10,000 users):
- Storage: 2-5GB
- Reads: 100M/month
- Writes: 10M/month
- **Recommended**: Pro tier ($29/month)

**Expected Usage** (100,000 users):
- Storage: 10-20GB
- Reads: 1B/month
- Writes: 100M/month
- **Recommended**: Enterprise tier (custom pricing)

### 4. CDN (Cloudflare)

**Pricing Tiers**:

| Tier | Monthly Cost | Features |
|------|--------------|----------|
| Free | $0 | Unlimited bandwidth, basic DDoS protection |
| Pro | $20 | Advanced DDoS, WAF, image optimization |
| Business | $200 | Custom SSL, advanced rate limiting |
| Enterprise | Custom | Dedicated support, SLA |

**Bandwidth Costs** (alternatives):
- AWS CloudFront: $0.085/GB (first 10TB)
- Google Cloud CDN: $0.08/GB (first 10TB)
- Cloudflare: Unlimited (all tiers)

**Recommendation**: Start with Free tier, upgrade to Pro if needed

### 5. Storage

#### Persistent Volumes (GCP)

**Cost**:
```
Standard persistent disk: $0.04/GB/month
SSD persistent disk: $0.17/GB/month

Expected usage:
  - Logs: 5GB = $0.20-0.85/month
  - Database cache: 10GB = $0.40-1.70/month
  - Container images: 5GB = $0.20-0.85/month

Total: ~$1-3/month (low traffic)
       ~$5-10/month (medium traffic)
```

#### Container Registry (GCR)

**Cost**:
```
Storage: $0.026/GB/month
Egress: $0.12/GB (within same region: free)

Expected usage:
  - Backend image: 100MB = $0.0026/month
  - Frontend image: 100MB = $0.0026/month
  - Total: ~$0.01/month (negligible)
```

### 6. Bandwidth

#### Ingress (Inbound)

**Cost**: Free (all cloud providers)

#### Egress (Outbound)

**GCP Pricing**:
```
Same region: Free
Same continent: $0.01/GB
Worldwide (first 1TB): Free
Worldwide (1TB-10TB): $0.12/GB
Worldwide (10TB+): $0.08/GB
```

**Expected Usage**:
```
1,000 users:
  - Average: 100MB/user/month = 100GB
  - Cost: Free (under 1TB)

10,000 users:
  - Average: 100MB/user/month = 1TB
  - Cost: Free (at threshold)

100,000 users:
  - Average: 100MB/user/month = 10TB
  - Cost: $1,080/month (9TB × $0.12/GB)
  - With CDN: ~$100/month (CDN offloads 90%)
```

**CDN Impact**: Reduces egress by 80-95%

### 7. Optional Services

#### Monitoring (Prometheus + Grafana)

**Self-Hosted**:
- Resources: 1 node (e2-small) = $12.50/month
- Storage: 100GB = $4/month
- **Total**: ~$16.50/month

**Managed (Datadog)**:
- 5 hosts × $15/host = $75/month
- Logs: $0.10/GB = ~$10/month
- **Total**: ~$85/month

**Managed (New Relic)**:
- 100GB data × $0.30/GB = $30/month
- **Total**: ~$30/month

#### Log Aggregation

**Cloud Logging (GCP)**:
- First 50GB: Free
- Additional: $0.50/GB
- Expected: 10GB/month = Free

**Elasticsearch/Logstash/Kibana (ELK)**:
- Resources: 1 node (e2-standard-2) = $49.34/month
- Storage: 100GB = $4/month
- **Total**: ~$53/month

#### Email Service (Resend)

**Pricing**:
```
Free: 3,000 emails/month
Pro: $20/month (50,000 emails)
Enterprise: Custom

Expected usage:
  - 1,000 users: 5,000 emails/month = $20/month
  - 10,000 users: 50,000 emails/month = $20/month
  - 100,000 users: 500,000 emails/month = Custom
```

## Cost Scaling Projections

### Scenario 1: Small (1,000 users)

**Traffic**:
- 100,000 API requests/day
- 10GB egress/month
- 100GB database queries/month

**Infrastructure**:
- 2 nodes (minimum)
- 2 backend pods
- 2 frontend pods

**Monthly Cost**: $207

| Component | Cost |
|-----------|------|
| Compute | $49 |
| Load Balancer | $18 |
| Database | $29 |
| CDN | $0 |
| Storage | $10 |
| Bandwidth | $0 |
| Email | $20 |
| **Total** | **$126** |

### Scenario 2: Medium (10,000 users)

**Traffic**:
- 1,000,000 API requests/day
- 100GB egress/month
- 1TB database queries/month

**Infrastructure**:
- 5 nodes (average)
- 5 backend pods
- 2 frontend pods

**Monthly Cost**: $295

| Component | Cost |
|-----------|------|
| Compute | $123 |
| Load Balancer | $18 |
| Database | $29 |
| CDN | $20 |
| Storage | $15 |
| Bandwidth | $0 |
| Email | $20 |
| Monitoring | $70 |
| **Total** | **$295** |

### Scenario 3: Large (100,000 users)

**Traffic**:
- 10,000,000 API requests/day
- 1TB egress/month
- 10TB database queries/month

**Infrastructure**:
- 10 nodes (maximum)
- 10 backend pods
- 4 frontend pods

**Monthly Cost**: $677

| Component | Cost |
|-----------|------|
| Compute | $247 |
| Load Balancer | $18 |
| Database | $100 (Custom) |
| CDN | $20 |
| Storage | $30 |
| Bandwidth | $100 |
| Email | $62 (Custom) |
| Monitoring | $100 |
| **Total** | **$677** |

## Cost Optimization Strategies

### 1. Compute Optimization

**Use Spot/Preemptible Instances**:
- Savings: 60-91% ($10/node instead of $25)
- Risk: Instances can be terminated
- Strategy: Use for stateless workloads (backend pods)
- Implementation: Set node pool to preemptible

**Reserved Capacity**:
- Savings: 37% for 1-year, 55% for 3-year
- Strategy: Reserve baseline capacity (2 nodes)
- Dynamic workloads: Use on-demand for autoscaling

**Right-Sizing**:
- Monitor actual resource usage
- Reduce resource requests if over-provisioned
- Use smaller node types if possible

**Cluster Autoscaling**:
- Scale down unused nodes automatically
- Set aggressive scale-down timeout (5min)
- Use pod disruption budgets

### 2. Database Optimization

**Query Optimization**:
- Use indexes for common queries
- Batch writes where possible
- Cache frequently accessed data

**Data Lifecycle**:
- Archive old data to cold storage
- Delete unnecessary logs
- Implement data retention policies

**Connection Pooling**:
- Reuse database connections
- Set appropriate pool size (25 max)
- Monitor connection usage

### 3. CDN Optimization

**Aggressive Caching**:
- Cache static assets for 1 year
- Use cache-control headers
- Implement asset versioning

**Image Optimization**:
- Use WebP/AVIF formats
- Lazy load images
- Compress images (Cloudflare Polish)

**Minimize API Calls**:
- Batch API requests
- Use GraphQL for selective fetching
- Implement client-side caching

### 4. Bandwidth Optimization

**Use CDN**:
- Offloads 80-95% of traffic
- Reduces origin bandwidth costs
- Improves performance globally

**Compression**:
- Enable gzip/brotli for text content
- Reduces bandwidth by 70-80%

**Optimize Payloads**:
- Minimize JSON response sizes
- Use pagination for large datasets
- Implement response caching

### 5. Monitoring Optimization

**Self-Host**:
- Prometheus + Grafana: ~$17/month
- vs. Datadog: ~$85/month
- Savings: $68/month ($816/year)

**Sample Metrics**:
- Don't collect every metric
- Sample high-cardinality metrics
- Set appropriate retention (30 days)

### 6. Development Environments

**Shared Development Cluster**:
- Single cluster for all devs
- Use namespaces for isolation
- Scale down when not in use

**Ephemeral Environments**:
- Spin up for PR review
- Destroy after merge
- Use smaller instance types

## Cost Comparison: Cloud Providers

### 3-Month Total Cost Comparison

| Provider | Compute | Managed K8s | Load Balancer | Total |
|----------|---------|-------------|---------------|-------|
| **GCP** | $370 | $0 | $54 | **$424** |
| **AWS** | $456 | $219 | $67 | **$742** |
| **Azure** | $456 | $0 | $54 | **$510** |

**Winner**: GCP (most cost-effective for Kubernetes workloads)

### Alternative: Serverless

**Cloud Run (GCP)**:
```
Pricing:
  - CPU: $0.00002400/vCPU-second
  - Memory: $0.00000250/GiB-second
  - Requests: $0.40/million

Estimated (1,000 users):
  - Requests: 3M/month = $1.20
  - CPU time: 100 CPU-hours = $8.64
  - Memory: 200GB-hours = $0.50
  - Total: ~$10.34/month

Estimated (10,000 users):
  - Requests: 30M/month = $12
  - CPU time: 1,000 CPU-hours = $86.40
  - Memory: 2,000GB-hours = $5
  - Total: ~$103.40/month

Pros:
  - Pay only for actual usage
  - No cluster management
  - Auto-scaling built-in

Cons:
  - Cold starts (~1s)
  - Less control
  - Vendor lock-in
```

## Break-Even Analysis

**Kubernetes vs. Cloud Run**:

```
Kubernetes fixed cost: $207/month
Cloud Run variable cost: $0.034/1000 requests

Break-even:
  $207 ÷ $0.034 = 6,088,235 requests/month
  = 203,000 requests/day
  = 8,458 requests/hour

Recommendation:
  - < 200K requests/day: Cloud Run
  - > 200K requests/day: Kubernetes
```

## Annual Cost Projections

### Year 1 (Startup)

| Quarter | Users | Monthly Cost | Quarterly Cost |
|---------|-------|--------------|----------------|
| Q1 | 100 | $126 | $378 |
| Q2 | 500 | $170 | $510 |
| Q3 | 2,000 | $220 | $660 |
| Q4 | 5,000 | $270 | $810 |
| **Total** | - | - | **$2,358** |

### Year 2 (Growth)

| Quarter | Users | Monthly Cost | Quarterly Cost |
|---------|-------|--------------|----------------|
| Q1 | 10,000 | $295 | $885 |
| Q2 | 25,000 | $400 | $1,200 |
| Q3 | 50,000 | $500 | $1,500 |
| Q4 | 100,000 | $677 | $2,031 |
| **Total** | - | - | **$5,616** |

## Recommendations

### Immediate (Month 1-3)

1. **Start with minimum configuration**:
   - 2 nodes
   - Cloudflare Free tier
   - Turso Pro tier
   - Self-hosted monitoring
   - **Cost**: ~$126/month

2. **Monitor usage closely**:
   - Set up billing alerts
   - Track cost per user
   - Identify optimization opportunities

### Short-term (Month 3-6)

3. **Optimize based on data**:
   - Right-size node types
   - Implement spot instances
   - Optimize queries and caching
   - **Target**: < $200/month for first 5K users

### Medium-term (Month 6-12)

4. **Scale efficiently**:
   - Use reserved capacity for baseline
   - Aggressive CDN caching
   - Consider Cloud Run for spiky workloads
   - **Target**: < $400/month for first 50K users

### Long-term (Year 2+)

5. **Enterprise optimizations**:
   - Multi-region deployment
   - Committed use discounts
   - Custom pricing negotiations
   - **Target**: < $1000/month for 100K users

## Summary

**Starting Cost**: $126-207/month
**Target Efficiency**: $0.002-0.005 per active user/month
**Break-even**: ~200K requests/day vs. serverless
**Scaling**: Linear up to 100K users, then economies of scale

**Key Cost Drivers**:
1. Compute (60% of cost)
2. Database (15% of cost)
3. Bandwidth (10% of cost)
4. Monitoring (10% of cost)
5. Other (5% of cost)

**Optimization Priority**:
1. Use spot instances (save $60/month)
2. Aggressive CDN caching (save $100/month at scale)
3. Self-host monitoring (save $70/month)
4. Right-size resources (save $30/month)
5. Reserved capacity (save $50/month)

**Total Savings Potential**: $310/month (60% reduction)
