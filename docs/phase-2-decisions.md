# Phase 2: Individual Tier Backend - Decision Log

## Overview
Record of major architectural and strategic decisions made during Phase 2 planning and implementation. This log ensures decisions are documented with rationale and can be revisited if circumstances change.

**Phase:** Phase 2 - Weeks 5-12
**Last Updated:** 2025-10-23
**Status:** Active

---

## Decision Template

Each decision includes:
- **Decision ID:** Unique identifier
- **Date:** When decision was made
- **Status:** Proposed | Approved | Implemented | Superseded
- **Decision:** What was decided
- **Context:** Why this decision was needed
- **Options Considered:** Alternatives evaluated
- **Decision:** Final choice
- **Rationale:** Why this option was chosen
- **Consequences:** Positive and negative impacts
- **Owner:** Who made/owns the decision
- **Review Date:** When to revisit

---

## Authentication Decisions

### D2-001: Auth Provider Selection

**Date:** 2025-10-23
**Status:** Approved
**Owner:** Backend Developer / Tech Lead

**Context:**
Need authentication provider for user registration, login, OAuth, and session management.

**Options Considered:**

| Option | Pros | Cons | Cost |
|--------|------|------|------|
| **Supabase Auth** | 100K MAU, includes Postgres DB, good docs | Vendor lock-in | $25/mo |
| Clerk | Beautiful UI, great DX | Only 10K MAU | $25/mo |
| Auth0 | Industry standard, robust | Free tier only 7.5K MAU, expensive scaling | $35/mo |
| Custom (JWT + Postgres) | Full control, no vendor lock-in | Complex, security risks, time-consuming | $0 + dev time |

**Decision:** Use Supabase Auth

**Rationale:**
1. **Best value:** $25/mo for 100K MAU (10x better than Clerk)
2. **Bonus database:** Includes Postgres (could use for metadata)
3. **Complete feature set:** Email, OAuth, JWT, sessions all built-in
4. **Good developer experience:** Excellent documentation, active community
5. **Security:** SOC2 compliant, battle-tested
6. **Time to market:** Weeks faster than custom auth

**Consequences:**
- ✓ Positive: Faster implementation (1 week vs 3+ weeks)
- ✓ Positive: Reliable, secure, maintained by experts
- ✓ Positive: Scales to 100K users without upgrade
- ✗ Negative: Vendor lock-in (but migration possible if needed)
- ✗ Negative: $25/mo cost (acceptable given value)

**Review Date:** After Phase 2 (if issues arise)

---

### D2-002: OAuth Providers

**Date:** 2025-10-23
**Status:** Approved
**Owner:** Backend Developer

**Context:**
Which OAuth providers to support for social login?

**Options Considered:**
- GitHub (developer-focused)
- Google (broad appeal)
- Microsoft (enterprise)
- Apple (iOS users)
- Twitter/X
- LinkedIn

**Decision:** GitHub + Google only (Phase 2)

**Rationale:**
1. **GitHub:** Target audience is developers, GitHub is essential
2. **Google:** Broadest reach for non-GitHub users
3. **Simplicity:** Two providers sufficient for beta
4. **Implementation time:** Each provider takes 4-8 hours
5. **Future:** Can add Microsoft, Apple in Phase 3+ if demand

**Consequences:**
- ✓ Positive: Faster implementation
- ✓ Positive: Covers 90%+ of beta users
- ✗ Negative: Some users may prefer Microsoft/Apple
- ✗ Negative: Need to add more providers later if requested

**Review Date:** Post-beta feedback (Week 13+)

---

## Sync Architecture Decisions

### D2-003: Sync Conflict Resolution Strategy

**Date:** 2025-10-23
**Status:** Approved
**Owner:** Senior Developer

**Context:**
How to resolve conflicts when same entity modified on multiple devices?

**Options Considered:**

| Strategy | Pros | Cons | Complexity |
|----------|------|------|------------|
| **Last-Write-Wins (LWW)** | Simple, automatic, no user intervention | Can lose data if timestamps close | Low |
| Three-Way Merge | Preserves both changes | Complex, requires base version | High |
| Manual Resolution | User has full control | Interrupts workflow, confusing | Medium |
| Operational Transform | Real-time, no conflicts | Very complex, overkill | Very High |

**Decision:** Last-Write-Wins (LWW) with conflict logging

**Rationale:**
1. **Simplicity:** Easy to implement and understand
2. **Automatic:** No user interruption
3. **Good enough:** Works for 99% of cases (rare concurrent edits)
4. **Logged:** Conflicts logged for audit, can add manual resolution later
5. **Proven:** Used by Dropbox, Google Docs (for some conflicts)

