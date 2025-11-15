# Quick Start Guide - Test Databases

## TL;DR

```bash
# Start all test databases with seed data
docker-compose -f docker-compose.testdb.yml up -d

# Wait 30-60 seconds for seeding to complete, then connect:

# PostgreSQL
psql -h localhost -p 5433 -U testuser -d testdb

# MySQL
mysql -h 127.0.0.1 -P 3307 -u testuser -ptestpass testdb

# MongoDB
mongosh "mongodb://testuser:testpass@localhost:27018/testdb?authSource=admin"

# ElasticSearch
curl http://localhost:9201/_cat/indices?v

# ClickHouse
docker exec -it howlerops-clickhouse-test clickhouse-client

# Redis
redis-cli -p 6380

# Stop everything
docker-compose -f docker-compose.testdb.yml down

# Reset all data
docker-compose -f docker-compose.testdb.yml down -v
```

## What You Get

Each database contains:
- **150 users** (active, inactive, suspended)
- **250 products** (8 categories, various prices)
- **600 orders** (multiple statuses, realistic dates)
- **~2100 order items** (2-5 items per order)
- **1200 audit logs** (track changes)
- **50 active sessions**
- **2000 analytics events** (time-series data)

## Connection Ports

| Database | Port | Standard Port |
|----------|------|---------------|
| PostgreSQL | 5433 | 5432 |
| MySQL | 3307 | 3306 |
| MongoDB | 27018 | 27017 |
| ElasticSearch | 9201 | 9200 |
| ClickHouse HTTP | 8124 | 8123 |
| ClickHouse Native | 9001 | 9000 |
| Redis | 6380 | 6379 |

All ports offset to avoid conflicts with dev databases.

## First Query to Try

### PostgreSQL / MySQL
```sql
SELECT status, COUNT(*) FROM users GROUP BY status;
```

### MongoDB
```javascript
db.users.aggregate([
  { $group: { _id: "$status", count: { $sum: 1 } } }
])
```

### ElasticSearch
```bash
curl http://localhost:9201/users/_search?size=0 -H 'Content-Type: application/json' -d'
{"aggs": {"by_status": {"terms": {"field": "status"}}}}'
```

### ClickHouse
```sql
SELECT status, count() FROM testdb.users GROUP BY status;
```

## Credentials

All databases use:
- **Username:** testuser
- **Password:** testpass
- **Database:** testdb

(MongoDB requires `authSource=admin`)

## See Full Documentation

For detailed examples, queries, and troubleshooting:
- [README.md](./README.md) - Complete documentation
- [docker-compose.testdb.yml](../../docker-compose.testdb.yml) - Full config

## Troubleshooting One-Liners

```bash
# View logs for a specific database
docker-compose -f docker-compose.testdb.yml logs postgres-test

# Restart a single database
docker-compose -f docker-compose.testdb.yml restart mysql-test

# Check if seeding completed
docker-compose -f docker-compose.testdb.yml logs | grep -i "seeding complete"

# Check health status
docker-compose -f docker-compose.testdb.yml ps
```
