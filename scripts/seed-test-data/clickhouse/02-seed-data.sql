-- ClickHouse Test Data Seeding
-- This script populates the database with realistic test data

-- Seed users (150 users)
INSERT INTO testdb.users (id, uuid, username, email, full_name, password_hash, status, role, created_at, updated_at, last_login, metadata)
SELECT
    number + 1 AS id,
    generateUUIDv4() AS uuid,
    concat('user', toString(number + 1)) AS username,
    concat('user', toString(number + 1), '@example.com') AS email,
    if((number + 1) % 10 = 0, NULL, concat('User ', toString(number + 1))) AS full_name,
    concat('$2a$10$', substring(toString(generateUUIDv4()), 1, 22)) AS password_hash,
    multiIf(
        (number + 1) % 20 = 0, 'inactive',
        (number + 1) % 30 = 0, 'suspended',
        'active'
    ) AS status,
    multiIf(
        (number + 1) % 25 = 0, 'admin',
        (number + 1) % 10 = 0, 'manager',
        'user'
    ) AS role,
    now() - toIntervalDay(number + 1) AS created_at,
    now() AS updated_at,
    if((number + 1) % 5 = 0, NULL, now() - toIntervalHour(rand() % (number + 1))) AS last_login,
    concat('{"signup_source":"',
        multiIf((number + 1) % 3 = 0, 'web', (number + 1) % 3 = 1, 'mobile', 'api'),
        '","preferences":{"newsletter":', if((number + 1) % 2 = 0, 'true', 'false'),
        ',"notifications":', if((number + 1) % 3 = 0, 'true', 'false'),
        '},"account_tier":"', if((number + 1) % 20 = 0, 'premium', 'free'), '"}') AS metadata
FROM system.numbers
LIMIT 150;

-- Seed products (250 products)
INSERT INTO testdb.products (id, uuid, sku, name, description, category, price, cost, stock_quantity, status, created_at, updated_at, metadata)
SELECT
    number + 1 AS id,
    generateUUIDv4() AS uuid,
    concat('SKU-', leftPad(toString(number + 1), 6, '0')) AS sku,
    concat(
        multiIf(
            (number + 1) % 5 = 0, 'Premium Product ',
            (number + 1) % 5 = 1, 'Standard Item ',
            (number + 1) % 5 = 2, 'Budget Option ',
            (number + 1) % 5 = 3, 'Deluxe Edition ',
            'Classic Model '
        ),
        toString(number + 1)
    ) AS name,
    concat('Detailed description for product ', toString(number + 1), '. This is a high-quality item with excellent features and benefits.') AS description,
    multiIf(
        (number + 1) % 8 = 0, 'Electronics',
        (number + 1) % 8 = 1, 'Clothing',
        (number + 1) % 8 = 2, 'Home & Garden',
        (number + 1) % 8 = 3, 'Sports',
        (number + 1) % 8 = 4, 'Books',
        (number + 1) % 8 = 5, 'Toys',
        (number + 1) % 8 = 6, 'Food',
        'Other'
    ) AS category,
    round(10 + rand() / 4294967296 * 990, 2) AS price,
    round(5 + rand() / 4294967296 * 495, 2) AS cost,
    rand() % 1000 AS stock_quantity,
    multiIf(
        (number + 1) % 30 = 0, 'inactive',
        (number + 1) % 40 = 0, 'discontinued',
        'active'
    ) AS status,
    now() - toIntervalDay(rand() % 365) AS created_at,
    now() AS updated_at,
    concat('{"weight_kg":', toString(round(rand() / 4294967296 * 10, 2)),
        ',"dimensions":{"length":', toString(round(rand() / 4294967296 * 100, 1)),
        ',"width":', toString(round(rand() / 4294967296 * 100, 1)),
        ',"height":', toString(round(rand() / 4294967296 * 100, 1)),
        '},"manufacturer":"Manufacturer ', toString((number + 1) % 20 + 1),
        '","rating":', toString(round(3 + rand() / 4294967296 * 2, 1)), '}') AS metadata
FROM system.numbers
LIMIT 250;

