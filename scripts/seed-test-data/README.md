# HowlerOps Test Database Environment

This directory contains seed data scripts for all database types supported by HowlerOps. The Docker Compose configuration provides a complete test database environment with realistic data.

## Quick Start

### Start All Databases

```bash
# From the project root
docker-compose -f docker-compose.testdb.yml up -d

# View logs
docker-compose -f docker-compose.testdb.yml logs -f

# Check status
docker-compose -f docker-compose.testdb.yml ps
```

### Stop All Databases

```bash
docker-compose -f docker-compose.testdb.yml down
```

### Reset All Data (Clean Slate)

```bash
# Stop containers and remove volumes
docker-compose -f docker-compose.testdb.yml down -v

# Start fresh
docker-compose -f docker-compose.testdb.yml up -d
```

## Database Configurations

### Connection Details

| Database | Port | Username | Password | Database |
|----------|------|----------|----------|----------|
| PostgreSQL | 5433 | testuser | testpass | testdb |
| MySQL | 3307 | testuser | testpass | testdb |
| MongoDB | 27018 | testuser | testpass | testdb |
| ElasticSearch | 9201 | N/A | N/A | N/A |
| ClickHouse | 9001 (native), 8124 (HTTP) | testuser | testpass | testdb |
| Redis | 6380 | N/A | N/A | N/A |

**Note:** All ports are offset from standard ports to avoid conflicts with local development databases.

## Seeded Data

Each database contains the following datasets:

### Users (150 records)
- **Fields:** id, uuid, username, email, full_name, password_hash, status, role, created_at, updated_at, last_login, metadata
- **Statuses:** active (majority), inactive (~7%), suspended (~5%)
- **Roles:** user (majority), manager (~10%), admin (~4%)
- **Use Cases:** Testing authentication, user management, permission systems

### Products (250 records)
- **Fields:** id, uuid, sku, name, description, category, price, cost, stock_quantity, status, created_at, updated_at, metadata
- **Categories:** Electronics, Clothing, Home & Garden, Sports, Books, Toys, Food, Other
- **Statuses:** active (majority), inactive (~3%), discontinued (~2.5%)
- **Price Range:** $10 - $1000
- **Use Cases:** Testing catalog queries, search, inventory management

### Orders (600 records)
- **Fields:** id, uuid, order_number, user_id, status, total_amount, tax_amount, shipping_amount, discount_amount, payment_method, addresses, timestamps, metadata
- **Statuses:** pending, processing, shipped, delivered, cancelled, refunded
- **Payment Methods:** credit_card, paypal, bank_transfer, cash
- **Total Range:** $20 - $1000
- **Use Cases:** Testing order processing, reporting, analytics

### Order Items (~2100 records)
- **Fields:** id, uuid, order_id, product_id, quantity, unit_price, discount_percent, total_price, created_at, metadata
- **Each order has 2-5 items**
- **Use Cases:** Testing joins, aggregations, order details

### Audit Logs (1200 records)
- **Fields:** id, uuid, user_id, action, entity_type, entity_id, old_values, new_values, ip_address, user_agent, created_at, metadata
- **Actions:** user.created, user.updated, order.created, order.updated, product.created, product.updated
- **Use Cases:** Testing audit trails, compliance, debugging

### Sessions (50 records)
- **Fields:** id, session_id, user_id, data, created_at, updated_at, expires_at
- **Use Cases:** Testing session management, active user tracking

### Analytics Events (2000 records)
- **Fields:** id, event_type, user_id, event_data, created_at
- **Event Types:** page_view, product_view, add_to_cart, remove_from_cart, checkout_started, checkout_completed, search, login, logout, error
- **Use Cases:** Testing time-series queries, analytics, event tracking

## Example Queries

### PostgreSQL

```bash
# Connect to PostgreSQL
docker exec -it howlerops-postgres-test psql -U testuser -d testdb

# Or from host
psql -h localhost -p 5433 -U testuser -d testdb
```