**Consequences:**
- ✓ Positive: Fast, simple implementation
- ✓ Positive: No user confusion
- ✓ Positive: Can upgrade to three-way merge in Phase 3 if needed
- ✗ Negative: Rare data loss if concurrent edits
- ✗ Negative: May surprise users if their change overwritten

**Mitigation:**
- Show notification when conflict auto-resolved
- Provide conflict history view
- Add "Restore previous version" option

**Review Date:** After Week 8 testing

---

### D2-004: Sync Frequency

**Date:** 2025-10-23
**Status:** Approved
**Owner:** Senior Developer

**Context:**
How often should different entities sync?

**Options Considered:**
- Real-time (WebSocket for everything)
- Frequent (every 5 seconds)
- Moderate (every 30 seconds)
- Infrequent (every 5 minutes)
- Manual only

**Decision:** Hybrid approach

| Entity Type | Sync Trigger | Reason |
|-------------|-------------|--------|
| Query tabs (content) | 2s debounce after change | Balance responsiveness & efficiency |
| Connections | Immediate | Infrequent changes, important |
| Saved queries | Immediate | Infrequent changes |
| Query history | Batch every 10 queries | High volume, less critical |
| UI preferences | On change (debounced 1s) | Infrequent |
| Background sync | Every 5 minutes | Non-critical data |

**Rationale:**
1. **User experience:** Critical changes feel instant
2. **Efficiency:** Debouncing reduces unnecessary syncs
3. **Cost optimization:** Fewer row writes to Turso
4. **Battery:** Less background activity

**Consequences:**
- ✓ Positive: Responsive for important changes
- ✓ Positive: Cost-effective (95% fewer writes)
- ✓ Positive: Better battery life
- ✗ Negative: 2s delay for tab content (acceptable)

**Review Date:** After Week 9 optimization

---

### D2-005: Offline Queue Size Limit

**Date:** 2025-10-23
**Status:** Approved
**Owner:** Senior Developer

**Context:**
How many operations to queue when offline?

**Options Considered:**
- Unlimited (risk of memory issues)
- 100 operations
- 500 operations
- 1,000 operations
- 5,000 operations

**Decision:** 1,000 operations with warning at 500

**Rationale:**
1. **Sufficient:** Typical user generates <100 ops/day
2. **Safety:** Prevents memory exhaustion
3. **Warning:** Alert user at 50% to go online
4. **Storage:** ~1MB for 1,000 operations (acceptable)

**Consequences:**
- ✓ Positive: Handles multi-day offline scenarios
- ✓ Positive: Prevents app crash from memory
- ✗ Negative: Could lose changes if limit exceeded (rare)

**Mitigation:**
- Show "Offline queue: 500/1000" warning
- Encourage user to go online and sync
- Provide "Export queue" as backup

**Review Date:** After Week 10 testing

---

## Database Decisions

### D2-006: Turso vs Alternatives

**Date:** 2025-10-23
**Status:** Approved
**Owner:** Backend Developer / Tech Lead

**Context:**
Which cloud database for sync backend?

**Options Considered:**

| Option | Pros | Cons | Cost |
|--------|------|------|------|
| **Turso** | Edge replicas, libSQL, great pricing | Relatively new | $29/mo |
| Supabase Postgres | Mature, full SQL, real-time | No edge, expensive | $25-125/mo |
| PlanetScale | MySQL, great DX | Expensive ($39+), MySQL | $39/mo |
| MongoDB Atlas | Flexible schema | Not relational, overkill | $57/mo |
| Firebase Firestore | Real-time, Google | NoSQL, vendor lock-in | Pay-as-go |
| Custom Postgres | Full control | Ops burden, no edge | $50+/mo |

**Decision:** Turso

**Rationale:**
1. **Edge replication:** Low latency globally
2. **Cost:** $29/mo for 500M rows (excellent value)
3. **SQLite compatibility:** Easy to work with
4. **Embedded replicas:** Offline support built-in
5. **Pricing model:** Rows written (aligns with our usage)

**Consequences:**
- ✓ Positive: Excellent performance globally
- ✓ Positive: Cost-effective at scale
- ✓ Positive: SQLite = simple, familiar
- ✗ Negative: Newer product (less battle-tested)
- ✗ Negative: libSQL limitations vs Postgres

**Review Date:** After Week 6 implementation

---

### D2-007: Tenant Isolation Strategy

**Date:** 2025-10-23
**Status:** Approved
**Owner:** Backend Developer

**Context:**
How to isolate user data in Turso?

