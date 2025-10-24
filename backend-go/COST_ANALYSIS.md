# SQL Studio Backend - Cost Analysis & Comparison

Comprehensive cost analysis for deploying SQL Studio backend on Google Cloud Run vs Fly.io.

**Last Updated:** October 2025
**Analysis Period:** Monthly costs based on actual 2025 pricing

---

## Executive Summary

| Deployment Option | Best For | Monthly Cost (Low) | Monthly Cost (Medium) | Monthly Cost (High) |
|-------------------|----------|-------------------|----------------------|---------------------|
| **GCP Cloud Run** | Production, high traffic | $0-5 | $10-50 | $100-500 |
| **Fly.io** | Hobby, low-cost startups | $0-2 | $5-15 | $30-100 |

**Recommendation:**
- **Personal/Hobby Projects:** Fly.io (better free tier, simpler billing)
- **Production/Business:** GCP Cloud Run (better scalability, SLA, monitoring)
- **Global Applications:** GCP Cloud Run (superior global infrastructure)

---

## Cost Breakdown by Platform

### Google Cloud Run

#### Pricing Components (as of 2025)

**1. CPU Allocation**
- **Price:** $0.00002400 per vCPU-second
- **Free tier:** 180,000 vCPU-seconds/month
- **Calculation:** Only charged during request processing (with CPU throttling)

**2. Memory Allocation**
- **Price:** $0.00000250 per GB-second
- **Free tier:** 360,000 GB-seconds/month
- **Calculation:** Only charged during request processing (with CPU throttling)

**3. Requests**
- **Price:** $0.40 per million requests
- **Free tier:** 2 million requests/month
- **Calculation:** Every HTTP request counts

**4. Networking (Egress)**
- **Price:**
  - $0.12/GB (North America)
  - $0.15/GB (Asia)
  - $0.19/GB (Australia)
- **Free tier:** None for egress
- **Note:** Ingress is free

#### Resource Configuration

**Recommended Config:**
```yaml
CPU: 1 vCPU
Memory: 512 MB
Concurrency: 80 requests
Min instances: 0 (scale to zero)
Max instances: 10
```

#### Cost Scenarios

##### Scenario 1: Hobby Project (Low Traffic)
- **Traffic:** 50,000 requests/month
- **Avg response time:** 200ms
- **Data transfer:** 5 GB/month

**Calculations:**
```
CPU time: 50,000 requests × 0.2s = 10,000 vCPU-seconds
Memory time: 50,000 × 0.2s × 0.5GB = 5,000 GB-seconds
Requests: 50,000 requests

Costs:
- CPU: 10,000 vCPU-seconds (FREE - under 180k)
- Memory: 5,000 GB-seconds (FREE - under 360k)
- Requests: 50,000 (FREE - under 2M)
- Egress: 5GB × $0.12 = $0.60

Total: $0.60/month
```

##### Scenario 2: Small Business (Medium Traffic)
- **Traffic:** 1,000,000 requests/month
- **Avg response time:** 250ms
- **Data transfer:** 50 GB/month

**Calculations:**
```
CPU time: 1,000,000 × 0.25s = 250,000 vCPU-seconds
Memory time: 1,000,000 × 0.25s × 0.5GB = 125,000 GB-seconds
Requests: 1,000,000

Costs:
- CPU: (250,000 - 180,000) × $0.000024 = $1.68
- Memory: 125,000 GB-seconds (FREE - under 360k)
- Requests: 1,000,000 (FREE - under 2M)
- Egress: 50GB × $0.12 = $6.00

Total: $7.68/month
```

##### Scenario 3: Growing Startup (High Traffic)
- **Traffic:** 10,000,000 requests/month
- **Avg response time:** 300ms
- **Data transfer:** 500 GB/month

**Calculations:**
```
CPU time: 10,000,000 × 0.3s = 3,000,000 vCPU-seconds
Memory time: 10,000,000 × 0.3s × 0.5GB = 1,500,000 GB-seconds
Requests: 10,000,000

Costs:
- CPU: (3,000,000 - 180,000) × $0.000024 = $67.68
- Memory: (1,500,000 - 360,000) × $0.0000025 = $2.85
- Requests: (10,000,000 - 2,000,000) × $0.0000004 = $3.20
- Egress: 500GB × $0.12 = $60.00

Total: $133.73/month
```

##### Scenario 4: Enterprise (Very High Traffic)
- **Traffic:** 100,000,000 requests/month
- **Avg response time:** 200ms
- **Data transfer:** 5 TB/month
- **Min instances:** 2 (always running)