**Example Queries:**

```sql
-- Count users by status
SELECT status, COUNT(*) as count
FROM users
GROUP BY status
ORDER BY count DESC;

-- Top 10 selling products
SELECT
    p.name,
    COUNT(DISTINCT oi.order_id) as order_count,
    SUM(oi.quantity) as total_sold,
    SUM(oi.total_price) as revenue
FROM products p
JOIN order_items oi ON p.id = oi.product_id
GROUP BY p.id, p.name
ORDER BY revenue DESC
LIMIT 10;

-- Orders by status with user details
SELECT
    o.order_number,
    u.username,
    u.email,
    o.status,
    o.total_amount,
    o.created_at
FROM orders o
JOIN users u ON o.user_id = u.id
WHERE o.status = 'pending'
ORDER BY o.created_at DESC
LIMIT 10;

-- Daily order volume and revenue
SELECT
    DATE(created_at) as order_date,
    COUNT(*) as order_count,
    SUM(total_amount) as total_revenue,
    AVG(total_amount) as avg_order_value
FROM orders
WHERE status != 'cancelled'
GROUP BY DATE(created_at)
ORDER BY order_date DESC;

-- Products running low on stock
SELECT sku, name, category, stock_quantity, price
FROM products
WHERE status = 'active' AND stock_quantity < 100
ORDER BY stock_quantity ASC;

-- User activity analysis
SELECT
    DATE(created_at) as event_date,
    event_type,
    COUNT(*) as event_count
FROM analytics_events
WHERE created_at >= NOW() - INTERVAL '7 days'
GROUP BY DATE(created_at), event_type
ORDER BY event_date DESC, event_count DESC;

-- Full-text search on products
SELECT name, description, category, price
FROM products
WHERE name ILIKE '%premium%'
  OR description ILIKE '%premium%'
ORDER BY price DESC;

-- JSON metadata queries
SELECT
    username,
    email,
    metadata->>'account_tier' as tier,
    metadata->'preferences'->>'newsletter' as newsletter_opt_in
FROM users
WHERE metadata->>'account_tier' = 'premium';

-- Order summary view
SELECT * FROM order_summaries
WHERE status = 'delivered'
ORDER BY created_at DESC
LIMIT 20;

-- Materialized view query
SELECT * FROM product_analytics
ORDER BY revenue DESC
LIMIT 10;
```

### MySQL

```bash
# Connect to MySQL
docker exec -it howlerops-mysql-test mysql -u testuser -ptestpass testdb

# Or from host
mysql -h 127.0.0.1 -P 3307 -u testuser -ptestpass testdb
```

**Example Queries:**

```sql
-- Similar queries as PostgreSQL, with some syntax differences:

-- JSON queries in MySQL
SELECT
    username,
    email,
    JSON_EXTRACT(metadata, '$.account_tier') as tier,
    JSON_EXTRACT(metadata, '$.preferences.newsletter') as newsletter
FROM users
WHERE JSON_EXTRACT(metadata, '$.account_tier') = 'premium';

-- Full-text search
SELECT name, description, category, price
FROM products
WHERE MATCH(name) AGAINST('premium' IN NATURAL LANGUAGE MODE)
ORDER BY price DESC;

-- Date aggregations
SELECT
    DATE(created_at) as order_date,
    COUNT(*) as order_count,
    SUM(total_amount) as total_revenue
FROM orders
WHERE created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
GROUP BY DATE(created_at)
ORDER BY order_date DESC;
```

### MongoDB

```bash
# Connect to MongoDB
docker exec -it howlerops-mongodb-test mongosh -u testuser -p testpass --authenticationDatabase admin testdb

# Or from host
mongosh "mongodb://testuser:testpass@localhost:27018/testdb?authSource=admin"
```

**Example Queries:**

