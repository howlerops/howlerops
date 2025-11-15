-- PostgreSQL Test Data Seeding
-- This script populates the database with realistic test data

-- Seed users (150 users)
INSERT INTO users (username, email, full_name, password_hash, status, role, created_at, last_login, metadata)
SELECT
    'user' || n,
    'user' || n || '@example.com',
    CASE WHEN n % 10 = 0 THEN NULL ELSE 'User ' || n END,
    '$2a$10$' || md5(random()::text), -- Mock bcrypt hash
    CASE
        WHEN n % 20 = 0 THEN 'inactive'
        WHEN n % 30 = 0 THEN 'suspended'
        ELSE 'active'
    END,
    CASE
        WHEN n % 25 = 0 THEN 'admin'
        WHEN n % 10 = 0 THEN 'manager'
        ELSE 'user'
    END,
    CURRENT_TIMESTAMP - (n || ' days')::interval,
    CASE WHEN n % 5 = 0 THEN NULL ELSE CURRENT_TIMESTAMP - (random() * n || ' hours')::interval END,
    jsonb_build_object(
        'signup_source', CASE WHEN n % 3 = 0 THEN 'web' WHEN n % 3 = 1 THEN 'mobile' ELSE 'api' END,
        'preferences', jsonb_build_object('newsletter', n % 2 = 0, 'notifications', n % 3 = 0),
        'account_tier', CASE WHEN n % 20 = 0 THEN 'premium' ELSE 'free' END
    )
FROM generate_series(1, 150) AS n;

-- Seed products (250 products)
INSERT INTO products (sku, name, description, category, price, cost, stock_quantity, status, created_at, metadata)
SELECT
    'SKU-' || LPAD(n::text, 6, '0'),
    CASE
        WHEN n % 5 = 0 THEN 'Premium Product ' || n
        WHEN n % 5 = 1 THEN 'Standard Item ' || n
        WHEN n % 5 = 2 THEN 'Budget Option ' || n
        WHEN n % 5 = 3 THEN 'Deluxe Edition ' || n
        ELSE 'Classic Model ' || n
    END,
    'Detailed description for product ' || n || '. This is a high-quality item with excellent features and benefits.',
    CASE
        WHEN n % 8 = 0 THEN 'Electronics'
        WHEN n % 8 = 1 THEN 'Clothing'
        WHEN n % 8 = 2 THEN 'Home & Garden'
        WHEN n % 8 = 3 THEN 'Sports'
        WHEN n % 8 = 4 THEN 'Books'
        WHEN n % 8 = 5 THEN 'Toys'
        WHEN n % 8 = 6 THEN 'Food'
        ELSE 'Other'
    END,
    ROUND((10 + random() * 990)::numeric, 2),
    ROUND((5 + random() * 495)::numeric, 2),
    FLOOR(random() * 1000)::int,
    CASE
        WHEN n % 30 = 0 THEN 'inactive'
        WHEN n % 40 = 0 THEN 'discontinued'
        ELSE 'active'
    END,
    CURRENT_TIMESTAMP - (random() * 365 || ' days')::interval,
    jsonb_build_object(
        'weight_kg', ROUND((random() * 10)::numeric, 2),
        'dimensions', jsonb_build_object('length', ROUND(random() * 100, 1), 'width', ROUND(random() * 100, 1), 'height', ROUND(random() * 100, 1)),
        'manufacturer', 'Manufacturer ' || (n % 20 + 1),
        'rating', ROUND((3 + random() * 2)::numeric, 1)
    )
FROM generate_series(1, 250) AS n;