**Calculations:**
```
CPU time: 100,000,000 × 0.2s = 20,000,000 vCPU-seconds
Memory time: 100,000,000 × 0.2s × 0.5GB = 10,000,000 GB-seconds
Idle time (2 instances): 2 × 30 days × 86400s = 5,184,000s
Requests: 100,000,000

Costs:
- CPU (requests): 20,000,000 × $0.000024 = $480
- CPU (idle): 2 instances × 1 vCPU × 5,184,000s × $0.000024 = $248.83
- Memory (requests): 10,000,000 × $0.0000025 = $25
- Memory (idle): 2 × 0.5GB × 5,184,000s × $0.0000025 = $12.96
- Requests: 100,000,000 × $0.0000004 = $40
- Egress: 5,000GB × $0.12 = $600

Total: $1,406.79/month
```

---

### Fly.io

#### Pricing Components (as of 2025)

**1. Compute (VMs)**

| VM Type | vCPU | Memory | Price/month (always on) |
|---------|------|--------|------------------------|
| shared-cpu-1x | 1 | 256 MB | $1.94 |
| shared-cpu-1x | 1 | 512 MB | $3.47 |
| shared-cpu-1x | 1 | 1 GB | $5.70 |
| shared-cpu-2x | 2 | 512 MB | $6.94 |
| shared-cpu-2x | 2 | 1 GB | $11.40 |

**Scale-to-Zero Pricing:**
- Stopped VMs: $0.15/GB RAM per month
- 512 MB stopped: $0.075/month
- 1 GB stopped: $0.15/month

**2. Bandwidth**
- **Included:** 100 GB/month (free tier)
- **Additional:** $0.02/GB
- **Note:** Ingress is unlimited and free

**3. Free Tier**
- Up to 3 shared-cpu-1x VMs (256MB each)
- 160 GB outbound data transfer
- No time limits

#### Resource Configuration

**Recommended Config:**
```toml
CPU: 1 shared vCPU
Memory: 512 MB
Min machines: 0 (scale to zero)
Auto-start: true
```

#### Cost Scenarios

##### Scenario 1: Hobby Project (Low Traffic)
- **Traffic:** 50,000 requests/month
- **Uptime:** ~5 hours/month (due to scale-to-zero)
- **Data transfer:** 5 GB/month

**Calculations:**
```
VM (scale-to-zero, 512MB): $0.075/month (stopped state)
Running time: 5 hours × $3.47/730h = $0.02
Bandwidth: 5GB (FREE - under 100GB)

Total: $0.10/month
```

##### Scenario 2: Small Business (Medium Traffic)
- **Traffic:** 1,000,000 requests/month
- **Uptime:** ~200 hours/month (27% uptime)
- **Data transfer:** 50 GB/month

**Calculations:**
```
VM running: 200h × $3.47/730h = $0.95
VM stopped: 530h × $0.075/730h = $0.05
Bandwidth: 50GB (FREE - under 100GB)

Total: $1.00/month
```

##### Scenario 3: Growing Startup (High Traffic)
- **Traffic:** 10,000,000 requests/month
- **Uptime:** Always on (1 instance)
- **Additional instances:** 1 (during peak)
- **Data transfer:** 500 GB/month

**Calculations:**
```
Primary VM (512MB, always on): $3.47
Secondary VM (512MB, 50% uptime): 365h × $3.47/730h = $1.73
Bandwidth: (500GB - 100GB) × $0.02 = $8.00

Total: $13.20/month
```

##### Scenario 4: Enterprise (Very High Traffic)
- **Traffic:** 100,000,000 requests/month
- **Uptime:** Always on (3 instances minimum)
- **Peak scaling:** +2 instances
- **Data transfer:** 5 TB/month

**Calculations:**
```
3 VMs (1GB each, always on): 3 × $5.70 = $17.10
2 Peak VMs (1GB, 50% uptime): 2 × 365h × $5.70/730h = $5.70
Bandwidth: (5,000GB - 100GB) × $0.02 = $98.00

Total: $120.80/month
```

---

## Side-by-Side Comparison

### Low Traffic (Hobby Project)

| Metric | GCP Cloud Run | Fly.io | Winner |
|--------|---------------|---------|---------|
| Monthly requests | 50,000 | 50,000 | - |
| Monthly cost | $0.60 | $0.10 | **Fly.io** |
| Setup complexity | Medium | Low | **Fly.io** |
| Scale-to-zero | Yes | Yes | Tie |
| Cold start time | ~200ms | ~500ms | **GCP** |
| Free tier coverage | 100% | 100% | Tie |

**Winner:** **Fly.io** - Lower cost, simpler setup

---

### Medium Traffic (Small Business)

| Metric | GCP Cloud Run | Fly.io | Winner |
|--------|---------------|---------|---------|
| Monthly requests | 1M | 1M | - |
| Monthly cost | $7.68 | $1.00 | **Fly.io** |
| Uptime SLA | 99.95% | 99.9% | **GCP** |
| Monitoring | Advanced | Basic | **GCP** |
| Global CDN | Yes | Yes | Tie |
| Support | Email/Chat | Community | **GCP** |

