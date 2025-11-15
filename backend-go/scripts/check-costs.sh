#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PROJECT_ID=${1:-$GCP_PROJECT_ID}

if [ -z "$PROJECT_ID" ]; then
    echo -e "${RED}❌ Error: GCP_PROJECT_ID is required${NC}"
    echo ""
    echo "Usage: $0 <project-id>"
    echo "   or: export GCP_PROJECT_ID=your-project-id && $0"
    exit 1
fi

echo -e "${BLUE}💰 Cloud Run Cost Report for Howlerops Backend${NC}"
echo "=================================================="
echo "Project: $PROJECT_ID"
echo "Date: $(date)"
echo ""

# Set project
gcloud config set project $PROJECT_ID --quiet 2>/dev/null

# Check if billing is enabled
echo -e "${BLUE}Checking billing status...${NC}"
BILLING_ENABLED=$(gcloud beta billing projects describe $PROJECT_ID --format="value(billingEnabled)" 2>/dev/null || echo "unknown")

if [ "$BILLING_ENABLED" = "True" ]; then
    echo -e "${GREEN}✅ Billing enabled${NC}"
elif [ "$BILLING_ENABLED" = "False" ]; then
    echo -e "${YELLOW}⚠️  Billing not enabled - service may stop working${NC}"
else
    echo -e "${YELLOW}⚠️  Cannot determine billing status (might need permissions)${NC}"
fi
echo ""

# Get Cloud Run services
echo -e "${BLUE}Active Cloud Run Services:${NC}"
SERVICES=$(gcloud run services list --project=$PROJECT_ID --format="table(name,region,status.url)" 2>/dev/null)

if [ -n "$SERVICES" ]; then
    echo "$SERVICES"
else
    echo "No Cloud Run services found"
fi
echo ""

# Get revision count
echo -e "${BLUE}Active Revisions:${NC}"
REVISIONS=$(gcloud run revisions list --project=$PROJECT_ID --format="table(metadata.name,status.conditions[0].status,metadata.creationTimestamp)" --limit=10 2>/dev/null)

if [ -n "$REVISIONS" ]; then
    echo "$REVISIONS"
    REVISION_COUNT=$(echo "$REVISIONS" | tail -n +2 | wc -l | tr -d ' ')
    echo ""
    if [ "$REVISION_COUNT" -gt 5 ]; then
        echo -e "${YELLOW}⚠️  Warning: $REVISION_COUNT revisions detected${NC}"
        echo "  Consider cleaning up old revisions to reduce costs"
        echo "  Command: gcloud run revisions list --service=SERVICE_NAME"
    else
        echo -e "${GREEN}✅ Revision count looks good ($REVISION_COUNT)${NC}"
    fi
else
    echo "No revisions found"
fi
echo ""

# Get request metrics (last 24 hours)
echo -e "${BLUE}Request Metrics (last 24 hours):${NC}"

# Try to get request count
REQUEST_COUNT=$(gcloud monitoring time-series list \
    --filter='metric.type="run.googleapis.com/request_count"' \
    --project=$PROJECT_ID \
    --format="value(point.value.int64Value)" \
    --start-time="$(date -u -v-24H +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d '24 hours ago' +%Y-%m-%dT%H:%M:%SZ)" \
    2>/dev/null | awk '{s+=$1} END {print s}' || echo "0")

if [ "$REQUEST_COUNT" = "0" ] || [ -z "$REQUEST_COUNT" ]; then
    echo "  Requests (24h): No data available"
    echo -e "${YELLOW}  Note: Metrics may take a few hours to appear after deployment${NC}"
else
    echo "  Requests (24h): $REQUEST_COUNT"

    # Estimate monthly requests
    MONTHLY_ESTIMATE=$((REQUEST_COUNT * 30))
    echo "  Estimated monthly: $MONTHLY_ESTIMATE requests"

    # Free tier is 2M requests/month
    FREE_TIER_REQUESTS=2000000
    if [ $MONTHLY_ESTIMATE -lt $FREE_TIER_REQUESTS ]; then
        PERCENTAGE=$((MONTHLY_ESTIMATE * 100 / FREE_TIER_REQUESTS))
        echo -e "  ${GREEN}✅ Within free tier ($PERCENTAGE% of 2M/month limit)${NC}"
    else
        OVERAGE=$((MONTHLY_ESTIMATE - FREE_TIER_REQUESTS))
        echo -e "  ${RED}❌ Exceeds free tier by $OVERAGE requests${NC}"
        echo "  Estimated overage cost: \$$(echo "scale=2; $OVERAGE * 0.0000004" | bc)/month"
    fi
fi
echo ""

# Get instance count
echo -e "${BLUE}Instance Metrics:${NC}"

INSTANCE_DATA=$(gcloud monitoring time-series list \
    --filter='metric.type="run.googleapis.com/container/instance_count"' \
    --project=$PROJECT_ID \
    --format="table(metric.labels.service_name,point.value.int64Value)" \
    --start-time="$(date -u -v-1H +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%SZ)" \
    --limit=5 \
    2>/dev/null)

if [ -n "$INSTANCE_DATA" ]; then
    echo "$INSTANCE_DATA"
else
    echo "  No instance data available yet"
fi
echo ""

# Get container memory usage
echo -e "${BLUE}Memory Usage:${NC}"