-- Seed orders (600 orders)
INSERT INTO testdb.orders (id, uuid, order_number, user_id, status, total_amount, tax_amount, shipping_amount, discount_amount, payment_method, shipping_address, billing_address, created_at, updated_at, shipped_at, delivered_at, metadata)
SELECT
    number + 1 AS id,
    generateUUIDv4() AS uuid,
    concat('ORD-', formatDateTime(now() - toIntervalHour(number + 1), '%Y%m%d'), '-', leftPad(toString(number + 1), 6, '0')) AS order_number,
    (rand() % 150) + 1 AS user_id,
    multiIf(
        (number + 1) % 10 = 0, 'pending',
        (number + 1) % 10 = 1, 'processing',
        (number + 1) % 10 IN (2,3), 'shipped',
        (number + 1) % 10 IN (4,5,6,7), 'delivered',
        (number + 1) % 10 = 8, 'cancelled',
        'refunded'
    ) AS status,
    round(20 + rand() / 4294967296 * 980, 2) AS total_amount,
    round(2 + rand() / 4294967296 * 98, 2) AS tax_amount,
    round(5 + rand() / 4294967296 * 20, 2) AS shipping_amount,
    if((number + 1) % 5 = 0, round(rand() / 4294967296 * 50, 2), 0) AS discount_amount,
    multiIf(
        (number + 1) % 4 = 0, 'credit_card',
        (number + 1) % 4 = 1, 'paypal',
        (number + 1) % 4 = 2, 'bank_transfer',
        'cash'
    ) AS payment_method,
    concat('{"street":"', toString(100 + number + 1), ' Main St","city":"City ', toString((number + 1) % 50 + 1),
        '","state":"', multiIf((number + 1) % 4 = 0, 'CA', (number + 1) % 4 = 1, 'NY', (number + 1) % 4 = 2, 'TX', 'FL'),
        '","zip":"', leftPad(toString(10000 + (number + 1) % 90000), 5, '0'), '","country":"US"}') AS shipping_address,
    concat('{"street":"', toString(100 + number + 1), ' Main St","city":"City ', toString((number + 1) % 50 + 1),
        '","state":"', multiIf((number + 1) % 4 = 0, 'CA', (number + 1) % 4 = 1, 'NY', (number + 1) % 4 = 2, 'TX', 'FL'),
        '","zip":"', leftPad(toString(10000 + (number + 1) % 90000), 5, '0'), '","country":"US"}') AS billing_address,
    now() - toIntervalHour(number + 1) AS created_at,
    now() - toIntervalHour(number + 1) + toIntervalHour(rand() % 10) AS updated_at,
    if((number + 1) % 10 IN (2,3,4,5,6,7), now() - toIntervalHour(number + 1) + toIntervalHour(rand() % 24), NULL) AS shipped_at,
    if((number + 1) % 10 IN (4,5,6,7), now() - toIntervalHour(number + 1) + toIntervalHour(rand() % 72), NULL) AS delivered_at,
    concat('{"customer_note":', if((number + 1) % 3 = 0, '"Please deliver to back door"', 'null'),
        ',"gift_wrap":', if((number + 1) % 5 = 0, 'true', 'false'),
        ',"priority":"', if((number + 1) % 10 = 0, 'high', 'normal'), '"}') AS metadata
FROM system.numbers
LIMIT 600;

-- Seed order items (2-5 items per order = ~2100 items)
INSERT INTO testdb.order_items (id, uuid, order_id, product_id, quantity, unit_price, discount_percent, total_price, created_at, metadata)
SELECT
    rowNumberInAllBlocks() AS id,
    generateUUIDv4() AS uuid,
    o.id AS order_id,
    (rand() % 200) + 1 AS product_id, -- Random active product
    (rand() % 5) + 1 AS quantity,
    round(10 + rand() / 4294967296 * 990, 2) AS unit_price,
    if(rand() % 5 = 0, round(rand() / 4294967296 * 20, 2), 0) AS discount_percent,
    round((10 + rand() / 4294967296 * 990) * ((rand() % 5) + 1) * (1 - if(rand() % 5 = 0, rand() / 4294967296 * 0.2, 0)), 2) AS total_price,
    o.created_at AS created_at,
    concat('{"warehouse_location":"WH-', toString((rand() % 5) + 1),
        '","picked_at":"', formatDateTime(now() - toIntervalHour(rand() % 24), '%Y-%m-%d %H:%M:%S'), '"}') AS metadata