```javascript
// Count users by status
db.users.aggregate([
  { $group: { _id: "$status", count: { $sum: 1 } } },
  { $sort: { count: -1 } }
])

// Top selling products
db.orders.aggregate([
  { $unwind: "$items" },
  {
    $group: {
      _id: "$items.productId",
      productName: { $first: "$items.productName" },
      orderCount: { $sum: 1 },
      totalSold: { $sum: "$items.quantity" },
      revenue: { $sum: "$items.totalPrice" }
    }
  },
  { $sort: { revenue: -1 } },
  { $limit: 10 }
])

// Orders by status
db.orders.find(
  { status: "pending" },
  { orderNumber: 1, userEmail: 1, status: 1, totalAmount: 1, createdAt: 1 }
).sort({ createdAt: -1 }).limit(10)

// Daily order analytics
db.orders.aggregate([
  {
    $group: {
      _id: { $dateToString: { format: "%Y-%m-%d", date: "$createdAt" } },
      orderCount: { $sum: 1 },
      totalRevenue: { $sum: "$totalAmount" },
      avgOrderValue: { $avg: "$totalAmount" }
    }
  },
  { $sort: { _id: -1 } }
])

// Products with low stock
db.products.find(
  { status: "active", stockQuantity: { $lt: 100 } },
  { sku: 1, name: 1, category: 1, stockQuantity: 1, price: 1 }
).sort({ stockQuantity: 1 })

// Text search on products
db.products.find(
  { $text: { $search: "premium" } },
  { name: 1, description: 1, category: 1, price: 1 }
).sort({ price: -1 })

// User activity by event type
db.analyticsEvents.aggregate([
  {
    $match: {
      createdAt: { $gte: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000) }
    }
  },
  {
    $group: {
      _id: {
        date: { $dateToString: { format: "%Y-%m-%d", date: "$createdAt" } },
        eventType: "$eventType"
      },
      count: { $sum: 1 }
    }
  },
  { $sort: { "_id.date": -1, count: -1 } }
])

// Premium users
db.users.find(
  { "metadata.accountTier": "premium" },
  { username: 1, email: 1, "metadata.accountTier": 1 }
)

// Orders with items (embedded documents)
db.orders.find(
  { "items.quantity": { $gte: 3 } },
  { orderNumber: 1, status: 1, totalAmount: 1, items: 1 }
).limit(10)
```

### ElasticSearch

```bash
# From host
curl http://localhost:9201/_cat/indices?v

# Or use dev tools in Kibana (if you add it)
```

**Example Queries:**

```bash
# Count users by status
curl -X GET "http://localhost:9201/users/_search?pretty" -H 'Content-Type: application/json' -d'
{
  "size": 0,
  "aggs": {
    "by_status": {
      "terms": { "field": "status" }
    }
  }
}'

# Search products by name
curl -X GET "http://localhost:9201/products/_search?pretty" -H 'Content-Type: application/json' -d'
{
  "query": {
    "match": { "name": "premium" }
  },
  "sort": [ { "price": "desc" } ],
  "size": 10
}'

# Orders by date range
curl -X GET "http://localhost:9201/orders/_search?pretty" -H 'Content-Type: application/json' -d'
{
  "query": {
    "range": {
      "created_at": {
        "gte": "now-7d/d",
        "lte": "now/d"
      }
    }
  }
}'

# Aggregation - Daily order volume
curl -X GET "http://localhost:9201/orders/_search?pretty" -H 'Content-Type: application/json' -d'
{
  "size": 0,
  "aggs": {
    "daily_orders": {
      "date_histogram": {
        "field": "created_at",
        "calendar_interval": "day"
      },
      "aggs": {
        "total_revenue": { "sum": { "field": "total_amount" } }
      }
    }
  }
}'

# Multi-field search
curl -X GET "http://localhost:9201/products/_search?pretty" -H 'Content-Type: application/json' -d'
{
  "query": {
    "multi_match": {
      "query": "premium quality",
      "fields": ["name^2", "description"]
    }
  }
}'

# Analytics events by type
curl -X GET "http://localhost:9201/analytics_events/_search?pretty" -H 'Content-Type: application/json' -d'
{
  "size": 0,
  "aggs": {
    "event_types": {
      "terms": { "field": "event_type" }
    }
  }
}'
```