MEMORY_DATA=$(gcloud monitoring time-series list \
    --filter='metric.type="run.googleapis.com/container/memory/utilizations"' \
    --project=$PROJECT_ID \
    --format="table(point.value.doubleValue)" \
    --start-time="$(date -u -v-1H +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%SZ)" \
    --limit=1 \
    2>/dev/null | tail -n 1)

if [ -n "$MEMORY_DATA" ] && [ "$MEMORY_DATA" != "DOUBLE_VALUE" ]; then
    MEMORY_PCT=$(echo "$MEMORY_DATA * 100" | bc -l | cut -d. -f1)
    echo "  Current usage: ${MEMORY_PCT}%"
    if [ "$MEMORY_PCT" -gt 80 ]; then
        echo -e "  ${YELLOW}⚠️  Warning: High memory usage${NC}"
        echo "  Consider increasing memory allocation"
    else
        echo -e "  ${GREEN}✅ Memory usage looks good${NC}"
    fi
else
    echo "  No memory data available yet"
fi
echo ""

# Recent errors
echo -e "${BLUE}Recent Errors (last hour):${NC}"

ERROR_COUNT=$(gcloud logging read "resource.type=cloud_run_revision AND severity>=ERROR" \
    --project=$PROJECT_ID \
    --limit=100 \
    --format="value(timestamp)" \
    --freshness=1h \
    2>/dev/null | wc -l | tr -d ' ')

if [ "$ERROR_COUNT" -gt 0 ]; then
    echo -e "  ${YELLOW}⚠️  $ERROR_COUNT errors in the last hour${NC}"
    echo "  View errors: gcloud logging read \"resource.type=cloud_run_revision AND severity>=ERROR\" --limit=10"
else
    echo -e "  ${GREEN}✅ No errors in the last hour${NC}"
fi
echo ""

# Cost optimization tips
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}💡 Cost Optimization Tips${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "Cloud Run Free Tier (per month):"
echo "  ✓ 2M requests"
echo "  ✓ 180,000 vCPU-seconds"
echo "  ✓ 360,000 GiB-seconds"
echo "  ✓ 1 GB network egress (North America)"
echo ""
echo "To stay within free tier:"
echo "  1. ✓ Min instances = 0 (scale-to-zero)"
echo "  2. ✓ Max instances = 10 (prevent runaway)"
echo "  3. ✓ Memory = 512Mi (don't overprovision)"
echo "  4. ✓ CPU allocation = request processing only"
echo "  5. ✓ Region = us-central1 (lowest cost)"
echo "  6. ✓ Clean up old revisions"
echo ""
echo "Monitoring commands:"
echo "  - View all costs: gcloud billing accounts list"
echo "  - View logs: gcloud run services logs tail sql-studio-backend"
echo "  - View metrics: gcloud monitoring dashboards list"
echo "  - Set budget: gcloud billing budgets create --amount=5USD"
echo ""

# Check service configuration
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Service Configuration${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

SERVICE_CONFIG=$(gcloud run services describe sql-studio-backend \
    --project=$PROJECT_ID \
    --region=us-central1 \
    --format="value(spec.template.spec.containers[0].resources.limits.memory,spec.template.metadata.annotations.autoscaling\.knative\.dev/minScale,spec.template.metadata.annotations.autoscaling\.knative\.dev/maxScale)" \
    2>/dev/null || echo "unknown unknown unknown")

MEMORY=$(echo $SERVICE_CONFIG | cut -d' ' -f1)
MIN_INSTANCES=$(echo $SERVICE_CONFIG | cut -d' ' -f2)
MAX_INSTANCES=$(echo $SERVICE_CONFIG | cut -d' ' -f3)

if [ "$MEMORY" != "unknown" ]; then
    echo "  Memory: $MEMORY"
    echo "  Min instances: ${MIN_INSTANCES:-0}"
    echo "  Max instances: ${MAX_INSTANCES:-100}"
    echo ""

    # Validate settings
    if [ "${MIN_INSTANCES:-0}" = "0" ]; then
        echo -e "  ${GREEN}✅ Scale-to-zero enabled (good for costs)${NC}"
    else
        echo -e "  ${YELLOW}⚠️  Min instances > 0 (will incur constant costs)${NC}"
    fi

    if [ "${MAX_INSTANCES:-100}" -le 10 ]; then
        echo -e "  ${GREEN}✅ Max instances limited (good for cost control)${NC}"
    else
        echo -e "  ${YELLOW}⚠️  Max instances > 10 (could incur high costs)${NC}"
    fi
else
    echo "  Service not found or not accessible"
fi
echo ""

# Links
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}🔗 Useful Links${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "  - Cloud Run Console: https://console.cloud.google.com/run?project=$PROJECT_ID"
echo "  - Billing: https://console.cloud.google.com/billing?project=$PROJECT_ID"
echo "  - Logs: https://console.cloud.google.com/logs?project=$PROJECT_ID"
echo "  - Monitoring: https://console.cloud.google.com/monitoring?project=$PROJECT_ID"
echo "  - Metrics Explorer: https://console.cloud.google.com/monitoring/metrics-explorer?project=$PROJECT_ID"
echo ""

# Summary
echo -e "${GREEN}✅ Cost report complete${NC}"
echo ""
echo "Run this script regularly to monitor costs:"
echo "  ./scripts/check-costs.sh"
echo "  make check-costs"
echo ""
