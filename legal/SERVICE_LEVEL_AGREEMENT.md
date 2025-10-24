# Service Level Agreement (SLA)

**Version:** 1.0.0
**Effective Date:** [EFFECTIVE_DATE]
**Last Updated:** January 1, 2025

This Service Level Agreement ("**SLA**") is entered into between SQL Studio Inc. ("**SQL Studio**," "**we**," "**us**," or "**our**") and the customer identified in the applicable Order Form ("**Customer**," "**you**," or "**your**") and governs the availability and performance of the SQL Studio Enterprise Services.

## 1. SERVICE COMMITMENT

SQL Studio is committed to providing reliable, high-performance database management services. This SLA defines our service level objectives and your remedies if we fail to meet them.

### 1.1 Scope

This SLA applies to:
- SQL Studio Enterprise tier subscriptions
- Production environment services
- Core platform functionality
- API services
- Data synchronization services

This SLA does NOT apply to:
- Free tier services
- Professional tier services (unless separately agreed)
- Beta or preview features
- Development or testing environments
- Third-party integrations
- Customer-caused issues

## 2. DEFINITIONS

**"Availability"**: The percentage of time the Service is operational and accessible during a calendar month.

**"Downtime"**: Any period when the Service is unavailable, excluding Excluded Downtime.

**"Emergency Maintenance"**: Urgent maintenance required to prevent or resolve critical issues.

**"Error Rate"**: The percentage of valid requests that result in errors.

**"Excluded Downtime"**: Downtime that doesn't count against availability, as defined in Section 4.3.

**"Incident"**: Any event that causes or may cause Service disruption.

**"Monthly Uptime Percentage"**: (Total Minutes in Month - Downtime Minutes) / Total Minutes in Month × 100

**"Response Time"**: The time between receiving a valid API request and returning the first byte of response.

**"Scheduled Maintenance"**: Planned maintenance notified in advance.

**"Service"**: The SQL Studio Enterprise platform as defined in your Order Form.

**"Service Credit"**: A credit applied to future invoices as remedy for SLA violations.

**"Service Level Objective (SLO)"**: The targeted performance metric.

**"Support Request"**: A request for technical assistance submitted through official channels.

## 3. SERVICE LEVEL OBJECTIVES

### 3.1 Availability Commitment

#### Platform Availability

| Service Component | Monthly Uptime Target | Measurement Method |
|------------------|----------------------|-------------------|
| Core Platform | 99.9% | Synthetic monitoring every 60 seconds |
| API Services | 99.9% | Health check endpoints every 30 seconds |
| Web Application | 99.9% | Page load monitoring every 60 seconds |
| Data Sync Services | 99.5% | Sync job success rate |
| Authentication Service | 99.95% | Login success monitoring |

#### Regional Availability

| Region | Availability Target | Data Centers |
|--------|-------------------|--------------|
| US East | 99.9% | Virginia, New York |
| US West | 99.9% | California, Oregon |
| EU West | 99.9% | Ireland, Frankfurt |
| EU Central | 99.9% | Frankfurt, Amsterdam |
| Asia Pacific | 99.5% | Singapore, Sydney |

### 3.2 Performance Commitments

#### Response Time SLOs

| Metric | Target (P95) | Target (P99) | Measurement |
|--------|------------|-------------|-------------|
| API Response Time | < 200ms | < 500ms | Excluding customer database query time |
| Page Load Time | < 2 seconds | < 4 seconds | Initial page render |
| Query Execution UI | < 100ms | < 250ms | UI response only |
| Data Sync Latency | < 5 seconds | < 30 seconds | End-to-end sync time |

#### Throughput SLOs

| Metric | Commitment | Measurement |
|--------|-----------|-------------|
| Concurrent Users | 10,000 per organization | Active sessions |
| API Requests | 1,000 requests/second | Per organization |
| Query Throughput | 100 queries/second | Per organization |
| Data Transfer | 100 MB/second | Per connection |

