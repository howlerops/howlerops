// MongoDB Test Data Seeding
// This script populates the database with realistic test data

// Switch to test database
db = db.getSiblingDB('testdb');

// Helper function to generate random data
function randomInt(min, max) {
    return Math.floor(Math.random() * (max - min + 1)) + min;
}

function randomFloat(min, max, decimals) {
    const value = Math.random() * (max - min) + min;
    return parseFloat(value.toFixed(decimals));
}

function randomElement(array) {
    return array[Math.floor(Math.random() * array.length)];
}

function randomDate(start, end) {
    return new Date(start.getTime() + Math.random() * (end.getTime() - start.getTime()));
}

function subtractDays(date, days) {
    const result = new Date(date);
    result.setDate(result.setDate() - days);
    return result;
}

// Generate users (150 users)
print('Seeding users...');
const users = [];
const userStatuses = ['active', 'inactive', 'suspended'];
const userRoles = ['admin', 'user', 'manager'];
const signupSources = ['web', 'mobile', 'api'];

for (let i = 1; i <= 150; i++) {
    const user = {
        _id: ObjectId(),
        username: `user${i}`,
        email: `user${i}@example.com`,
        fullName: i % 10 === 0 ? null : `User ${i}`,
        passwordHash: `$2a$10$${new ObjectId().str}`,
        status: i % 20 === 0 ? 'inactive' : i % 30 === 0 ? 'suspended' : 'active',
        role: i % 25 === 0 ? 'admin' : i % 10 === 0 ? 'manager' : 'user',
        createdAt: subtractDays(new Date(), i),
        updatedAt: new Date(),
        lastLogin: i % 5 === 0 ? null : randomDate(subtractDays(new Date(), i), new Date()),
        metadata: {
            signupSource: randomElement(signupSources),
            preferences: {
                newsletter: i % 2 === 0,
                notifications: i % 3 === 0
            },
            accountTier: i % 20 === 0 ? 'premium' : 'free'
        }
    };
    users.push(user);
}

db.users.insertMany(users);
print(`Inserted ${users.length} users`);

// Create indexes for users
db.users.createIndex({ email: 1 }, { unique: true });
db.users.createIndex({ username: 1 }, { unique: true });
db.users.createIndex({ status: 1 });
db.users.createIndex({ createdAt: 1 });
db.users.createIndex({ 'metadata.accountTier': 1 });

// Generate products (250 products)
print('Seeding products...');
const products = [];
const categories = ['Electronics', 'Clothing', 'Home & Garden', 'Sports', 'Books', 'Toys', 'Food', 'Other'];
const productTypes = ['Premium Product', 'Standard Item', 'Budget Option', 'Deluxe Edition', 'Classic Model'];

for (let i = 1; i <= 250; i++) {
    const product = {
        _id: ObjectId(),
        sku: `SKU-${String(i).padStart(6, '0')}`,
        name: `${productTypes[i % productTypes.length]} ${i}`,
        description: `Detailed description for product ${i}. This is a high-quality item with excellent features and benefits.`,
        category: categories[i % categories.length],
        price: randomFloat(10, 1000, 2),
        cost: randomFloat(5, 500, 2),
        stockQuantity: randomInt(0, 1000),
        status: i % 30 === 0 ? 'inactive' : i % 40 === 0 ? 'discontinued' : 'active',
        createdAt: randomDate(subtractDays(new Date(), 365), new Date()),
        updatedAt: new Date(),
        metadata: {
            weightKg: randomFloat(0.1, 10, 2),
            dimensions: {
                length: randomFloat(1, 100, 1),
                width: randomFloat(1, 100, 1),
                height: randomFloat(1, 100, 1)
            },
            manufacturer: `Manufacturer ${(i % 20) + 1}`,
            rating: randomFloat(3, 5, 1)
        }
    };
    products.push(product);
}

db.products.insertMany(products);
print(`Inserted ${products.length} products`);

// Create indexes for products
db.products.createIndex({ sku: 1 }, { unique: true });
db.products.createIndex({ category: 1 });
db.products.createIndex({ status: 1 });
db.products.createIndex({ name: 'text' });
db.products.createIndex({ createdAt: 1 });