**Winner:** **Fly.io** for cost, **GCP** for reliability

---

### High Traffic (Growing Startup)

| Metric | GCP Cloud Run | Fly.io | Winner |
|--------|---------------|---------|---------|
| Monthly requests | 10M | 10M | - |
| Monthly cost | $133.73 | $13.20 | **Fly.io** |
| Auto-scaling | Excellent | Good | **GCP** |
| Max instances | 1000 | 100 | **GCP** |
| Global regions | 35+ | 30+ | **GCP** |
| DDoS protection | Cloud Armor | Built-in | **GCP** |

**Winner:** **Fly.io** for cost, **GCP** for scale

---

### Enterprise (Very High Traffic)

| Metric | GCP Cloud Run | Fly.io | Winner |
|--------|---------------|---------|---------|
| Monthly requests | 100M | 100M | - |
| Monthly cost | $1,406.79 | $120.80 | **Fly.io** |
| Enterprise SLA | 99.95% | Custom | **GCP** |
| Compliance | SOC 2, ISO | SOC 2 | **GCP** |
| VPC support | Yes | Yes | Tie |
| Dedicated support | Yes | Yes | Tie |

**Winner:** **Fly.io** for cost, **GCP** for compliance

---

## Hidden Costs & Considerations

### Google Cloud Run

**Additional Costs:**
- **Cloud Build:** $0.003/build-minute (first 120 minutes/day free)
- **Container Registry:** $0.026/GB storage
- **Secret Manager:** $0.06 per 10,000 accesses
- **Load Balancer (if using custom domain):** $18/month + $0.008/GB
- **Cloud Armor (DDoS protection):** $5/policy + $1/million requests

**Hidden Benefits:**
- Free SSL certificates
- Automatic CDN caching
- Integration with GCP ecosystem (Firestore, Cloud SQL, etc.)
- Advanced IAM and security controls

### Fly.io

**Additional Costs:**
- **Persistent Volumes:** $0.15/GB/month (if needed)
- **IPv4 addresses:** Free (1 per app)
- **Additional IPv4:** $2/month each
- **Anycast private network:** Free
- **Log retention:** Free (30 days)

**Hidden Benefits:**
- Built-in global Anycast network
- Free Postgres database (3GB)
- Free Redis (256MB)
- SSH access to machines
- No charge for stopped machines (only $0.15/GB RAM)

---

## Break-Even Analysis

### When Fly.io is Cheaper

Fly.io is more cost-effective when:
- **Requests/month:** < 50 million
- **Always-on instances:** < 5
- **Data transfer:** < 2 TB/month
- **Budget:** < $100/month

### When GCP Cloud Run is Cheaper

GCP becomes competitive when:
- **High burst traffic** with long idle periods (scale-to-zero)
- **Very low traffic** (both are nearly free)
- **Need automatic global CDN** (saves on bandwidth costs)

### Crossover Point

At approximately **30-50 million requests/month** with moderate data transfer, the costs become similar. Beyond this, pricing depends heavily on:
- Request processing time
- Memory usage
- Data transfer patterns
- Number of always-on instances

---

## Cost Optimization Strategies

### For GCP Cloud Run

1. **Enable CPU Throttling**
   ```bash
   gcloud run services update sql-studio-backend --cpu-throttling
   ```
   Saves ~50% on CPU costs

2. **Scale to Zero**
   ```bash
   gcloud run services update sql-studio-backend --min-instances=0
   ```
   Eliminates idle costs

3. **Right-Size Resources**
   - Start with 256MB RAM, scale up only if needed
   - Monitor and adjust based on actual usage

4. **Use Committed Use Discounts**
   - For predictable traffic, commit to 1 or 3 years
   - Saves up to 57% on compute costs

5. **Optimize Response Size**
   - Enable compression
   - Minimize response payloads
   - Reduces egress bandwidth costs

### For Fly.io

1. **Scale to Zero**
   ```toml
   min_machines_running = 0
   auto_stop_machines = "stop"
   ```
   Only pay $0.15/GB RAM when idle

2. **Use Smallest VM Possible**
   - Start with 256MB, increase only if needed
   - 256MB vs 512MB saves $1.53/month per VM

3. **Single Region (if possible)**
   - Multiple regions multiply VM costs
   - Use Fly's Anycast for global reach without multiple VMs

4. **Monitor Bandwidth**
   - First 100GB free
   - Optimize payload sizes to stay under limit

5. **Use Free Postgres/Redis**
   - Included with Fly.io
   - Saves on external database costs

---

## Total Cost of Ownership (TCO)

### 1-Year TCO for Small Business (1M req/month)