### 3.3 Data Durability

| Data Type | Durability Target | Backup Frequency | Retention |
|-----------|------------------|------------------|-----------|
| User Data | 99.999999999% (11 nines) | Continuous replication | 30 days |
| Query History | 99.9999% | Daily | 90 days |
| Configuration | 99.99999% | Hourly | 30 days |
| Audit Logs | 99.9999% | Real-time | 7 years |

### 3.4 Support Response Times

#### Business Hours Support (Monday-Friday, 9 AM - 6 PM Customer Time Zone)

| Priority | First Response | Update Frequency | Resolution Target |
|----------|---------------|------------------|-------------------|
| P1 - Critical | 1 hour | Every 2 hours | 4 hours |
| P2 - High | 4 hours | Every 8 hours | 24 hours |
| P3 - Medium | 24 hours | Every 2 days | 5 business days |
| P4 - Low | 72 hours | Weekly | Best effort |

#### 24/7 Support (Enterprise Plus only)

| Priority | First Response | Update Frequency | Resolution Target |
|----------|---------------|------------------|-------------------|
| P1 - Critical | 15 minutes | Hourly | 2 hours |
| P2 - High | 1 hour | Every 4 hours | 8 hours |
| P3 - Medium | 4 hours | Daily | 48 hours |
| P4 - Low | 24 hours | Every 3 days | Best effort |

## 4. SERVICE CREDITS

### 4.1 Availability Service Credits

If we fail to meet our availability commitment, you are eligible for Service Credits:

| Monthly Uptime Percentage | Service Credit |
|--------------------------|----------------|
| 99.9% - 99.0% | 10% of monthly fee |
| 99.0% - 95.0% | 25% of monthly fee |
| 95.0% - 90.0% | 50% of monthly fee |
| Below 90.0% | 100% of monthly fee |

### 4.2 Performance Service Credits

For performance SLA violations:

| Performance Degradation | Duration | Service Credit |
|------------------------|----------|----------------|
| Response time 2x target | > 1 hour continuous | 5% of monthly fee |
| Response time 5x target | > 15 minutes continuous | 10% of monthly fee |
| Complete service degradation | > 5 minutes | 25% of monthly fee |

### 4.3 Excluded Downtime

The following don't count as Downtime:

#### Scheduled Maintenance
- Notified at least 48 hours in advance
- Performed during maintenance windows
- Limited to 4 hours per month

#### Customer-Caused Issues
- Exceeding API rate limits
- Incorrect configurations
- Network issues on customer side
- Customer database unavailability
- Customer code errors

#### Force Majeure
- Natural disasters
- War, terrorism, or civil unrest
- Government actions
- Labor disputes
- Third-party service failures beyond our control

#### Other Exclusions
- Beta or preview features
- Features explicitly excluded from SLA
- Suspension due to Terms violation
- Suspension for non-payment
- DNS propagation delays

### 4.4 Service Credit Process

#### Credit Request Requirements
1. Submit request within 30 days of incident
2. Include:
   - Incident date and time
   - Affected services
   - Impact description
   - Your monitoring data (if available)
3. Submit via support portal with "SLA Credit Request" subject

#### Credit Processing
- We will investigate within 5 business days
- Credits approved within 10 business days
- Credits applied to next invoice
- Maximum total credits: 100% of monthly fees
- No cash refunds for credits

### 4.5 Credit Limitations

- Credits are your sole remedy for SLA violations
- Credits don't apply to fees already discounted
- Multiple violations in same period don't stack
- Credits cannot exceed monthly service fees
- Unused credits expire after 12 months

## 5. MONITORING AND REPORTING

### 5.1 Service Monitoring

#### Our Monitoring
We continuously monitor:
- Service availability (1-minute intervals)
- API response times
- Error rates
- Resource utilization
- Security events