### ClickHouse

```bash
# Connect to ClickHouse (HTTP interface)
curl 'http://localhost:8124/?user=testuser&password=testpass&database=testdb' \
  -d 'SELECT count() FROM users'

# Or use clickhouse-client
docker exec -it howlerops-clickhouse-test clickhouse-client \
  --user=testuser --password=testpass --database=testdb
```

**Example Queries:**

```sql
-- Count users by status
SELECT status, count() as count
FROM testdb.users
GROUP BY status
ORDER BY count DESC;

-- Top selling products with revenue
SELECT
    p.name,
    count(DISTINCT oi.order_id) as order_count,
    sum(oi.quantity) as total_sold,
    sum(oi.total_price) as revenue
FROM testdb.products p
JOIN testdb.order_items oi ON p.id = oi.product_id
GROUP BY p.id, p.name
ORDER BY revenue DESC
LIMIT 10;

-- Daily order analytics (time-series optimized)
SELECT
    toDate(created_at) as order_date,
    count() as order_count,
    sum(total_amount) as total_revenue,
    avg(total_amount) as avg_order_value
FROM testdb.orders
WHERE status != 'cancelled'
GROUP BY order_date
ORDER BY order_date DESC;

-- Products with low stock
SELECT sku, name, category, stock_quantity, price
FROM testdb.products
WHERE status = 'active' AND stock_quantity < 100
ORDER BY stock_quantity ASC;

-- Event analytics with time-series partitioning
SELECT
    event_type,
    toStartOfHour(created_at) as hour,
    count() as event_count
FROM testdb.analytics_events
WHERE created_at >= now() - INTERVAL 24 HOUR
GROUP BY event_type, hour
ORDER BY hour DESC, event_count DESC;

-- JSON extraction from metadata
SELECT
    username,
    email,
    JSONExtractString(metadata, 'account_tier') as tier,
    JSONExtractString(metadata, 'signup_source') as source
FROM testdb.users
WHERE JSONExtractString(metadata, 'account_tier') = 'premium';

-- User activity summary
SELECT
    u.username,
    u.email,
    count(DISTINCT o.id) as order_count,
    sum(o.total_amount) as total_spent
FROM testdb.users u
LEFT JOIN testdb.orders o ON u.id = o.user_id
WHERE u.status = 'active'
GROUP BY u.id, u.username, u.email
ORDER BY total_spent DESC
LIMIT 20;

-- Query materialized view
SELECT * FROM testdb.product_analytics
ORDER BY revenue DESC
LIMIT 10;
```

### Redis

```bash
# Connect to Redis
docker exec -it howlerops-redis-test redis-cli

# Or from host
redis-cli -p 6380
```

**Example Commands:**

```bash
# Set and get values
SET test:key "test value"
GET test:key

# Hash operations (simulate session)
HSET session:user123 username "user1" email "user1@example.com" login_count 5
HGET session:user123 username
HGETALL session:user123

# List operations (simulate activity log)
LPUSH activity:user1 "logged_in" "viewed_product_42" "added_to_cart"
LRANGE activity:user1 0 -1

# Set operations (simulate tags)
SADD product:42:tags "premium" "electronics" "featured"
SMEMBERS product:42:tags

# Sorted set (leaderboard)
ZADD leaderboard 1000 "user1" 850 "user2" 1200 "user3"
ZREVRANGE leaderboard 0 9 WITHSCORES

# Key expiration (cache)
SET cache:product:42 '{"name":"Product 42","price":99.99}' EX 3600
TTL cache:product:42
```

## Performance Testing

### Test Database Performance