FROM
(
    SELECT id, created_at
    FROM testdb.orders
) AS o
ARRAY JOIN arrayMap(x -> x, range(2 + rand() % 4)) AS item_num;

-- Seed audit logs (1200 entries)
INSERT INTO testdb.audit_logs (id, uuid, user_id, action, entity_type, entity_id, old_values, new_values, ip_address, user_agent, created_at, metadata)
SELECT
    number + 1 AS id,
    generateUUIDv4() AS uuid,
    if(rand() % 10 < 9, (rand() % 150) + 1, NULL) AS user_id,
    multiIf(
        (number + 1) % 6 = 0, 'user.created',
        (number + 1) % 6 = 1, 'user.updated',
        (number + 1) % 6 = 2, 'order.created',
        (number + 1) % 6 = 3, 'order.updated',
        (number + 1) % 6 = 4, 'product.created',
        'product.updated'
    ) AS action,
    multiIf(
        (number + 1) % 3 = 0, 'user',
        (number + 1) % 3 = 1, 'order',
        'product'
    ) AS entity_type,
    toString((number + 1) % 100 + 1) AS entity_id,
    if((number + 1) % 2 = 0, '{"field":"old_value"}', '{}') AS old_values,
    '{"field":"new_value"}' AS new_values,
    concat('192.168.', toString((number + 1) % 255), '.', toString((number + 1) % 255)) AS ip_address,
    'Mozilla/5.0 (compatible; TestAgent/1.0)' AS user_agent,
    now() - toIntervalHour(number + 1) AS created_at,
    concat('{"request_id":"', toString(generateUUIDv4()),
        '","session_id":"sess_', substring(toString(generateUUIDv4()), 1, 16), '"}') AS metadata
FROM system.numbers
LIMIT 1200;

-- Seed analytics events (2000 events)
INSERT INTO testdb.analytics_events (id, event_type, user_id, event_data, created_at)
SELECT
    number + 1 AS id,
    multiIf(
        (number + 1) % 10 = 0, 'page_view',
        (number + 1) % 10 = 1, 'product_view',
        (number + 1) % 10 = 2, 'add_to_cart',
        (number + 1) % 10 = 3, 'remove_from_cart',
        (number + 1) % 10 = 4, 'checkout_started',
        (number + 1) % 10 = 5, 'checkout_completed',
        (number + 1) % 10 = 6, 'search',
        (number + 1) % 10 = 7, 'login',
        (number + 1) % 10 = 8, 'logout',
        'error'
    ) AS event_type,
    if(rand() % 10 < 8, (rand() % 150) + 1, NULL) AS user_id,
    concat('{"page":"/page/', toString((number + 1) % 20 + 1),
        '","product_id":', if((number + 1) % 10 IN (1,2), toString((rand() % 200) + 1), 'null'),
        ',"search_query":', if((number + 1) % 10 = 6, concat('"search term ', toString((number + 1) % 50 + 1), '"'), 'null'),
        ',"error_message":', if((number + 1) % 10 = 9, concat('"Error message ', toString(number + 1), '"'), 'null'),
        ',"duration_ms":', toString(rand() % 5000), '}') AS event_data,
    now() - toIntervalMinute(number + 1) AS created_at
FROM system.numbers
LIMIT 2000;

-- Optimize tables
OPTIMIZE TABLE testdb.users FINAL;
OPTIMIZE TABLE testdb.products FINAL;
OPTIMIZE TABLE testdb.orders FINAL;
OPTIMIZE TABLE testdb.order_items FINAL;
OPTIMIZE TABLE testdb.audit_logs FINAL;
OPTIMIZE TABLE testdb.analytics_events FINAL;
