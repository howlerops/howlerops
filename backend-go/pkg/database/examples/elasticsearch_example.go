package examples

import (
	"context"
	"fmt"
	"log"

	"github.com/sirupsen/logrus"
	"github.com/sql-studio/backend-go/pkg/database"
)

// ElasticsearchExample demonstrates how to use the Elasticsearch connector
func ElasticsearchExample() {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Create Elasticsearch connection configuration
	config := database.ConnectionConfig{
		Type:       database.Elasticsearch,
		Host:       "localhost",
		Port:       9200,
		Database:   "my-elasticsearch", // Logical name for the connection
		Username:   "elastic",          // Optional
		Password:   "password",         // Optional
		SSLMode:    "disable",          // Use "require" for HTTPS
		Parameters: map[string]string{
			// Optionally use API key instead of username/password
			// "api_key": "your-base64-encoded-api-key",
		},
	}

	// Create Elasticsearch database instance
	es, err := database.NewElasticsearchDatabase(config, logger)
	if err != nil {
		log.Fatalf("Failed to create Elasticsearch connection: %v", err)
	}
	defer es.Disconnect()

	ctx := context.Background()

	// 1. Test connection
	fmt.Println("Testing connection...")
	if err := es.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping Elasticsearch: %v", err)
	}
	fmt.Println("Connection successful!")

	// 2. Get connection information
	fmt.Println("\nGetting connection info...")
	info, err := es.GetConnectionInfo(ctx)
	if err != nil {
		log.Fatalf("Failed to get connection info: %v", err)
	}
	fmt.Printf("Elasticsearch info: %+v\n", info)

	// 3. List schemas (in Elasticsearch, this returns ["default"])
	fmt.Println("\nListing schemas...")
	schemas, err := es.GetSchemas(ctx)
	if err != nil {
		log.Fatalf("Failed to get schemas: %v", err)
	}
	fmt.Printf("Schemas: %v\n", schemas)

	// 4. List indices (tables)
	fmt.Println("\nListing indices...")
	tables, err := es.GetTables(ctx, "default")
	if err != nil {
		log.Fatalf("Failed to get tables: %v", err)
	}
	for _, table := range tables {
		fmt.Printf("Index: %s, Docs: %d, Size: %d bytes\n",
			table.Name, table.RowCount, table.SizeBytes)
	}

	// 5. Get index structure
	if len(tables) > 0 {
		indexName := tables[0].Name
		fmt.Printf("\nGetting structure for index: %s\n", indexName)
		structure, err := es.GetTableStructure(ctx, "default", indexName)
		if err != nil {
			log.Fatalf("Failed to get table structure: %v", err)
		}
		fmt.Printf("Index: %s\n", structure.Table.Name)
		fmt.Println("Fields:")
		for _, col := range structure.Columns {
			fmt.Printf("  - %s (%s)\n", col.Name, col.DataType)
		}
	}

	// 6. Execute SQL query using Elasticsearch SQL API
	fmt.Println("\nExecuting SQL query...")
	query := "SELECT * FROM my-index LIMIT 10"
	result, err := es.Execute(ctx, query)
	if err != nil {
		log.Fatalf("Failed to execute query: %v", err)
	}
	fmt.Printf("Columns: %v\n", result.Columns)
	fmt.Printf("Rows returned: %d\n", result.RowCount)
	fmt.Printf("Duration: %v\n", result.Duration)

	// Print first few rows
	for i, row := range result.Rows {
		if i >= 3 {
			break
		}
		fmt.Printf("Row %d: %v\n", i+1, row)
	}

	// 7. Stream large result sets
	fmt.Println("\nStreaming query results...")
	streamQuery := "SELECT * FROM my-index"
	err = es.ExecuteStream(ctx, streamQuery, 100, func(batch [][]interface{}) error {
		fmt.Printf("Received batch of %d rows\n", len(batch))
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to stream query: %v", err)
	}

	// 8. Explain query (get Query DSL translation)
	fmt.Println("\nExplaining query...")
	explainQuery := "SELECT name, age FROM users WHERE age > 25"
	plan, err := es.ExplainQuery(ctx, explainQuery)
	if err != nil {
		log.Printf("Failed to explain query: %v", err)
	} else {
		fmt.Printf("Query DSL:\n%s\n", plan)
	}
}

// OpenSearchExample demonstrates using the same connector for OpenSearch
func OpenSearchExample() {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// OpenSearch uses the same connector as Elasticsearch
	config := database.ConnectionConfig{
		Type:     database.OpenSearch, // Use OpenSearch type
		Host:     "localhost",
		Port:     9200,
		Database: "my-opensearch",
		Username: "admin",
		Password: "admin",
		SSLMode:  "disable",
	}

	// The rest is identical to Elasticsearch
	os, err := database.NewElasticsearchDatabase(config, logger)
	if err != nil {
		log.Fatalf("Failed to create OpenSearch connection: %v", err)
	}
	defer os.Disconnect()

	ctx := context.Background()

	// Test connection
	if err := os.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping OpenSearch: %v", err)
	}
	fmt.Println("OpenSearch connection successful!")

	// Use all the same methods as Elasticsearch
	tables, err := os.GetTables(ctx, "default")
	if err != nil {
		log.Fatalf("Failed to get tables: %v", err)
	}
	fmt.Printf("Found %d indices\n", len(tables))
}