-- Seed orders (600 orders)
INSERT INTO orders (order_number, user_id, status, total_amount, tax_amount, shipping_amount, discount_amount, payment_method, shipping_address, billing_address, created_at, updated_at, shipped_at, delivered_at, metadata)
SELECT
    'ORD-' || TO_CHAR(CURRENT_TIMESTAMP - (n || ' hours')::interval, 'YYYYMMDD') || '-' || LPAD(n::text, 6, '0'),
    (SELECT id FROM users ORDER BY random() LIMIT 1),
    CASE
        WHEN n % 10 = 0 THEN 'pending'
        WHEN n % 10 = 1 THEN 'processing'
        WHEN n % 10 IN (2,3) THEN 'shipped'
        WHEN n % 10 IN (4,5,6,7) THEN 'delivered'
        WHEN n % 10 = 8 THEN 'cancelled'
        ELSE 'refunded'
    END,
    ROUND((20 + random() * 980)::numeric, 2),
    ROUND((2 + random() * 98)::numeric, 2),
    ROUND((5 + random() * 20)::numeric, 2),
    CASE WHEN n % 5 = 0 THEN ROUND((random() * 50)::numeric, 2) ELSE 0 END,
    CASE
        WHEN n % 4 = 0 THEN 'credit_card'
        WHEN n % 4 = 1 THEN 'paypal'
        WHEN n % 4 = 2 THEN 'bank_transfer'
        ELSE 'cash'
    END,
    jsonb_build_object(
        'street', (100 + n) || ' Main St',
        'city', 'City ' || (n % 50 + 1),
        'state', CASE WHEN n % 4 = 0 THEN 'CA' WHEN n % 4 = 1 THEN 'NY' WHEN n % 4 = 2 THEN 'TX' ELSE 'FL' END,
        'zip', LPAD((10000 + n % 90000)::text, 5, '0'),
        'country', 'US'
    ),
    jsonb_build_object(
        'street', (100 + n) || ' Main St',
        'city', 'City ' || (n % 50 + 1),
        'state', CASE WHEN n % 4 = 0 THEN 'CA' WHEN n % 4 = 1 THEN 'NY' WHEN n % 4 = 2 THEN 'TX' ELSE 'FL' END,
        'zip', LPAD((10000 + n % 90000)::text, 5, '0'),
        'country', 'US'
    ),
    CURRENT_TIMESTAMP - (n || ' hours')::interval,
    CURRENT_TIMESTAMP - (n || ' hours')::interval + (random() * 10 || ' hours')::interval,
    CASE WHEN n % 10 IN (2,3,4,5,6,7) THEN CURRENT_TIMESTAMP - (n || ' hours')::interval + (random() * 24 || ' hours')::interval ELSE NULL END,
    CASE WHEN n % 10 IN (4,5,6,7) THEN CURRENT_TIMESTAMP - (n || ' hours')::interval + (random() * 72 || ' hours')::interval ELSE NULL END,
    jsonb_build_object(
        'customer_note', CASE WHEN n % 3 = 0 THEN 'Please deliver to back door' ELSE NULL END,
        'gift_wrap', n % 5 = 0,
        'priority', CASE WHEN n % 10 = 0 THEN 'high' ELSE 'normal' END
    )
FROM generate_series(1, 600) AS n;

-- Seed order items (1500 items across orders)
INSERT INTO order_items (order_id, product_id, quantity, unit_price, discount_percent, total_price, metadata)
SELECT
    o.id,
    (SELECT id FROM products WHERE status = 'active' ORDER BY random() LIMIT 1),
    FLOOR(1 + random() * 5)::int,
    p.price,
    CASE WHEN random() < 0.2 THEN ROUND((random() * 20)::numeric, 2) ELSE 0 END,
    ROUND((p.price * FLOOR(1 + random() * 5) * (1 - CASE WHEN random() < 0.2 THEN random() * 0.2 ELSE 0 END))::numeric, 2),
    jsonb_build_object(
        'warehouse_location', 'WH-' || (1 + FLOOR(random() * 5)::int),
        'picked_at', CURRENT_TIMESTAMP - (random() * 24 || ' hours')::interval
    )
FROM orders o
CROSS JOIN LATERAL (
    SELECT id, price FROM products WHERE status = 'active' ORDER BY random() LIMIT (1 + FLOOR(random() * 4)::int)
) p;

