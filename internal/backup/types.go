package backup

import "time"

// DatabaseBackup represents a database backup
type DatabaseBackup struct {
	ID             string    `json:"id"`
	BackupType     string    `json:"backup_type"` // 'full', 'incremental'
	Status         string    `json:"status"`      // 'in_progress', 'completed', 'failed'
	FilePath       string    `json:"file_path"`
	FileSize       int64     `json:"file_size,omitempty"`
	TablesIncluded []string  `json:"tables_included,omitempty"`
	StartedAt      time.Time `json:"started_at"`
	CompletedAt    time.Time `json:"completed_at,omitempty"`
	ErrorMessage   string    `json:"error_message,omitempty"`
	CreatedBy      string    `json:"created_by,omitempty"`
}

// BackupOptions configures backup behavior
type BackupOptions struct {
	BackupType    string   `json:"backup_type"` // 'full', 'incremental'
	IncludeTables []string `json:"include_tables,omitempty"`
	ExcludeTables []string `json:"exclude_tables,omitempty"`
	Compress      bool     `json:"compress"`
	EncryptionKey string   `json:"encryption_key,omitempty"`
	MaxBackups    int      `json:"max_backups"` // Number of backups to retain
}

// RestoreOptions configures restore behavior
type RestoreOptions struct {
	BackupID      string   `json:"backup_id"`
	IncludeTables []string `json:"include_tables,omitempty"`
	ExcludeTables []string `json:"exclude_tables,omitempty"`
	DryRun        bool     `json:"dry_run"`
	DecryptionKey string   `json:"decryption_key,omitempty"`
}

// BackupStats provides statistics about backups
type BackupStats struct {
	TotalBackups      int       `json:"total_backups"`
	TotalSize         int64     `json:"total_size_bytes"`
	LatestBackup      time.Time `json:"latest_backup,omitempty"`
	OldestBackup      time.Time `json:"oldest_backup,omitempty"`
	SuccessfulBackups int       `json:"successful_backups"`
	FailedBackups     int       `json:"failed_backups"`
}