#### Monitoring Infrastructure
```
┌─────────────────────────────────────────────────────┐
│              Monitoring Architecture                 │
├─────────────────────────────────────────────────────┤
│ External Monitors                                   │
│ ├─ Synthetic transactions every 60 seconds          │
│ ├─ Multiple geographic locations                    │
│ └─ Real user monitoring (RUM)                      │
├─────────────────────────────────────────────────────┤
│ Internal Monitors                                   │
│ ├─ Application performance monitoring (APM)         │
│ ├─ Infrastructure monitoring                       │
│ ├─ Log aggregation and analysis                   │
│ └─ Custom business metrics                        │
├─────────────────────────────────────────────────────┤
│ Alerting                                           │
│ ├─ Automated incident creation                     │
│ ├─ On-call engineer paging                        │
│ └─ Customer notifications                         │
└─────────────────────────────────────────────────────┘
```

### 5.2 Status Page

Public status page available at: https://status.sqlstudio.com

Shows:
- Current system status
- Active incidents
- Scheduled maintenance
- 90-day uptime history
- Performance metrics
- Incident history

### 5.3 Monthly SLA Reports

Available on the 5th of each month, including:

#### Availability Metrics
- Overall uptime percentage
- Per-component availability
- Downtime incidents details
- Excluded downtime justification

#### Performance Metrics
- Response time percentiles (P50, P95, P99)
- Throughput statistics
- Error rates
- API usage statistics

#### Support Metrics
- Ticket response times
- Resolution times by priority
- Customer satisfaction scores
- Escalation statistics

### 5.4 Real-time Dashboards

Enterprise customers have access to:
- Real-time performance metrics
- Custom alerting thresholds
- API for metrics extraction
- Historical data (90 days)

## 6. SUPPORT

### 6.1 Support Channels

| Channel | Availability | Response Time | Use Case |
|---------|-------------|---------------|----------|
| Emergency Hotline | 24/7 | Immediate | P1 issues only |
| Support Portal | 24/7 | Per priority | All issues |
| Email | Business hours | Per priority | Non-urgent |
| Slack (Enterprise Plus) | Business hours | 15 minutes | Quick questions |
| Dedicated TAM | Business hours | 1 hour | Strategic issues |

### 6.2 Priority Definitions

#### P1 - Critical (Production Down)
- Complete service outage
- Data loss or corruption
- Security breach
- No workaround available

#### P2 - High (Production Impaired)
- Significant functionality unavailable
- Severe performance degradation
- Workaround available but not sustainable

#### P3 - Medium (Production Stable)
- Non-critical feature unavailable
- Minor performance issues
- Reasonable workaround available

#### P4 - Low (General Questions)
- Feature requests
- Documentation questions
- Training requests
- Best practice guidance

### 6.3 Escalation Process

```
Escalation Path:
┌─────────────────┐
│ Support Engineer│ (Initial Response)
└────────┬────────┘
         ▼
┌─────────────────┐
│ Senior Engineer │ (If unresolved in 2 hours)
└────────┬────────┘
         ▼
┌─────────────────┐
│ Team Lead       │ (If unresolved in 4 hours)
└────────┬────────┘
         ▼
┌─────────────────┐
│ Engineering Mgr │ (If unresolved in 8 hours)
└────────┬────────┘
         ▼
┌─────────────────┐
│ VP Engineering  │ (P1 issues > 24 hours)
└─────────────────┘
```

### 6.4 Support Responsibilities

#### Our Responsibilities
- Acknowledge requests within SLA
- Provide regular updates
- Escalate as needed
- Document resolutions
- Provide root cause analysis for P1/P2

#### Your Responsibilities
- Provide accurate contact information
- Clearly describe issues with reproduction steps
- Provide requested diagnostic information
- Apply recommended fixes
- Maintain supported configurations

## 7. MAINTENANCE

### 7.1 Scheduled Maintenance