| Component | GCP Cloud Run | Fly.io |
|-----------|---------------|---------|
| Compute | $92 | $12 |
| Bandwidth | $72 | $0 |
| Database (Turso) | $0 | $0 |
| Email (Resend) | $0 | $0 |
| Monitoring | $0 (free tier) | $0 |
| **Total Year 1** | **$164** | **$12** |

### 1-Year TCO for Growing Startup (10M req/month)

| Component | GCP Cloud Run | Fly.io |
|-----------|---------------|---------|
| Compute | $813 | $62 |
| Bandwidth | $792 | $96 |
| Database (Turso) | $29/month × 12 | $29/month × 12 |
| Email (Resend) | $20/month × 12 | $20/month × 12 |
| Monitoring | $50/month × 12 | $0 |
| **Total Year 1** | **$2,793** | **$746** |

### 3-Year TCO Comparison

| Traffic Level | GCP Cloud Run (3yr) | Fly.io (3yr) | Savings with Fly.io |
|---------------|---------------------|--------------|---------------------|
| Low (50k/mo) | $22 | $4 | $18 (82%) |
| Medium (1M/mo) | $277 | $36 | $241 (87%) |
| High (10M/mo) | $4,814 | $475 | $4,339 (90%) |
| Very High (100M/mo) | $50,644 | $4,349 | $46,295 (91%) |

---

## Recommendation Matrix

### Choose GCP Cloud Run if:

✅ You need enterprise SLA (99.95% uptime)
✅ You require advanced security (Cloud Armor, VPC Service Controls)
✅ You have extreme burst traffic patterns
✅ You're already using GCP services (Firestore, Cloud SQL, etc.)
✅ You need SOC 2, ISO 27001, HIPAA compliance
✅ You want auto-scaling to 1000+ instances
✅ You need advanced monitoring and logging (Cloud Logging/Monitoring)
✅ You have budget for premium services ($100+/month)

### Choose Fly.io if:

✅ You're cost-sensitive or bootstrapped
✅ You want simple, predictable pricing
✅ You have moderate traffic (< 50M requests/month)
✅ You want SSH access to production machines
✅ You prefer simplicity over advanced features
✅ You want to minimize DevOps complexity
✅ You're building a side project or MVP
✅ Budget is under $100/month

---

## Real-World Cost Examples

### Example 1: Personal Blog API
- **Traffic:** 100,000 requests/month
- **Data:** 10 GB/month
- **GCP Cost:** $1.20/month
- **Fly.io Cost:** $0.15/month
- **Winner:** Fly.io (saves $1.05/month, $12.60/year)

### Example 2: SaaS Startup (Beta)
- **Traffic:** 5M requests/month
- **Data:** 200 GB/month
- **GCP Cost:** $67/month
- **Fly.io Cost:** $8/month
- **Winner:** Fly.io (saves $59/month, $708/year)

### Example 3: Established SaaS
- **Traffic:** 50M requests/month
- **Data:** 2 TB/month
- **GCP Cost:** $670/month
- **Fly.io Cost:** $68/month
- **Winner:** Fly.io (saves $602/month, $7,224/year)

### Example 4: Enterprise API
- **Traffic:** 200M requests/month
- **Data:** 10 TB/month
- **High availability:** 3 regions
- **GCP Cost:** $3,500/month (with premium support)
- **Fly.io Cost:** $500/month
- **Winner:** Depends on compliance/SLA needs

---

## Conclusion

### Cost Summary

For most SQL Studio deployments, **Fly.io is 80-90% cheaper** than GCP Cloud Run, especially for consistent traffic patterns. However, GCP Cloud Run offers superior:
- Enterprise features
- Global scalability
- Advanced monitoring
- Compliance certifications

### Final Recommendation

**Start with Fly.io** for:
- MVP and early-stage development
- Hobbyist projects
- Bootstrapped startups
- Budget < $100/month

**Migrate to GCP Cloud Run** when:
- Monthly budget exceeds $500
- Require enterprise SLA/compliance
- Need extreme scalability (100M+ requests/month)
- Already invested in GCP ecosystem

### Hybrid Approach

Consider using **both**:
- **Fly.io:** Primary deployment (low cost)
- **GCP Cloud Run:** Backup/failover (reliability)
- **DNS failover:** Switch between them automatically

This provides:
- Cost savings of Fly.io
- Reliability of GCP Cloud Run
- Geographic redundancy
- Risk mitigation

---

**Cost Calculator Tool:** https://cloud.google.com/products/calculator
**Fly.io Pricing:** https://fly.io/docs/about/pricing/
**Turso Pricing:** https://turso.tech/pricing
**Resend Pricing:** https://resend.com/pricing

---

**Disclaimer:** All prices are estimates based on October 2025 pricing and may change. Always verify current pricing on provider websites before making decisions.