-- Seed audit logs (1200 entries)
INSERT INTO audit_logs (user_id, action, entity_type, entity_id, old_values, new_values, ip_address, user_agent, created_at, metadata)
SELECT
    CASE WHEN random() < 0.9 THEN (SELECT id FROM users ORDER BY random() LIMIT 1) ELSE NULL END,
    CASE
        WHEN n % 6 = 0 THEN 'user.created'
        WHEN n % 6 = 1 THEN 'user.updated'
        WHEN n % 6 = 2 THEN 'order.created'
        WHEN n % 6 = 3 THEN 'order.updated'
        WHEN n % 6 = 4 THEN 'product.created'
        ELSE 'product.updated'
    END,
    CASE
        WHEN n % 3 = 0 THEN 'user'
        WHEN n % 3 = 1 THEN 'order'
        ELSE 'product'
    END,
    (n % 100 + 1)::text,
    CASE WHEN n % 2 = 0 THEN jsonb_build_object('field', 'old_value') ELSE NULL END,
    jsonb_build_object('field', 'new_value'),
    ('192.168.' || (n % 255) || '.' || (n % 255))::inet,
    'Mozilla/5.0 (compatible; TestAgent/1.0)',
    CURRENT_TIMESTAMP - (n || ' hours')::interval,
    jsonb_build_object(
        'request_id', uuid_generate_v4(),
        'session_id', 'sess_' || md5(random()::text)
    )
FROM generate_series(1, 1200) AS n;

-- Seed sessions (50 active sessions)
INSERT INTO sessions (session_id, user_id, data, created_at, updated_at, expires_at)
SELECT
    'sess_' || md5(n::text),
    (SELECT id FROM users WHERE status = 'active' ORDER BY random() LIMIT 1),
    jsonb_build_object(
        'last_activity', CURRENT_TIMESTAMP - (random() * 60 || ' minutes')::interval,
        'ip_address', '192.168.' || (n % 255) || '.' || (n % 255),
        'cart_items', n % 3
    ),
    CURRENT_TIMESTAMP - (random() * 24 || ' hours')::interval,
    CURRENT_TIMESTAMP - (random() * 60 || ' minutes')::interval,
    CURRENT_TIMESTAMP + (random() * 7 || ' days')::interval
FROM generate_series(1, 50) AS n;

-- Seed analytics events (2000 events)
INSERT INTO analytics_events (event_type, user_id, event_data, created_at)
SELECT
    CASE
        WHEN n % 10 = 0 THEN 'page_view'
        WHEN n % 10 = 1 THEN 'product_view'
        WHEN n % 10 = 2 THEN 'add_to_cart'
        WHEN n % 10 = 3 THEN 'remove_from_cart'
        WHEN n % 10 = 4 THEN 'checkout_started'
        WHEN n % 10 = 5 THEN 'checkout_completed'
        WHEN n % 10 = 6 THEN 'search'
        WHEN n % 10 = 7 THEN 'login'
        WHEN n % 10 = 8 THEN 'logout'
        ELSE 'error'
    END,
    CASE WHEN random() < 0.8 THEN (SELECT id FROM users ORDER BY random() LIMIT 1) ELSE NULL END,
    jsonb_build_object(
        'page', '/page/' || (n % 20 + 1),
        'product_id', CASE WHEN n % 10 IN (1,2) THEN (SELECT id FROM products ORDER BY random() LIMIT 1) ELSE NULL END,
        'search_query', CASE WHEN n % 10 = 6 THEN 'search term ' || (n % 50 + 1) ELSE NULL END,
        'error_message', CASE WHEN n % 10 = 9 THEN 'Error message ' || n ELSE NULL END,
        'duration_ms', FLOOR(random() * 5000)::int
    ),
    CURRENT_TIMESTAMP - (n || ' minutes')::interval
FROM generate_series(1, 2000) AS n;

-- Refresh materialized view
REFRESH MATERIALIZED VIEW product_analytics;

-- Add some statistics
ANALYZE users;
ANALYZE products;
ANALYZE orders;
ANALYZE order_items;
ANALYZE audit_logs;
ANALYZE sessions;
ANALYZE analytics_events;