// Generate orders (600 orders)
print('Seeding orders...');
const orders = [];
const orderStatuses = ['pending', 'processing', 'shipped', 'delivered', 'cancelled', 'refunded'];
const paymentMethods = ['credit_card', 'paypal', 'bank_transfer', 'cash'];
const states = ['CA', 'NY', 'TX', 'FL'];

const activeUsers = users.filter(u => u.status === 'active');
const activeProducts = products.filter(p => p.status === 'active');

for (let i = 1; i <= 600; i++) {
    const user = randomElement(activeUsers);
    const createdAt = subtractDays(new Date(), Math.floor(i / 10));

    const order = {
        _id: ObjectId(),
        orderNumber: `ORD-${createdAt.toISOString().split('T')[0].replace(/-/g, '')}-${String(i).padStart(6, '0')}`,
        userId: user._id,
        userEmail: user.email,
        status: orderStatuses[i % orderStatuses.length],
        totalAmount: randomFloat(20, 1000, 2),
        taxAmount: randomFloat(2, 100, 2),
        shippingAmount: randomFloat(5, 25, 2),
        discountAmount: i % 5 === 0 ? randomFloat(5, 50, 2) : 0,
        paymentMethod: randomElement(paymentMethods),
        shippingAddress: {
            street: `${100 + i} Main St`,
            city: `City ${(i % 50) + 1}`,
            state: randomElement(states),
            zip: String(10000 + (i % 90000)).padStart(5, '0'),
            country: 'US'
        },
        billingAddress: {
            street: `${100 + i} Main St`,
            city: `City ${(i % 50) + 1}`,
            state: randomElement(states),
            zip: String(10000 + (i % 90000)).padStart(5, '0'),
            country: 'US'
        },
        items: [],
        createdAt: createdAt,
        updatedAt: new Date(createdAt.getTime() + randomInt(1, 10) * 3600000),
        shippedAt: [2,3,4,5,6,7].includes(i % 10) ? new Date(createdAt.getTime() + randomInt(1, 24) * 3600000) : null,
        deliveredAt: [4,5,6,7].includes(i % 10) ? new Date(createdAt.getTime() + randomInt(24, 72) * 3600000) : null,
        metadata: {
            customerNote: i % 3 === 0 ? 'Please deliver to back door' : null,
            giftWrap: i % 5 === 0,
            priority: i % 10 === 0 ? 'high' : 'normal'
        }
    };

    // Add order items (2-5 items per order)
    const itemCount = randomInt(2, 5);
    for (let j = 0; j < itemCount; j++) {
        const product = randomElement(activeProducts);
        const quantity = randomInt(1, 5);
        const discountPercent = Math.random() < 0.2 ? randomFloat(0, 20, 2) : 0;

        order.items.push({
            _id: ObjectId(),
            productId: product._id,
            productSku: product.sku,
            productName: product.name,
            quantity: quantity,
            unitPrice: product.price,
            discountPercent: discountPercent,
            totalPrice: product.price * quantity * (1 - discountPercent / 100),
            metadata: {
                warehouseLocation: `WH-${randomInt(1, 5)}`,
                pickedAt: randomDate(createdAt, new Date())
            }
        });
    }

    orders.push(order);
}

db.orders.insertMany(orders);
print(`Inserted ${orders.length} orders`);

// Create indexes for orders
db.orders.createIndex({ orderNumber: 1 }, { unique: true });
db.orders.createIndex({ userId: 1 });
db.orders.createIndex({ status: 1 });
db.orders.createIndex({ createdAt: 1 });
db.orders.createIndex({ 'items.productId': 1 });

// Generate audit logs (1200 entries)
print('Seeding audit logs...');
const auditLogs = [];
const actions = ['user.created', 'user.updated', 'order.created', 'order.updated', 'product.created', 'product.updated'];
const entityTypes = ['user', 'order', 'product'];

