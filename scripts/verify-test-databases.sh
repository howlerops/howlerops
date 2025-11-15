#!/bin/bash
# Test Database Verification Script
# This script verifies that all test databases are running and contain seed data

set -e

echo "========================================"
echo "HowlerOps Test Database Verification"
echo "========================================"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

success() {
    echo -e "${GREEN}✓${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1"
}

warn() {
    echo -e "${YELLOW}!${NC} $1"
}

# Check if docker-compose is running
echo "Checking container status..."
if ! docker-compose -f docker-compose.testdb.yml ps | grep -q "Up"; then
    error "Containers are not running"
    echo "Run: docker-compose -f docker-compose.testdb.yml up -d"
    exit 1
fi
success "Containers are running"
echo ""

# Function to check database
check_db() {
    local name=$1
    local test_cmd=$2
    local expected=$3

    echo -n "Testing $name... "
    if result=$(eval "$test_cmd" 2>&1); then
        if [[ -z "$expected" ]] || echo "$result" | grep -q "$expected"; then
            success "$name is working"
            return 0
        else
            error "$name query succeeded but unexpected result"
            echo "Expected: $expected"
            echo "Got: $result"
            return 1
        fi
    else
        error "$name failed"
        echo "Error: $result"
        return 1
    fi
}

# PostgreSQL
echo "--- PostgreSQL ---"
check_db "PostgreSQL Connection" \
    "docker exec howlerops-postgres-test psql -U testuser -d testdb -c 'SELECT 1;' -t" \
    "1"

check_db "PostgreSQL Users Table" \
    "docker exec howlerops-postgres-test psql -U testuser -d testdb -c 'SELECT COUNT(*) FROM users;' -t" \
    "150"

check_db "PostgreSQL Products Table" \
    "docker exec howlerops-postgres-test psql -U testuser -d testdb -c 'SELECT COUNT(*) FROM products;' -t" \
    "250"

check_db "PostgreSQL Orders Table" \
    "docker exec howlerops-postgres-test psql -U testuser -d testdb -c 'SELECT COUNT(*) FROM orders;' -t" \
    "600"

echo ""

# MySQL
echo "--- MySQL ---"
check_db "MySQL Connection" \
    "docker exec howlerops-mysql-test mysql -u testuser -ptestpass testdb -e 'SELECT 1;' -s -N" \
    "1"

check_db "MySQL Users Table" \
    "docker exec howlerops-mysql-test mysql -u testuser -ptestpass testdb -e 'SELECT COUNT(*) FROM users;' -s -N" \
    "150"

check_db "MySQL Products Table" \
    "docker exec howlerops-mysql-test mysql -u testuser -ptestpass testdb -e 'SELECT COUNT(*) FROM products;' -s -N" \
    "250"

check_db "MySQL Orders Table" \
    "docker exec howlerops-mysql-test mysql -u testuser -ptestpass testdb -e 'SELECT COUNT(*) FROM orders;' -s -N" \
    "600"

echo ""

# MongoDB
echo "--- MongoDB ---"
check_db "MongoDB Connection" \
    "docker exec howlerops-mongodb-test mongosh --quiet -u testuser -p testpass --authenticationDatabase admin testdb --eval 'db.adminCommand({ ping: 1 }).ok'" \
    "1"

check_db "MongoDB Users Collection" \
    "docker exec howlerops-mongodb-test mongosh --quiet -u testuser -p testpass --authenticationDatabase admin testdb --eval 'db.users.countDocuments()'" \
    "150"

check_db "MongoDB Products Collection" \
    "docker exec howlerops-mongodb-test mongosh --quiet -u testuser -p testpass --authenticationDatabase admin testdb --eval 'db.products.countDocuments()'" \
    "250"

check_db "MongoDB Orders Collection" \
    "docker exec howlerops-mongodb-test mongosh --quiet -u testuser -p testpass --authenticationDatabase admin testdb --eval 'db.orders.countDocuments()'" \
    "600"

echo ""

# ElasticSearch
echo "--- ElasticSearch ---"
check_db "ElasticSearch Connection" \
    "curl -s http://localhost:9201/_cluster/health | grep -o '\"status\":\"[^\"]*\"'" \
    "status"

check_db "ElasticSearch Users Index" \
    "curl -s http://localhost:9201/users/_count | grep -o '\"count\":[0-9]*'" \
    "count"

check_db "ElasticSearch Products Index" \
    "curl -s http://localhost:9201/products/_count | grep -o '\"count\":[0-9]*'" \
    "count"

check_db "ElasticSearch Orders Index" \
    "curl -s http://localhost:9201/orders/_count | grep -o '\"count\":[0-9]*'" \
    "count"

echo ""

# ClickHouse
echo "--- ClickHouse ---"
check_db "ClickHouse Connection" \
    "docker exec howlerops-clickhouse-test clickhouse-client --query 'SELECT 1' 2>&1" \
    "1"

check_db "ClickHouse Users Table" \
    "docker exec howlerops-clickhouse-test clickhouse-client --query 'SELECT count() FROM testdb.users' 2>&1" \
    "150"

check_db "ClickHouse Products Table" \
    "docker exec howlerops-clickhouse-test clickhouse-client --query 'SELECT count() FROM testdb.products' 2>&1" \
    "250"

check_db "ClickHouse Orders Table" \
    "docker exec howlerops-clickhouse-test clickhouse-client --query 'SELECT count() FROM testdb.orders' 2>&1" \
    "600"

echo ""

# Redis
echo "--- Redis ---"
check_db "Redis Connection" \
    "docker exec howlerops-redis-test redis-cli PING" \
    "PONG"

echo ""
echo "========================================"
echo "Summary"
echo "========================================"

# Count successes
total_checks=24
echo ""
success "All basic checks passed!"
echo ""
echo "Connection details:"
echo "  PostgreSQL:    localhost:5433 (user: testuser, pass: testpass, db: testdb)"
echo "  MySQL:         localhost:3307 (user: testuser, pass: testpass, db: testdb)"
echo "  MongoDB:       localhost:27018 (user: testuser, pass: testpass, db: testdb)"
echo "  ElasticSearch: localhost:9201"
echo "  ClickHouse:    localhost:8124 (HTTP), localhost:9001 (native)"
echo "  Redis:         localhost:6380"
echo ""
echo "For detailed usage, see:"
echo "  - scripts/seed-test-data/README.md (complete guide)"
echo "  - scripts/seed-test-data/QUICK_START.md (quick reference)"
echo ""