```bash
# PostgreSQL - Run EXPLAIN ANALYZE
docker exec -it howlerops-postgres-test psql -U testuser -d testdb -c \
  "EXPLAIN ANALYZE SELECT * FROM orders WHERE user_id = 42;"

# MySQL - Run EXPLAIN
docker exec -it howlerops-mysql-test mysql -u testuser -ptestpass testdb -e \
  "EXPLAIN SELECT * FROM orders WHERE user_id = 42;"

# MongoDB - Run explain
docker exec -it howlerops-mongodb-test mongosh -u testuser -p testpass \
  --authenticationDatabase admin testdb --eval \
  "db.orders.find({userId: ObjectId('...')}).explain('executionStats')"
```

### Load Testing

You can use the populated databases to test:
- Query performance with realistic data volumes
- Index effectiveness
- Join performance
- Aggregation speed
- Full-text search
- JSON query performance

## Troubleshooting

### Database Not Starting

```bash
# Check logs
docker-compose -f docker-compose.testdb.yml logs <service-name>

# Examples:
docker-compose -f docker-compose.testdb.yml logs postgres-test
docker-compose -f docker-compose.testdb.yml logs mysql-test
```

### Port Conflicts

If you get port conflicts, you can change the port mappings in `docker-compose.testdb.yml`:

```yaml
ports:
  - "5434:5432"  # Change 5433 to 5434 if needed
```

### Seed Data Issues

If seed data didn't load properly:

```bash
# Stop and remove volumes
docker-compose -f docker-compose.testdb.yml down -v

# Start fresh
docker-compose -f docker-compose.testdb.yml up -d

# Watch logs to see seeding progress
docker-compose -f docker-compose.testdb.yml logs -f
```

### Memory Issues

If containers are OOMKilled:

```bash
# Increase Docker memory limit in Docker Desktop settings
# Or reduce concurrent databases by commenting out services in docker-compose.testdb.yml
```

## Integration with HowlerOps

### Example Connection Configs

**PostgreSQL:**
```json
{
  "type": "postgres",
  "host": "localhost",
  "port": 5433,
  "database": "testdb",
  "username": "testuser",
  "password": "testpass"
}
```

**MySQL:**
```json
{
  "type": "mysql",
  "host": "localhost",
  "port": 3307,
  "database": "testdb",
  "username": "testuser",
  "password": "testpass"
}
```

**MongoDB:**
```json
{
  "type": "mongodb",
  "host": "localhost",
  "port": 27018,
  "database": "testdb",
  "username": "testuser",
  "password": "testpass",
  "authSource": "admin"
}
```

**ElasticSearch:**
```json
{
  "type": "elasticsearch",
  "host": "localhost",
  "port": 9201
}
```

**ClickHouse:**
```json
{
  "type": "clickhouse",
  "host": "localhost",
  "port": 9001,
  "database": "testdb",
  "username": "testuser",
  "password": "testpass"
}
```

## Maintenance

### Backup Test Data

```bash
# PostgreSQL
docker exec howlerops-postgres-test pg_dump -U testuser testdb > backup_postgres.sql

# MySQL
docker exec howlerops-mysql-test mysqldump -u testuser -ptestpass testdb > backup_mysql.sql

# MongoDB
docker exec howlerops-mongodb-test mongodump --username testuser --password testpass \
  --authenticationDatabase admin --db testdb --out /tmp/backup
```

### Restore from Backup

```bash
# PostgreSQL
docker exec -i howlerops-postgres-test psql -U testuser testdb < backup_postgres.sql

# MySQL
docker exec -i howlerops-mysql-test mysql -u testuser -ptestpass testdb < backup_mysql.sql

# MongoDB
docker exec howlerops-mongodb-test mongorestore --username testuser --password testpass \
  --authenticationDatabase admin --db testdb /tmp/backup/testdb
```

## Contributing

When adding new seed data:

1. Update the schema files (`*-schema.sql`)
2. Update the seed data files (`*-seed*.sql` or `.js`)
3. Update this README with new example queries
4. Test with `docker-compose -f docker-compose.testdb.yml up --build`

## License

This seed data is for testing purposes only. All data is synthetic and randomly generated.