**Options Considered:**
1. **Shared database with app-level RLS:** All users in one DB, filter by user_id
2. **Database per user:** Separate Turso DB per user
3. **Database per tier:** Free tier DB, Individual tier DB, Team tier DB

**Decision:** Shared database with app-level RLS

**Rationale:**
1. **Cost:** One $29/mo database serves unlimited users
2. **Simplicity:** No complex provisioning
3. **Migrations:** Single schema to maintain
4. **Sufficient security:** App enforces WHERE user_id = ?
5. **Industry standard:** Most SaaS uses this approach

**Consequences:**
- ✓ Positive: Cost-effective
- ✓ Positive: Simple to manage
- ✓ Positive: Easy backups (single DB)
- ✗ Negative: Must be careful with WHERE clauses (security)
- ✗ Negative: All users impacted if DB issue

**Mitigation:**
- Code review every query for user_id filter
- Automated tests for data isolation
- Prepared statements (prevent injection)

**Review Date:** After Phase 2 security audit

---

## Pricing & Business Decisions

### D2-008: Individual Tier Pricing

**Date:** 2025-10-23
**Status:** Approved
**Owner:** Product Manager / CEO

**Context:**
What price for Individual tier?

**Options Considered:**

| Price | Pros | Cons | Positioning |
|-------|------|------|-------------|
| $5/mo | Very competitive | Low margin, perceived as cheap | Budget option |
| $7/mo | Competitive | Still low-ish | Value option |
| **$9/mo** | **Sweet spot** | **Slightly above middle** | **Premium value** |
| $12/mo | Higher margin | May reduce conversions | Premium |
| $15/mo | Best margin | Too expensive vs competitors | Luxury |

**Decision:** $9/month (or $90/year with 17% discount)

**Rationale:**
1. **Market research:** TablePlus $89 one-time, DBeaver $30/mo
2. **Value proposition:** Only tool with full cloud sync
3. **Margin:** 91.8% gross margin at $9/mo
4. **Psychology:** $9 feels like single-digit (vs $10+)
5. **Annual discount:** $90/year = $7.50/mo (motivates annual)

**Consequences:**
- ✓ Positive: Competitive pricing
- ✓ Positive: Excellent margins
- ✓ Positive: Room to discount (sales, promotions)
- ✗ Negative: Could be higher (but reduces conversions)

**Review Date:** After beta feedback (Week 13)

---

### D2-009: Trial Duration

**Date:** 2025-10-23
**Status:** Approved
**Owner:** Product Manager

**Context:**
How long should free trial be?

**Options Considered:**
- 7 days (industry standard for low-ticket)
- **14 days** (common for B2B SaaS)
- 30 days (generous, but too long?)
- No trial (free tier only)

**Decision:** 14-day free trial

**Rationale:**
1. **Sufficient:** Enough time to evaluate value
2. **Industry standard:** Common for $9-$50/mo products
3. **Urgency:** Not so long users forget
4. **No credit card:** Can be no-CC trial (easier signups)

**Consequences:**
- ✓ Positive: Lowers signup friction
- ✓ Positive: Gives users confidence
- ✗ Negative: Some may abuse (create multiple accounts)

**Mitigation:**
- Track email/device fingerprint to prevent abuse
- Clearly communicate trial end date

**Review Date:** Post-beta (adjust if conversion low)

---

## Technical Decisions

### D2-010: Programming Language for Backend

**Date:** 2025-10-23
**Status:** Approved (inherited from Phase 1)
**Owner:** Tech Lead

**Context:**
Backend already in Go (Wails). Continue or switch?

**Decision:** Continue with Go

**Rationale:**
1. **Consistency:** Frontend (Wails) already Go
2. **Performance:** Go is fast, efficient
3. **Team expertise:** Team knows Go
4. **No migration:** Switching wastes time
5. **Libraries:** Good Go libs for Turso, Supabase, Stripe

**Consequences:**
- ✓ Positive: No migration time/risk
- ✓ Positive: Unified codebase
- ✗ Negative: None (Go is excellent for this)

**Review Date:** N/A (not changing)

---

### D2-011: WebSocket vs Polling for Real-Time Sync

**Date:** 2025-10-23
**Status:** Approved
**Owner:** Senior Developer

**Context:**
How to push changes to clients in real-time?

**Options Considered:**
1. **WebSocket:** Persistent connection, instant push
2. **Polling:** Client checks for changes every N seconds
3. **Server-Sent Events (SSE):** One-way server push
4. **No real-time:** Pull sync every 30s only

**Decision:** WebSocket for real-time updates

