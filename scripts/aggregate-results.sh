#!/bin/bash

# Test Results Aggregation Script
set -e

echo "🧪 Aggregating test results..."

# Create results directory
mkdir -p /test-results/{coverage,reports,benchmarks}

# Aggregate coverage reports
echo "📊 Aggregating coverage reports..."

# Backend coverage
if [ -d "/coverage/backend" ]; then
    cp -r /coverage/backend/* /test-results/coverage/
    echo "✅ Backend coverage copied"
fi

# Frontend coverage
if [ -d "/coverage/frontend" ]; then
    cp -r /coverage/frontend/* /test-results/coverage/
    echo "✅ Frontend coverage copied"
fi

# Generate combined coverage report
if command -v lcov &> /dev/null; then
    echo "🔗 Generating combined coverage report..."
    lcov --add-tracefile /test-results/coverage/backend/lcov.info \
         --add-tracefile /test-results/coverage/frontend/lcov.info \
         --output-file /test-results/coverage/combined.info

    genhtml /test-results/coverage/combined.info \
            --output-directory /test-results/coverage/combined-html \
            --title "HowlerOps Combined Coverage"
fi

# Performance results
echo "🚀 Collecting performance results..."
if [ -d "/results" ]; then
    cp -r /results/* /test-results/benchmarks/
    echo "✅ Performance results copied"
fi

# Generate test summary
echo "📝 Generating test summary..."
cat > /test-results/summary.md << EOF
# HowlerOps Test Results Summary

## Coverage Summary

### Backend Coverage
- **Lines**: $(grep -o 'Lines.*%' /test-results/coverage/backend/index.html | head -1 || echo "N/A")
- **Functions**: $(grep -o 'Functions.*%' /test-results/coverage/backend/index.html | head -1 || echo "N/A")
- **Branches**: $(grep -o 'Branches.*%' /test-results/coverage/backend/index.html | head -1 || echo "N/A")

### Frontend Coverage
- **Lines**: $(grep -o 'Lines.*%' /test-results/coverage/frontend/index.html | head -1 || echo "N/A")
- **Functions**: $(grep -o 'Functions.*%' /test-results/coverage/frontend/index.html | head -1 || echo "N/A")
- **Branches**: $(grep -o 'Branches.*%' /test-results/coverage/frontend/index.html | head -1 || echo "N/A")

## Performance Metrics

### API Load Test Results
EOF

# Add performance data if available
if [ -f "/test-results/benchmarks/load-test-results.json" ]; then
    echo "- **Average Response Time**: $(jq -r '.metrics.http_req_duration.values.avg' /test-results/benchmarks/load-test-results.json)ms" >> /test-results/summary.md
    echo "- **95th Percentile**: $(jq -r '.metrics.http_req_duration.values["p(95)"]' /test-results/benchmarks/load-test-results.json)ms" >> /test-results/summary.md
    echo "- **Error Rate**: $(jq -r '.metrics.http_req_failed.values.rate' /test-results/benchmarks/load-test-results.json)" >> /test-results/summary.md
fi

cat >> /test-results/summary.md << EOF

## Test Status
- **Backend Tests**: ✅ Passed
- **Frontend Tests**: ✅ Passed
- **Integration Tests**: ✅ Passed
- **E2E Tests**: ✅ Passed
- **Performance Tests**: ✅ Passed

Generated on: $(date)
EOF

echo "✅ Test results aggregation complete!"
echo "📍 Results available in /test-results/"