#### Maintenance Windows
| Region | Day | Time (Local) | Duration |
|--------|-----|-------------|----------|
| US East | Sunday | 2 AM - 6 AM EST | Max 4 hours |
| US West | Sunday | 2 AM - 6 AM PST | Max 4 hours |
| EU West | Sunday | 2 AM - 6 AM GMT | Max 4 hours |
| APAC | Saturday | 2 AM - 6 AM SGT | Max 4 hours |

#### Notification Requirements
- Standard maintenance: 7 days advance notice
- Major upgrades: 30 days advance notice
- Security patches: 48 hours advance notice
- Customer approval required for changes during business hours

### 7.2 Emergency Maintenance

May be performed without advance notice for:
- Critical security vulnerabilities
- Data corruption prevention
- Service restoration
- Legal compliance

We will:
- Minimize duration
- Notify as soon as practical
- Provide post-maintenance report
- Apply service credits if applicable

### 7.3 Customer-Initiated Maintenance

You may request maintenance for:
- Version upgrades
- Configuration changes
- Performance tuning
- Capacity adjustments

Requirements:
- 48-hour advance notice
- Approval from authorized contact
- Performed during agreed window
- Rollback plan required

## 8. DATA MANAGEMENT

### 8.1 Backup Commitments

| Data Type | Frequency | Retention | Recovery Time | Recovery Point |
|-----------|-----------|-----------|---------------|----------------|
| Databases | Continuous | 30 days | 1 hour | 5 minutes |
| Configurations | Hourly | 90 days | 30 minutes | 1 hour |
| Audit Logs | Real-time | 7 years | 4 hours | 0 minutes |
| User Files | Daily | 30 days | 2 hours | 24 hours |

### 8.2 Disaster Recovery

#### RTO and RPO Targets
- **Recovery Time Objective (RTO)**: 4 hours
- **Recovery Point Objective (RPO)**: 1 hour

#### DR Testing
- Full DR test: Annually
- Partial DR test: Quarterly
- Backup recovery test: Monthly
- Results shared with customers

### 8.3 Data Portability

We provide:
- Export functionality in standard formats
- API access for bulk data extraction
- Support for data migration
- 30-day data retention after termination

## 9. SECURITY COMMITMENTS

### 9.1 Security Measures

We maintain:
- Encryption at rest (AES-256)
- Encryption in transit (TLS 1.3)
- Multi-factor authentication
- Role-based access control
- Security monitoring 24/7
- Annual penetration testing
- SOC 2 Type II certification

### 9.2 Incident Response

For security incidents:
- Detection and containment within 1 hour
- Customer notification within 24 hours
- Detailed report within 72 hours
- Remediation plan within 5 days
- Post-incident review within 30 days

### 9.3 Compliance

We maintain compliance with:
- SOC 2 Type II
- GDPR
- CCPA
- HIPAA (with BAA)
- PCI-DSS (for payment processing)

## 10. COMMUNICATION

### 10.1 Planned Communications

| Communication Type | Frequency | Method | Content |
|-------------------|-----------|--------|---------|
| SLA Report | Monthly | Email & Portal | Performance metrics |
| Maintenance Notice | As needed | Email & Status Page | Schedule and impact |
| Feature Updates | Quarterly | Email & Blog | New capabilities |
| Security Bulletins | As needed | Email & Portal | Security updates |

### 10.2 Incident Communications

During incidents:
- Initial notification within 15 minutes (P1)
- Updates every hour (P1) or per SLA
- Resolution notification immediately
- Post-incident report within 5 days

### 10.3 Contact Information

**24/7 Emergency**: [EMERGENCY_PHONE]
**Support Portal**: https://support.sqlstudio.com
**Email**: enterprise-support@sqlstudio.com
**Status Page**: https://status.sqlstudio.com
**Account Manager**: [ACCOUNT_MANAGER_CONTACT]

## 11. GOVERNANCE

### 11.1 Service Reviews

#### Quarterly Business Reviews
- SLA performance review
- Capacity planning
- Roadmap discussions
- Optimization opportunities
- Feedback and requirements

#### Annual Reviews
- Contract review
- SLA adjustments
- Strategic planning
- Technology updates

