#!/bin/sh
# ElasticSearch Data Seeding Script

ES_HOST="http://elasticsearch-test:9200"

echo "Waiting for ElasticSearch to be ready..."
until curl -s "$ES_HOST/_cluster/health" > /dev/null; do
    sleep 2
done

echo "ElasticSearch is ready. Creating indexes and seeding data..."

# Create users index with mapping
curl -X PUT "$ES_HOST/users" -H 'Content-Type: application/json' -d '{
  "mappings": {
    "properties": {
      "username": { "type": "keyword" },
      "email": { "type": "keyword" },
      "full_name": { "type": "text" },
      "status": { "type": "keyword" },
      "role": { "type": "keyword" },
      "created_at": { "type": "date" },
      "last_login": { "type": "date" },
      "metadata": { "type": "object" }
    }
  }
}'

# Create products index with mapping
curl -X PUT "$ES_HOST/products" -H 'Content-Type: application/json' -d '{
  "mappings": {
    "properties": {
      "sku": { "type": "keyword" },
      "name": { "type": "text", "fields": { "keyword": { "type": "keyword" } } },
      "description": { "type": "text" },
      "category": { "type": "keyword" },
      "price": { "type": "float" },
      "stock_quantity": { "type": "integer" },
      "status": { "type": "keyword" },
      "created_at": { "type": "date" },
      "metadata": { "type": "object" }
    }
  }
}'

# Create orders index with mapping
curl -X PUT "$ES_HOST/orders" -H 'Content-Type: application/json' -d '{
  "mappings": {
    "properties": {
      "order_number": { "type": "keyword" },
      "user_id": { "type": "keyword" },
      "status": { "type": "keyword" },
      "total_amount": { "type": "float" },
      "payment_method": { "type": "keyword" },
      "created_at": { "type": "date" },
      "shipped_at": { "type": "date" },
      "delivered_at": { "type": "date" },
      "items": { "type": "nested" },
      "shipping_address": { "type": "object" },
      "metadata": { "type": "object" }
    }
  }
}'

# Create analytics_events index with mapping
curl -X PUT "$ES_HOST/analytics_events" -H 'Content-Type: application/json' -d '{
  "mappings": {
    "properties": {
      "event_type": { "type": "keyword" },
      "user_id": { "type": "keyword" },
      "event_data": { "type": "object" },
      "created_at": { "type": "date" }
    }
  }
}'

echo "Indexes created. Bulk indexing data..."

# Seed users (150 users)
# Note: ElasticSearch bulk API requires newline-delimited JSON
cat > /tmp/users_bulk.json << 'EOF'
EOF

for i in $(seq 1 150); do
    status="active"
    [ $((i % 20)) -eq 0 ] && status="inactive"
    [ $((i % 30)) -eq 0 ] && status="suspended"

    role="user"
    [ $((i % 25)) -eq 0 ] && role="admin"
    [ $((i % 10)) -eq 0 ] && role="manager"

    cat >> /tmp/users_bulk.json << EOF
{"index":{"_index":"users","_id":"$i"}}
{"username":"user$i","email":"user$i@example.com","full_name":"User $i","status":"$status","role":"$role","created_at":"$(date -u -d "-$i days" +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -v-${i}d +%Y-%m-%dT%H:%M:%SZ)","metadata":{"signup_source":"web","account_tier":"free"}}
EOF
done

curl -X POST "$ES_HOST/_bulk" -H 'Content-Type: application/x-ndjson' --data-binary @/tmp/users_bulk.json
echo "Users indexed."

# Seed products (250 products)
cat > /tmp/products_bulk.json << 'EOF'
EOF

for i in $(seq 1 250); do
    status="active"
    [ $((i % 30)) -eq 0 ] && status="inactive"
    [ $((i % 40)) -eq 0 ] && status="discontinued"

    category="Other"
    case $((i % 8)) in
        0) category="Electronics";;
        1) category="Clothing";;
        2) category="Home & Garden";;
        3) category="Sports";;
        4) category="Books";;
        5) category="Toys";;
        6) category="Food";;
    esac

    price=$(awk -v seed=$RANDOM 'BEGIN{srand(seed); printf "%.2f", 10 + rand() * 990}')
    stock=$((RANDOM % 1000))

    cat >> /tmp/products_bulk.json << EOF
{"index":{"_index":"products","_id":"$i"}}
{"sku":"SKU-$(printf '%06d' $i)","name":"Product $i","description":"Detailed description for product $i","category":"$category","price":$price,"stock_quantity":$stock,"status":"$status","created_at":"$(date -u +%Y-%m-%dT%H:%M:%SZ)","metadata":{"rating":4.5}}
EOF
done

curl -X POST "$ES_HOST/_bulk" -H 'Content-Type: application/x-ndjson' --data-binary @/tmp/products_bulk.json
echo "Products indexed."

# Seed orders (600 orders)
cat > /tmp/orders_bulk.json << 'EOF'
EOF

for i in $(seq 1 600); do
    status="delivered"
    case $((i % 10)) in
        0) status="pending";;
        1) status="processing";;
        2|3) status="shipped";;
        8) status="cancelled";;
        9) status="refunded";;
    esac

    total=$(awk -v seed=$RANDOM 'BEGIN{srand(seed); printf "%.2f", 20 + rand() * 980}')
    user_id=$((1 + RANDOM % 150))

    cat >> /tmp/orders_bulk.json << EOF
{"index":{"_index":"orders","_id":"$i"}}
{"order_number":"ORD-$(date -u +%Y%m%d)-$(printf '%06d' $i)","user_id":"$user_id","status":"$status","total_amount":$total,"payment_method":"credit_card","created_at":"$(date -u -d "-$((i/10)) hours" +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -v-$((i/10))H +%Y-%m-%dT%H:%M:%SZ)","items":[{"product_id":"1","quantity":2,"price":50.00}],"shipping_address":{"city":"City 1","state":"CA"},"metadata":{"priority":"normal"}}
EOF
done

curl -X POST "$ES_HOST/_bulk" -H 'Content-Type: application/x-ndjson' --data-binary @/tmp/orders_bulk.json
echo "Orders indexed."

# Seed analytics events (2000 events)
cat > /tmp/events_bulk.json << 'EOF'
EOF

event_types=("page_view" "product_view" "add_to_cart" "search" "login" "logout")

for i in $(seq 1 2000); do
    event_type=${event_types[$((i % 6))]}
    user_id=$((1 + RANDOM % 150))

    cat >> /tmp/events_bulk.json << EOF
{"index":{"_index":"analytics_events","_id":"$i"}}
{"event_type":"$event_type","user_id":"$user_id","event_data":{"page":"/page/1","duration_ms":$((RANDOM % 5000))},"created_at":"$(date -u -d "-$((i/100)) hours" +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -v-$((i/100))H +%Y-%m-%dT%H:%M:%SZ)"}
EOF
done

curl -X POST "$ES_HOST/_bulk" -H 'Content-Type: application/x-ndjson' --data-binary @/tmp/events_bulk.json
echo "Analytics events indexed."

# Clean up temp files
rm -f /tmp/users_bulk.json /tmp/products_bulk.json /tmp/orders_bulk.json /tmp/events_bulk.json

# Refresh indexes
curl -X POST "$ES_HOST/_refresh"

# Print summary
echo ""
echo "=== ElasticSearch Seeding Complete ==="
curl -s "$ES_HOST/_cat/indices?v" | grep -E "(users|products|orders|analytics_events)"
echo "======================================"
