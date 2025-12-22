//go:build migrate_shared_creds

// Migration Script for Shared Credentials
// ==========================================
//
// PURPOSE:
// This is a DIAGNOSTIC and REPORTING tool to analyze shared database connections
// and identify which ones need migration to the new Organization Envelope Key (OEK) system.
//
// IMPORTANT - WHY THIS CANNOT AUTO-MIGRATE:
// The script CANNOT automatically re-encrypt passwords because:
// 1. User master keys are NOT stored on the server (zero-knowledge architecture)
// 2. Passwords are currently encrypted with individual user master keys
// 3. To migrate to OEK encryption, passwords must be decrypted with the user's master key,
//    then re-encrypted with the organization's OEK
// 4. This requires the user to actively participate in the migration
//
// WHAT THIS SCRIPT DOES:
// - Connects to the Turso database
// - Identifies all shared connections (visibility='shared' AND organization_id IS NOT NULL)
// - Checks which shared connections already have OEK-encrypted credentials
// - Reports which connections need user action to complete migration
// - Generates detailed statistics and migration instructions
//
// HOW TO RUN:
//   # Preview what would be migrated (dry-run mode)
//   TURSO_URL="libsql://..." TURSO_AUTH_TOKEN="..." go run scripts/migrate_shared_credentials.go --dry-run
//
//   # Run with verbose output
//   TURSO_URL="libsql://..." TURSO_AUTH_TOKEN="..." go run scripts/migrate_shared_credentials.go --verbose
//
//   # Run actual analysis (same as dry-run, this is read-only)
//   TURSO_URL="libsql://..." TURSO_AUTH_TOKEN="..." go run scripts/migrate_shared_credentials.go
//
// MIGRATION INSTRUCTIONS FOR USERS:
// For each connection that needs migration, users must:
// 1. Log in to the application
// 2. Navigate to the connection settings
// 3. Click "Re-share with Organization" (or equivalent UI action)
// 4. The app will:
//    - Decrypt the password with the user's master key (client-side)
//    - Encrypt the password with the organization's OEK (client-side)
//    - Submit the OEK-encrypted password to create a shared_credentials entry
// 5. This completes the migration for that connection

package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

// MigrationStats tracks migration progress and analysis
type MigrationStats struct {
	// Overall counts
	TotalConnections    int
	PersonalConnections int
	SharedConnections   int

	// Migration status
	AlreadyMigrated int // Has shared_credentials entry
	NeedsMigration  int // Shared but no shared_credentials entry
	MissingOrgID    int // Shared visibility but no organization_id

	// Details
	ConnectionsNeedingMigration []ConnectionMigrationInfo
	Errors                      []string

	// Performance
	AnalysisStartTime time.Time
	AnalysisEndTime   time.Time
}

// ConnectionMigrationInfo contains details about a connection needing migration
type ConnectionMigrationInfo struct {
	ConnectionID    string
	Name            string
	Type            string
	OrganizationID  string
	CreatedBy       string
	Visibility      string
	CreatedAt       time.Time
	HasPersonalCred bool // Has encrypted_credentials entry
	HasSharedCred   bool // Has shared_credentials entry
}

// OrganizationInfo contains organization details
type OrganizationInfo struct {
	ID      string
	Name    string
	Members int
	HasOEK  bool // Has organization_envelope_keys entries
}

func main() {
	// Parse command-line flags
	dryRun := flag.Bool("dry-run", false, "Show what would be analyzed without making changes (same as normal mode, this is read-only)")
	verbose := flag.Bool("verbose", false, "Show detailed progress and debug information")
	flag.Parse()

	// Get database connection details from environment
	dbURL := os.Getenv("TURSO_URL")
	authToken := os.Getenv("TURSO_AUTH_TOKEN")

	if dbURL == "" {
		log.Fatal("TURSO_URL environment variable is required")
	}

	if authToken == "" {
		log.Fatal("TURSO_AUTH_TOKEN environment variable is required")
	}

	// Build connection string
	connStr := dbURL
	if authToken != "" {
		connStr = fmt.Sprintf("%s?authToken=%s", dbURL, authToken)
	}

	// Connect to database
	if *verbose {
		log.Printf("Connecting to database: %s", strings.Split(dbURL, "?")[0])
	}

	db, err := sql.Open("libsql", connStr)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Test connection
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	if *verbose {
		log.Println("Database connection established")
	}

	// Run migration analysis
	stats, err := analyzeMigrationNeeds(ctx, db, *verbose)
	if err != nil {
		log.Fatalf("Migration analysis failed: %v", err)
	}

	// Print results
	printMigrationReport(stats, *dryRun, *verbose)

	// Exit with appropriate code
	if len(stats.Errors) > 0 {
		os.Exit(1)
	}
}