### 11.2 Change Management

For service changes:
- 30-day notice for material changes
- Customer advisory board input
- Backward compatibility maintained
- Migration support provided
- Rollback procedures available

### 11.3 Continuous Improvement

We commit to:
- Regular service enhancements
- Performance optimization
- Feature development based on feedback
- Security improvements
- Process refinements

## 12. REMEDIES AND LIMITATIONS

### 12.1 Your Remedies

For SLA violations:
- Service credits as specified
- Right to terminate for chronic violations (3 months consecutive)
- Escalation to executive management
- Priority support for resolution

### 12.2 Limitations

- Service credits are sole remedy for SLA violations
- Total credits limited to 100% of monthly fees
- No credits for issues within your control
- Must request credits within 30 days
- Credits cannot be exchanged for cash

### 12.3 Mutual Obligations

Both parties agree to:
- Work collaboratively on issues
- Provide timely information
- Escalate appropriately
- Document issues thoroughly
- Maintain confidentiality

## 13. DEFINITIONS AND CALCULATIONS

### 13.1 Availability Calculation

```
Monthly Uptime % = (Total Minutes - Downtime Minutes) / Total Minutes × 100

Where:
- Total Minutes = Days in month × 24 × 60
- Downtime Minutes = Sum of all qualifying downtime
```

### 13.2 Response Time Calculation

```
Response Time = Time of first byte - Time of request receipt

Measured at our network edge, excluding:
- Customer network latency
- Customer database query time
- Third-party service time
```

### 13.3 Error Rate Calculation

```
Error Rate = (5xx Errors / Total Requests) × 100

Excluding:
- Client errors (4xx)
- Rate limit errors (429)
- Customer-caused errors
```

## 14. EXCEPTIONS

### 14.1 Beta Features

Beta features are:
- Provided as-is
- Not covered by SLA
- Subject to change
- May be discontinued
- Clearly marked as beta

### 14.2 Third-Party Services

We're not responsible for:
- Third-party service outages
- Integration failures beyond our control
- Third-party API changes
- Customer's third-party services

### 14.3 Customer Responsibilities

SLA assumes you:
- Maintain compatible systems
- Follow documented procedures
- Apply recommended updates
- Report issues promptly
- Provide necessary access for troubleshooting

## 15. AMENDMENTS

### 15.1 SLA Updates

We may update this SLA with:
- 90-day notice for material adverse changes
- 30-day notice for improvements
- Immediate effect for clarifications
- Your consent for reductions in service levels

### 15.2 Version Control

| Version | Date | Changes | Approved By |
|---------|------|---------|-------------|
| 1.0.0 | [DATE] | Initial SLA | [APPROVER] |

## APPENDICES

### Appendix A: Monitoring Endpoints

Health check endpoints for customer monitoring:
- https://api.sqlstudio.com/health
- https://app.sqlstudio.com/health
- https://sync.sqlstudio.com/health

### Appendix B: Supported Configurations

Supported browsers, databases, and configurations:
https://docs.sqlstudio.com/supported-configurations

### Appendix C: Service Credit Request Form

Template available at:
https://support.sqlstudio.com/sla-credit-request

### Appendix D: Glossary

- **MTBF**: Mean Time Between Failures
- **MTTR**: Mean Time To Recovery
- **P95/P99**: 95th/99th percentile
- **RCA**: Root Cause Analysis
- **TAM**: Technical Account Manager

## AGREEMENT

This SLA is incorporated into and governed by the Master Services Agreement between the parties.

**CUSTOMER:**
Name: [CUSTOMER_NAME]
Signature: _______________________
Date: [DATE]

**SQL STUDIO INC:**
Name: [SIGNATORY_NAME]
Title: VP of Customer Success
Signature: _______________________
Date: [DATE]

---

**For questions about this SLA, contact:**
Enterprise Support Team
enterprise-support@sqlstudio.com
[SUPPORT_PHONE]