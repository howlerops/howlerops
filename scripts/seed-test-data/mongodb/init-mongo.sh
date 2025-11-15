#!/bin/bash
# MongoDB Initialization Script
# This script runs the seed.js file to populate the database

echo "Waiting for MongoDB to be ready..."
until mongosh --eval "db.adminCommand('ping')" > /dev/null 2>&1; do
    sleep 2
done

echo "MongoDB is ready. Running seed script..."
mongosh --username testuser --password testpass --authenticationDatabase admin testdb /docker-entrypoint-initdb.d/seed.js

echo "MongoDB seeding complete!"