for (let i = 1; i <= 1200; i++) {
    const user = Math.random() < 0.9 ? randomElement(users) : null;
    const createdAt = subtractDays(new Date(), Math.floor(i / 50));

    const log = {
        _id: ObjectId(),
        userId: user ? user._id : null,
        action: randomElement(actions),
        entityType: randomElement(entityTypes),
        entityId: String((i % 100) + 1),
        oldValues: i % 2 === 0 ? { field: 'old_value' } : null,
        newValues: { field: 'new_value' },
        ipAddress: `192.168.${i % 255}.${i % 255}`,
        userAgent: 'Mozilla/5.0 (compatible; TestAgent/1.0)',
        createdAt: createdAt,
        metadata: {
            requestId: new ObjectId().str,
            sessionId: `sess_${new ObjectId().str}`
        }
    };
    auditLogs.push(log);
}

db.auditLogs.insertMany(auditLogs);
print(`Inserted ${auditLogs.length} audit logs`);

// Create indexes for audit logs
db.auditLogs.createIndex({ userId: 1 });
db.auditLogs.createIndex({ action: 1 });
db.auditLogs.createIndex({ entityType: 1 });
db.auditLogs.createIndex({ createdAt: 1 });

// Generate sessions (50 active sessions)
print('Seeding sessions...');
const sessions = [];

for (let i = 1; i <= 50; i++) {
    const user = randomElement(activeUsers);
    const createdAt = randomDate(subtractDays(new Date(), 1), new Date());

    const session = {
        _id: ObjectId(),
        sessionId: `sess_${new ObjectId().str}`,
        userId: user._id,
        data: {
            lastActivity: randomDate(createdAt, new Date()),
            ipAddress: `192.168.${i % 255}.${i % 255}`,
            cartItems: i % 3
        },
        createdAt: createdAt,
        updatedAt: randomDate(createdAt, new Date()),
        expiresAt: new Date(Date.now() + randomInt(1, 7) * 86400000)
    };
    sessions.push(session);
}

db.sessions.insertMany(sessions);
print(`Inserted ${sessions.length} sessions`);

// Create indexes for sessions
db.sessions.createIndex({ sessionId: 1 }, { unique: true });
db.sessions.createIndex({ userId: 1 });
db.sessions.createIndex({ expiresAt: 1 });

// Generate analytics events (2000 events)
print('Seeding analytics events...');
const analyticsEvents = [];
const eventTypes = ['page_view', 'product_view', 'add_to_cart', 'remove_from_cart', 'checkout_started', 'checkout_completed', 'search', 'login', 'logout', 'error'];

for (let i = 1; i <= 2000; i++) {
    const user = Math.random() < 0.8 ? randomElement(users) : null;
    const eventType = eventTypes[i % eventTypes.length];
    const createdAt = subtractDays(new Date(), Math.floor(i / 100));

    const event = {
        _id: ObjectId(),
        eventType: eventType,
        userId: user ? user._id : null,
        eventData: {
            page: `/page/${(i % 20) + 1}`,
            productId: [1,2].includes(i % 10) ? randomElement(activeProducts)._id : null,
            searchQuery: i % 10 === 6 ? `search term ${(i % 50) + 1}` : null,
            errorMessage: i % 10 === 9 ? `Error message ${i}` : null,
            durationMs: randomInt(100, 5000)
        },
        createdAt: createdAt
    };
    analyticsEvents.push(event);
}

db.analyticsEvents.insertMany(analyticsEvents);
print(`Inserted ${analyticsEvents.length} analytics events`);

// Create indexes for analytics events
db.analyticsEvents.createIndex({ eventType: 1 });
db.analyticsEvents.createIndex({ userId: 1 });
db.analyticsEvents.createIndex({ createdAt: 1 });

// Print summary
print('\n=== Database Seeding Complete ===');
print(`Users: ${db.users.countDocuments()}`);
print(`Products: ${db.products.countDocuments()}`);
print(`Orders: ${db.orders.countDocuments()}`);
print(`Audit Logs: ${db.auditLogs.countDocuments()}`);
print(`Sessions: ${db.sessions.countDocuments()}`);
print(`Analytics Events: ${db.analyticsEvents.countDocuments()}`);
print('=================================');