// analyzeMigrationNeeds analyzes the database to determine migration requirements
func analyzeMigrationNeeds(ctx context.Context, db *sql.DB, verbose bool) (*MigrationStats, error) {
	stats := &MigrationStats{
		AnalysisStartTime:           time.Now(),
		ConnectionsNeedingMigration: make([]ConnectionMigrationInfo, 0),
		Errors:                      make([]string, 0),
	}

	// First, check if migration 009 has been applied
	if verbose {
		log.Println("Checking if shared credentials tables exist...")
	}

	hasSharedCredsTables, err := checkTablesExist(ctx, db)
	if err != nil {
		stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to check for shared_credentials tables: %v", err))
		return stats, err
	}

	if !hasSharedCredsTables {
		stats.Errors = append(stats.Errors, "Migration 009 has not been applied - shared_credentials tables do not exist")
		return stats, fmt.Errorf("migration 009 (shared credentials) has not been applied yet")
	}

	// Get all connections
	if verbose {
		log.Println("Querying all connections...")
	}

	query := `
		SELECT
			id, name, type, user_id, organization_id, visibility, created_by, created_at
		FROM connection_templates
		ORDER BY visibility, organization_id, name
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query connections: %w", err)
	}
	defer rows.Close()

	// Process each connection
	for rows.Next() {
		var (
			id, name, connType, userID, createdBy string
			orgID, visibility                     sql.NullString
			createdAtUnix                         int64
		)

		if err := rows.Scan(&id, &name, &connType, &userID, &orgID, &visibility, &createdBy, &createdAtUnix); err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to scan connection row: %v", err))
			continue
		}

		stats.TotalConnections++

		// Determine connection type and migration status
		vis := "personal"
		if visibility.Valid {
			vis = visibility.String
		}

		if vis == "personal" || !orgID.Valid {
			stats.PersonalConnections++
			continue
		}

		// This is a shared connection
		stats.SharedConnections++

		info := ConnectionMigrationInfo{
			ConnectionID:   id,
			Name:           name,
			Type:           connType,
			OrganizationID: orgID.String,
			CreatedBy:      createdBy,
			Visibility:     vis,
			CreatedAt:      time.Unix(createdAtUnix, 0),
		}

		// Check if connection has personal encrypted credential
		var personalCredCount int
		err := db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM encrypted_credentials WHERE connection_id = ?",
			id,
		).Scan(&personalCredCount)
		if err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to check personal credentials for %s: %v", id, err))
		}
		info.HasPersonalCred = personalCredCount > 0

		// Check if connection has shared credential (OEK-encrypted)
		var sharedCredCount int
		err = db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM shared_credentials WHERE connection_id = ?",
			id,
		).Scan(&sharedCredCount)
		if err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to check shared credentials for %s: %v", id, err))
		}
		info.HasSharedCred = sharedCredCount > 0

		// Determine migration status
		if info.HasSharedCred {
			stats.AlreadyMigrated++
			if verbose {
				log.Printf("  ✓ %s: Already migrated (has shared_credentials)", name)
			}
		} else {
			stats.NeedsMigration++
			stats.ConnectionsNeedingMigration = append(stats.ConnectionsNeedingMigration, info)
			if verbose {
				log.Printf("  ✗ %s: Needs migration", name)
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating connection rows: %w", err)
	}

	stats.AnalysisEndTime = time.Now()
	return stats, nil
}

// checkTablesExist verifies that the shared credentials tables exist
func checkTablesExist(ctx context.Context, db *sql.DB) (bool, error) {
	// Check for shared_credentials table
	var tableName string
	err := db.QueryRowContext(ctx,
		"SELECT name FROM sqlite_master WHERE type='table' AND name='shared_credentials'",
	).Scan(&tableName)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// Check for organization_envelope_keys table
	err = db.QueryRowContext(ctx,
		"SELECT name FROM sqlite_master WHERE type='table' AND name='organization_envelope_keys'",
	).Scan(&tableName)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

// printMigrationReport prints a detailed migration report
func printMigrationReport(stats *MigrationStats, dryRun bool, verbose bool) {
	duration := stats.AnalysisEndTime.Sub(stats.AnalysisStartTime)

	fmt.Println()
	fmt.Println("================================================================================")
	fmt.Println("         SHARED CREDENTIALS MIGRATION ANALYSIS REPORT")
	fmt.Println("================================================================================")
	fmt.Println()

	if dryRun {
		fmt.Println("MODE: DRY RUN (Read-only analysis)")
		fmt.Println()
	}

	// Summary statistics
	fmt.Println("SUMMARY:")
	fmt.Printf("  Total Connections:           %d\n", stats.TotalConnections)
	fmt.Printf("  - Personal Connections:      %d\n", stats.PersonalConnections)
	fmt.Printf("  - Shared Connections:        %d\n", stats.SharedConnections)
	fmt.Println()
	fmt.Printf("  Already Migrated:            %d (%.1f%%)\n",
		stats.AlreadyMigrated,
		percentage(stats.AlreadyMigrated, stats.SharedConnections))
	fmt.Printf("  Needs Migration:             %d (%.1f%%)\n",
		stats.NeedsMigration,
		percentage(stats.NeedsMigration, stats.SharedConnections))
	fmt.Println()
	fmt.Printf("  Analysis Duration:           %v\n", duration.Round(time.Millisecond))
	fmt.Println()

	// Connections needing migration
	if stats.NeedsMigration > 0 {
		fmt.Println("CONNECTIONS REQUIRING MIGRATION:")
		fmt.Println("--------------------------------------------------------------------------------")

		// Group by organization
		orgMap := make(map[string][]ConnectionMigrationInfo)
		for _, conn := range stats.ConnectionsNeedingMigration {
			orgMap[conn.OrganizationID] = append(orgMap[conn.OrganizationID], conn)
		}

		for orgID, conns := range orgMap {
			fmt.Printf("\nOrganization: %s (%d connections)\n", orgID, len(conns))
			for _, conn := range conns {
				fmt.Printf("  • %s\n", conn.Name)
				fmt.Printf("    ID:           %s\n", conn.ConnectionID)
				fmt.Printf("    Type:         %s\n", conn.Type)
				fmt.Printf("    Created By:   %s\n", conn.CreatedBy)
				fmt.Printf("    Created At:   %s\n", conn.CreatedAt.Format(time.RFC3339))
				fmt.Printf("    Personal Cred: %v\n", conn.HasPersonalCred)
				fmt.Printf("    Shared Cred:   %v\n", conn.HasSharedCred)
				fmt.Println()
			}
		}
	}

	// Migration instructions
	if stats.NeedsMigration > 0 {
		fmt.Println("================================================================================")
		fmt.Println("MIGRATION INSTRUCTIONS:")
		fmt.Println("================================================================================")
		fmt.Println()
		fmt.Println("These shared connections are using the OLD encryption model (personal master")
		fmt.Println("keys) and need to be migrated to the NEW model (organization envelope keys).")
		fmt.Println()
		fmt.Println("WHY AUTOMATIC MIGRATION IS NOT POSSIBLE:")
		fmt.Println("  - User master keys are NOT stored on the server (zero-knowledge architecture)")
		fmt.Println("  - Passwords must be decrypted client-side with the user's master key")
		fmt.Println("  - Then re-encrypted client-side with the organization envelope key (OEK)")
		fmt.Println()
		fmt.Println("WHAT USERS NEED TO DO:")
		fmt.Println("  1. Log in to the application")
		fmt.Println("  2. Navigate to Organization Connections")
		fmt.Println("  3. For each connection listed above:")
		fmt.Println("     a. Click on the connection")
		fmt.Println("     b. Click 'Re-share with Organization' or 'Migrate to OEK'")
		fmt.Println("     c. The app will handle client-side re-encryption automatically")
		fmt.Println()
		fmt.Println("WHAT HAPPENS DURING MIGRATION:")
		fmt.Println("  1. User's browser decrypts the password using their master key")
		fmt.Println("  2. User's browser encrypts the password using the organization's OEK")
		fmt.Println("  3. New encrypted password is stored in the shared_credentials table")
		fmt.Println("  4. Connection's encryption_type is updated to 'shared'")
		fmt.Println("  5. All organization members can now access the connection")
		fmt.Println()
	} else {
		fmt.Println("================================================================================")
		fmt.Println("✓ ALL SHARED CONNECTIONS HAVE BEEN MIGRATED!")
		fmt.Println("================================================================================")
		fmt.Println()
	}

	// Errors
	if len(stats.Errors) > 0 {
		fmt.Println("================================================================================")
		fmt.Println("ERRORS ENCOUNTERED:")
		fmt.Println("================================================================================")
		for i, err := range stats.Errors {
			fmt.Printf("%d. %s\n", i+1, err)
		}
		fmt.Println()
	}

	fmt.Println("================================================================================")
	fmt.Println("End of Report")
	fmt.Println("================================================================================")
	fmt.Println()
}

// percentage calculates percentage, handling division by zero
func percentage(part, total int) float64 {
	if total == 0 {
		return 0.0
	}
	return float64(part) / float64(total) * 100.0
}
