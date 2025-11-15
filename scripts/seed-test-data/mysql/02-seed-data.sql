-- MySQL Test Data Seeding
-- This script populates the database with realistic test data

USE testdb;

-- Create procedure to generate random data
DELIMITER $$

CREATE PROCEDURE generate_test_data()
BEGIN
    DECLARE i INT DEFAULT 1;
    DECLARE user_count INT DEFAULT 150;
    DECLARE product_count INT DEFAULT 250;
    DECLARE order_count INT DEFAULT 600;
    DECLARE max_user_id INT;
    DECLARE max_product_id INT;
    DECLARE rand_user_id INT;
    DECLARE rand_product_id INT;
    DECLARE rand_order_id INT;
    DECLARE order_item_count INT;
    DECLARE j INT;

    -- Seed users
    WHILE i <= user_count DO
        INSERT INTO users (username, email, full_name, password_hash, status, role, created_at, last_login, metadata)
        VALUES (
            CONCAT('user', i),
            CONCAT('user', i, '@example.com'),
            IF(i % 10 = 0, NULL, CONCAT('User ', i)),
            CONCAT('$2a$10$', MD5(RAND())),
            CASE
                WHEN i % 20 = 0 THEN 'inactive'
                WHEN i % 30 = 0 THEN 'suspended'
                ELSE 'active'
            END,
            CASE
                WHEN i % 25 = 0 THEN 'admin'
                WHEN i % 10 = 0 THEN 'manager'
                ELSE 'user'
            END,
            DATE_SUB(NOW(), INTERVAL i DAY),
            IF(i % 5 = 0, NULL, DATE_SUB(NOW(), INTERVAL FLOOR(RAND() * i) HOUR)),
            JSON_OBJECT(
                'signup_source', CASE WHEN i % 3 = 0 THEN 'web' WHEN i % 3 = 1 THEN 'mobile' ELSE 'api' END,
                'preferences', JSON_OBJECT('newsletter', i % 2 = 0, 'notifications', i % 3 = 0),
                'account_tier', IF(i % 20 = 0, 'premium', 'free')
            )
        );
        SET i = i + 1;
    END WHILE;

    -- Seed products
    SET i = 1;
    WHILE i <= product_count DO
        INSERT INTO products (sku, name, description, category, price, cost, stock_quantity, status, created_at, metadata)
        VALUES (
            CONCAT('SKU-', LPAD(i, 6, '0')),
            CONCAT(
                CASE
                    WHEN i % 5 = 0 THEN 'Premium Product '
                    WHEN i % 5 = 1 THEN 'Standard Item '
                    WHEN i % 5 = 2 THEN 'Budget Option '
                    WHEN i % 5 = 3 THEN 'Deluxe Edition '
                    ELSE 'Classic Model '
                END,
                i
            ),
            CONCAT('Detailed description for product ', i, '. This is a high-quality item with excellent features and benefits.'),
            CASE
                WHEN i % 8 = 0 THEN 'Electronics'
                WHEN i % 8 = 1 THEN 'Clothing'
                WHEN i % 8 = 2 THEN 'Home & Garden'
                WHEN i % 8 = 3 THEN 'Sports'
                WHEN i % 8 = 4 THEN 'Books'
                WHEN i % 8 = 5 THEN 'Toys'
                WHEN i % 8 = 6 THEN 'Food'
                ELSE 'Other'
            END,
            ROUND(10 + RAND() * 990, 2),
            ROUND(5 + RAND() * 495, 2),
            FLOOR(RAND() * 1000),
            CASE
                WHEN i % 30 = 0 THEN 'inactive'
                WHEN i % 40 = 0 THEN 'discontinued'
                ELSE 'active'
            END,
            DATE_SUB(NOW(), INTERVAL FLOOR(RAND() * 365) DAY),
            JSON_OBJECT(
                'weight_kg', ROUND(RAND() * 10, 2),
                'dimensions', JSON_OBJECT('length', ROUND(RAND() * 100, 1), 'width', ROUND(RAND() * 100, 1), 'height', ROUND(RAND() * 100, 1)),
                'manufacturer', CONCAT('Manufacturer ', (i % 20 + 1)),
                'rating', ROUND(3 + RAND() * 2, 1)
            )
        );
        SET i = i + 1;
    END WHILE;

    -- Get max IDs for random selection
    SELECT MAX(id) INTO max_user_id FROM users;
    SELECT MAX(id) INTO max_product_id FROM products WHERE status = 'active';

    -- Seed orders
    SET i = 1;
    WHILE i <= order_count DO
        SET rand_user_id = FLOOR(1 + RAND() * max_user_id);

        INSERT INTO orders (order_number, user_id, status, total_amount, tax_amount, shipping_amount, discount_amount, payment_method, shipping_address, billing_address, created_at, updated_at, shipped_at, delivered_at, metadata)
        VALUES (
            CONCAT('ORD-', DATE_FORMAT(DATE_SUB(NOW(), INTERVAL i HOUR), '%Y%m%d'), '-', LPAD(i, 6, '0')),
            rand_user_id,
            CASE
                WHEN i % 10 = 0 THEN 'pending'
                WHEN i % 10 = 1 THEN 'processing'
                WHEN i % 10 IN (2,3) THEN 'shipped'
                WHEN i % 10 IN (4,5,6,7) THEN 'delivered'
                WHEN i % 10 = 8 THEN 'cancelled'
                ELSE 'refunded'
            END,
            ROUND(20 + RAND() * 980, 2),
            ROUND(2 + RAND() * 98, 2),
            ROUND(5 + RAND() * 20, 2),
            IF(i % 5 = 0, ROUND(RAND() * 50, 2), 0),
            CASE
                WHEN i % 4 = 0 THEN 'credit_card'
                WHEN i % 4 = 1 THEN 'paypal'
                WHEN i % 4 = 2 THEN 'bank_transfer'
                ELSE 'cash'
            END,
            JSON_OBJECT(
                'street', CONCAT((100 + i), ' Main St'),
                'city', CONCAT('City ', (i % 50 + 1)),
                'state', CASE WHEN i % 4 = 0 THEN 'CA' WHEN i % 4 = 1 THEN 'NY' WHEN i % 4 = 2 THEN 'TX' ELSE 'FL' END,
                'zip', LPAD((10000 + i % 90000), 5, '0'),
                'country', 'US'
            ),
            JSON_OBJECT(
                'street', CONCAT((100 + i), ' Main St'),
                'city', CONCAT('City ', (i % 50 + 1)),
                'state', CASE WHEN i % 4 = 0 THEN 'CA' WHEN i % 4 = 1 THEN 'NY' WHEN i % 4 = 2 THEN 'TX' ELSE 'FL' END,
                'zip', LPAD((10000 + i % 90000), 5, '0'),
                'country', 'US'
            ),
            DATE_SUB(NOW(), INTERVAL i HOUR),
            DATE_ADD(DATE_SUB(NOW(), INTERVAL i HOUR), INTERVAL FLOOR(RAND() * 10) HOUR),
            IF(i % 10 IN (2,3,4,5,6,7), DATE_ADD(DATE_SUB(NOW(), INTERVAL i HOUR), INTERVAL FLOOR(RAND() * 24) HOUR), NULL),
            IF(i % 10 IN (4,5,6,7), DATE_ADD(DATE_SUB(NOW(), INTERVAL i HOUR), INTERVAL FLOOR(RAND() * 72) HOUR), NULL),
            JSON_OBJECT(
                'customer_note', IF(i % 3 = 0, 'Please deliver to back door', NULL),
                'gift_wrap', i % 5 = 0,
                'priority', IF(i % 10 = 0, 'high', 'normal')
            )
        );
        SET i = i + 1;
    END WHILE;

    -- Seed order items (2-5 items per order)
    SET i = 1;
    WHILE i <= order_count DO
        SET order_item_count = 2 + FLOOR(RAND() * 4);
        SET j = 1;

        WHILE j <= order_item_count DO
            SET rand_product_id = FLOOR(1 + RAND() * max_product_id);

            INSERT INTO order_items (order_id, product_id, quantity, unit_price, discount_percent, total_price, metadata)
            SELECT
                i,
                rand_product_id,
                FLOOR(1 + RAND() * 5),
                price,
                IF(RAND() < 0.2, ROUND(RAND() * 20, 2), 0),
                ROUND(price * FLOOR(1 + RAND() * 5) * (1 - IF(RAND() < 0.2, RAND() * 0.2, 0)), 2),
                JSON_OBJECT(
                    'warehouse_location', CONCAT('WH-', (1 + FLOOR(RAND() * 5))),
                    'picked_at', DATE_SUB(NOW(), INTERVAL FLOOR(RAND() * 24) HOUR)
                )
            FROM products
            WHERE id = rand_product_id
            LIMIT 1;

            SET j = j + 1;
        END WHILE;
        SET i = i + 1;
    END WHILE;

    -- Seed audit logs
    SET i = 1;
    WHILE i <= 1200 DO
        SET rand_user_id = IF(RAND() < 0.9, FLOOR(1 + RAND() * max_user_id), NULL);

        INSERT INTO audit_logs (user_id, action, entity_type, entity_id, old_values, new_values, ip_address, user_agent, created_at, metadata)
        VALUES (
            rand_user_id,
            CASE
                WHEN i % 6 = 0 THEN 'user.created'
                WHEN i % 6 = 1 THEN 'user.updated'
                WHEN i % 6 = 2 THEN 'order.created'
                WHEN i % 6 = 3 THEN 'order.updated'
                WHEN i % 6 = 4 THEN 'product.created'
                ELSE 'product.updated'
            END,
            CASE
                WHEN i % 3 = 0 THEN 'user'
                WHEN i % 3 = 1 THEN 'order'
                ELSE 'product'
            END,
            (i % 100 + 1),
            IF(i % 2 = 0, JSON_OBJECT('field', 'old_value'), NULL),
            JSON_OBJECT('field', 'new_value'),
            CONCAT('192.168.', (i % 255), '.', (i % 255)),
            'Mozilla/5.0 (compatible; TestAgent/1.0)',
            DATE_SUB(NOW(), INTERVAL i HOUR),
            JSON_OBJECT(
                'request_id', UUID(),
                'session_id', CONCAT('sess_', MD5(RAND()))
            )
        );
        SET i = i + 1;
    END WHILE;

    -- Seed sessions
    SET i = 1;
    WHILE i <= 50 DO
        SET rand_user_id = (SELECT id FROM users WHERE status = 'active' ORDER BY RAND() LIMIT 1);

        INSERT INTO sessions (session_id, user_id, data, created_at, updated_at, expires_at)
        VALUES (
            CONCAT('sess_', MD5(i)),
            rand_user_id,
            JSON_OBJECT(
                'last_activity', DATE_SUB(NOW(), INTERVAL FLOOR(RAND() * 60) MINUTE),
                'ip_address', CONCAT('192.168.', (i % 255), '.', (i % 255)),
                'cart_items', i % 3
            ),
            DATE_SUB(NOW(), INTERVAL FLOOR(RAND() * 24) HOUR),
            DATE_SUB(NOW(), INTERVAL FLOOR(RAND() * 60) MINUTE),
            DATE_ADD(NOW(), INTERVAL FLOOR(RAND() * 7) DAY)
        );
        SET i = i + 1;
    END WHILE;

    -- Seed analytics events
    SET i = 1;
    WHILE i <= 2000 DO
        SET rand_user_id = IF(RAND() < 0.8, FLOOR(1 + RAND() * max_user_id), NULL);

        INSERT INTO analytics_events (event_type, user_id, event_data, created_at)
        VALUES (
            CASE
                WHEN i % 10 = 0 THEN 'page_view'
                WHEN i % 10 = 1 THEN 'product_view'
                WHEN i % 10 = 2 THEN 'add_to_cart'
                WHEN i % 10 = 3 THEN 'remove_from_cart'
                WHEN i % 10 = 4 THEN 'checkout_started'
                WHEN i % 10 = 5 THEN 'checkout_completed'
                WHEN i % 10 = 6 THEN 'search'
                WHEN i % 10 = 7 THEN 'login'
                WHEN i % 10 = 8 THEN 'logout'
                ELSE 'error'
            END,
            rand_user_id,
            JSON_OBJECT(
                'page', CONCAT('/page/', (i % 20 + 1)),
                'product_id', IF(i % 10 IN (1,2), FLOOR(1 + RAND() * max_product_id), NULL),
                'search_query', IF(i % 10 = 6, CONCAT('search term ', (i % 50 + 1)), NULL),
                'error_message', IF(i % 10 = 9, CONCAT('Error message ', i), NULL),
                'duration_ms', FLOOR(RAND() * 5000)
            ),
            DATE_SUB(NOW(), INTERVAL i MINUTE)
        );
        SET i = i + 1;
    END WHILE;

END$$

DELIMITER ;

-- Execute the procedure
CALL generate_test_data();

-- Clean up
DROP PROCEDURE generate_test_data;

-- Optimize tables
OPTIMIZE TABLE users;
OPTIMIZE TABLE products;
OPTIMIZE TABLE orders;
OPTIMIZE TABLE order_items;
OPTIMIZE TABLE audit_logs;
OPTIMIZE TABLE sessions;
OPTIMIZE TABLE analytics_events;