**Rationale:**
1. **User experience:** Instant updates feel magical
2. **Efficiency:** No repeated polling overhead
3. **Battery:** Less network activity than polling
4. **Scalability:** 10K+ concurrent connections feasible
5. **Fallback:** Can fall back to polling if WebSocket fails

**Consequences:**
- ✓ Positive: Excellent UX
- ✓ Positive: Efficient
- ✓ Positive: Scalable
- ✗ Negative: More complex than polling
- ✗ Negative: Requires WebSocket infrastructure

**Review Date:** After Week 8 implementation

---

### D2-012: Client-Side Encryption

**Date:** 2025-10-23
**Status:** Deferred to Phase 3
**Owner:** Security Specialist

**Context:**
Should we encrypt query content before syncing?

**Options Considered:**
1. **No encryption:** Trust Turso's encryption at rest
2. **Client-side encryption:** Encrypt before sync
3. **Optional encryption:** User choice

**Decision:** Defer to Phase 3, start without client-side encryption

**Rationale:**
1. **Turso encryption:** Already encrypted at rest (AES-256)
2. **TLS:** Encrypted in transit (TLS 1.3)
3. **Complexity:** Client-side encryption adds complexity
4. **Key management:** Where to store encryption key?
5. **Trade-off:** Search, dedup require plaintext server-side
6. **User demand:** Assess demand in beta first

**Consequences:**
- ✓ Positive: Simpler implementation (Phase 2)
- ✓ Positive: Can add later if users want it
- ✗ Negative: Turso could theoretically read queries
- ✗ Negative: Regulatory concerns for some users

**Mitigation:**
- Offer query redaction (Phase 2)
- Document privacy approach clearly
- Add client-side encryption in Phase 3 if needed

**Review Date:** Post-beta feedback

---

## Superseded Decisions

### D2-000: Auth Provider (Superseded)

**Original Decision:** Use Clerk
**Date:** 2025-10-15
**Superseded By:** D2-001 (Supabase)
**Superseded Date:** 2025-10-23

**Reason for Change:**
Discovered Supabase offers 100K MAU vs Clerk's 10K MAU for same price. Better value.

---

## Decision-Making Process

### Who Decides?

| Decision Type | Decision Maker | Consultation |
|---------------|----------------|--------------|
| Architecture | Tech Lead | Senior Developers |
| Technology Choice | Tech Lead + PM | Team |
| Pricing | CEO + PM | Finance |
| Feature Scope | PM | CEO, Tech Lead |
| Design/UX | UI Designer | PM, Users |
| Security | Security Specialist | Tech Lead |

### Decision Template

1. **Identify Decision:** What needs deciding?
2. **Gather Options:** Research alternatives
3. **Evaluate:** Pros/cons, cost, risk
4. **Consult:** Get input from stakeholders
5. **Decide:** Make call, document rationale
6. **Communicate:** Share decision with team
7. **Review:** Set review date

---

## Pending Decisions

### PD-001: Custom Auth UI vs Provider UI

**Status:** Pending research
**Owner:** Frontend Developer
**Deadline:** Week 5, Day 2

**Question:** Use Supabase Auth UI components or build custom?

**Options:**
- Supabase Auth UI (faster, consistent)
- Custom UI (branded, flexible)

**Next Steps:** Research Supabase Auth UI customization

---

### PD-002: Background Sync Workers

**Status:** Pending implementation
**Owner:** Senior Developer
**Deadline:** Week 9

**Question:** Use Web Workers or main thread for background sync?

**Options:**
- Web Worker (non-blocking, better performance)
- Main thread (simpler, sufficient?)

**Next Steps:** Prototype both approaches

---

## Document Metadata

**Version:** 1.0
**Status:** Active
**Last Updated:** 2025-10-23
**Next Review:** Weekly during Phase 2
**Owner:** Tech Lead / Product Manager

**Change Log:**
| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-10-23 | 1.0 | Initial decision log | PM Agent |

---

## References

- [Phase 2 Tasks](./phase-2-tasks.md)
- [Phase 2 Tech Specs](./phase-2-tech-specs.md)
- [Phase 2 Costs](./phase-2-costs.md)
- [Turso Integration Design](./turso-integration-design.md)

---

**Instructions for Use:**
1. Document all major decisions as they're made
2. Include rationale (for future reference)
3. Set review dates (decisions can change)
4. Communicate decisions to entire team
5. Update status as decisions evolve
6. Reference decisions in PR descriptions

**When to Log a Decision:**
- Choosing between technologies
- Architectural patterns
- Pricing/business model
- Security approach
- Trade-offs with significant impact
