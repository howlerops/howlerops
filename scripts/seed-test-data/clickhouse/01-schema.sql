-- ClickHouse Test Database Schema
-- This script creates the database schema for testing HowlerOps

-- Users table
CREATE TABLE IF NOT EXISTS testdb.users (
    id UInt32,
    uuid UUID,
    username String,
    email String,
    full_name Nullable(String),
    password_hash String,
    status LowCardinality(String),
    role LowCardinality(String),
    created_at DateTime,
    updated_at DateTime,
    last_login Nullable(DateTime),
    metadata String -- JSON stored as String
) ENGINE = MergeTree()
ORDER BY (id, created_at)
SETTINGS index_granularity = 8192;

-- Products table
CREATE TABLE IF NOT EXISTS testdb.products (
    id UInt32,
    uuid UUID,
    sku String,
    name String,
    description String,
    category LowCardinality(String),
    price Decimal(10, 2),
    cost Decimal(10, 2),
    stock_quantity Int32,
    status LowCardinality(String),
    created_at DateTime,
    updated_at DateTime,
    metadata String -- JSON stored as String
) ENGINE = MergeTree()
ORDER BY (id, category, created_at)
SETTINGS index_granularity = 8192;

-- Orders table
CREATE TABLE IF NOT EXISTS testdb.orders (
    id UInt32,
    uuid UUID,
    order_number String,
    user_id UInt32,
    status LowCardinality(String),
    total_amount Decimal(10, 2),
    tax_amount Decimal(10, 2),
    shipping_amount Decimal(10, 2),
    discount_amount Decimal(10, 2),
    payment_method LowCardinality(String),
    shipping_address String, -- JSON stored as String
    billing_address String, -- JSON stored as String
    created_at DateTime,
    updated_at DateTime,
    shipped_at Nullable(DateTime),
    delivered_at Nullable(DateTime),
    metadata String -- JSON stored as String
) ENGINE = MergeTree()
ORDER BY (user_id, created_at, id)
SETTINGS index_granularity = 8192;

-- Order items table
CREATE TABLE IF NOT EXISTS testdb.order_items (
    id UInt32,
    uuid UUID,
    order_id UInt32,
    product_id UInt32,
    quantity Int32,
    unit_price Decimal(10, 2),
    discount_percent Decimal(5, 2),
    total_price Decimal(10, 2),
    created_at DateTime,
    metadata String -- JSON stored as String
) ENGINE = MergeTree()
ORDER BY (order_id, product_id, id)
SETTINGS index_granularity = 8192;

-- Audit logs table
CREATE TABLE IF NOT EXISTS testdb.audit_logs (
    id UInt32,
    uuid UUID,
    user_id Nullable(UInt32),
    action String,
    entity_type LowCardinality(String),
    entity_id String,
    old_values String, -- JSON stored as String
    new_values String, -- JSON stored as String
    ip_address String,
    user_agent String,
    created_at DateTime,
    metadata String -- JSON stored as String
) ENGINE = MergeTree()
ORDER BY (created_at, user_id, id)
SETTINGS index_granularity = 8192;

-- Analytics events table (optimized for time-series data)
CREATE TABLE IF NOT EXISTS testdb.analytics_events (
    id UInt32,
    event_type LowCardinality(String),
    user_id Nullable(UInt32),
    event_data String, -- JSON stored as String
    created_at DateTime
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(created_at)
ORDER BY (event_type, created_at, id)
SETTINGS index_granularity = 8192;

-- Create materialized view for order summaries
CREATE MATERIALIZED VIEW IF NOT EXISTS testdb.order_summaries
ENGINE = AggregatingMergeTree()
ORDER BY (order_id, user_id)
AS SELECT
    o.id as order_id,
    o.order_number,
    o.created_at,
    o.user_id,
    o.status,
    o.total_amount,
    countState() as item_count,
    sumState(oi.quantity) as total_items
FROM testdb.orders o
LEFT JOIN testdb.order_items oi ON o.id = oi.order_id
GROUP BY order_id, order_number, created_at, user_id, status, total_amount;

-- Create materialized view for product analytics
CREATE MATERIALIZED VIEW IF NOT EXISTS testdb.product_analytics
ENGINE = SummingMergeTree()
ORDER BY (product_id, category)
AS SELECT
    p.id as product_id,
    p.sku,
    p.name,
    p.category,
    p.price,
    count(DISTINCT oi.order_id) as order_count,
    sum(oi.quantity) as total_sold,
    sum(oi.total_price) as revenue
FROM testdb.products p
LEFT JOIN testdb.order_items oi ON p.id = oi.product_id
GROUP BY product_id, sku, name, category, price;
