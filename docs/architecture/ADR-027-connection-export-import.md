# ADR-027: Connection Export/Import Feature Architecture

**Status:** Proposed
**Date:** 2026-02-05
**Author:** System Architecture Designer

## Context

Users need the ability to export their database connections to a portable JSON file and import connections from such files. This enables:
- Backup and restore of connection configurations
- Sharing connection templates between team members (without credentials)
- Migration between machines or HowlerOps instances
- Version control of connection configurations

## Decision

### 1. Export File Format

The export file will use a well-structured JSON format with metadata and connection data:

```typescript
interface ConnectionExportFile {
  // Metadata section
  metadata: {
    version: "1.0.0"           // Schema version for future compatibility
    exportedAt: string         // ISO 8601 timestamp
    appVersion: string         // HowlerOps version (e.g., "3.0.0")
    exportedBy?: string        // Optional: anonymized user identifier
    connectionCount: number    // Quick reference for UI
    includesPasswords: boolean // Security indicator flag
  }

  // Connection data
  connections: ExportedConnection[]
}

interface ExportedConnection {
  // Core identification
  id: string                   // Original UUID (for duplicate detection)
  name: string                 // Human-readable name

  // Connection type
  type: DatabaseTypeString     // postgresql, mysql, sqlite, etc.

  // Core connection details
  host?: string
  port?: number
  database: string
  username?: string
  sslMode?: string

  // Environment tags
  environments?: string[]

  // SSH Tunnel (credentials excluded by default)
  useTunnel?: boolean
  sshTunnel?: ExportedSSHTunnelConfig

  // VPC Config
  useVpc?: boolean
  vpcConfig?: ExportedVPCConfig

  // Database-specific parameters (sanitized)
  parameters?: Record<string, string>

  // Optional sensitive data (only if explicitly requested)
  password?: string            // Only included with user confirmation
}

interface ExportedSSHTunnelConfig {
  host: string
  port: number
  user: string
  authMethod: SSHAuthMethod
  privateKeyPath?: string      // Path reference only (not the key itself)
  knownHostsPath?: string
  strictHostKeyChecking: boolean
  timeoutSeconds: number
  keepAliveIntervalSeconds: number
  // NOTE: password and privateKey content are NEVER exported
}

interface ExportedVPCConfig {
  vpcId: string
  subnetId: string
  securityGroupIds: string[]
  privateLinkService?: string
  endpointServiceName?: string
  // NOTE: customConfig is sanitized before export
}
```

### 2. Password Handling Strategy

**Default behavior: Exclude all passwords**

Passwords and sensitive credentials are excluded by default for security:

| Field | Default Behavior | With "Include Passwords" |
|-------|-----------------|-------------------------|
| `password` | Excluded | Included with warning |
| `sshTunnel.password` | Excluded | Excluded (never exported) |
| `sshTunnel.privateKey` | Excluded | Excluded (never exported) |
| `parameters.*` | Sanitized | Sanitized |
| `vpcConfig.customConfig` | Sanitized | Sanitized |

**Security rationale:**
- SSH private keys should NEVER be exported (use paths instead)
- SSH passwords should NEVER be exported (security risk)
- Database passwords can optionally be included with explicit user confirmation
- Parameters are always run through the existing credential sanitizer

**Export options:**
```typescript
interface ExportOptions {
  includePasswords: boolean  // Default: false
  selectedConnectionIds?: string[]  // Export subset, or all if undefined
}
```

### 3. Import Conflict Resolution

When importing connections, duplicates are identified by matching the `id` field:

```typescript
interface ImportOptions {
  conflictResolution: 'skip' | 'overwrite' | 'keep-both'
  // skip: Ignore connections with matching IDs
  // overwrite: Replace existing connections with imported ones
  // keep-both: Import with new UUID (creates duplicates)
}

interface ImportResult {
  imported: number           // Successfully imported
  skipped: number            // Skipped due to conflicts
  overwritten: number        // Replaced existing connections
  failed: ImportFailure[]    // Failed with reasons
}

interface ImportFailure {
  connectionName: string
  originalId: string
  reason: string             // Validation error message
}
```

### 4. Validation on Import

All imported connections must pass validation:

```typescript
interface ValidationResult {
  isValid: boolean
  errors: ValidationError[]
  warnings: ValidationWarning[]
}

// Required field validation
const REQUIRED_FIELDS = ['id', 'name', 'type', 'database']

// Valid database types
const VALID_TYPES: DatabaseTypeString[] = [
  'postgresql', 'mysql', 'sqlite', 'mssql', 'mariadb',
  'elasticsearch', 'opensearch', 'clickhouse', 'mongodb', 'tidb'
]

// Validation rules:
// 1. Required fields must be present and non-empty
// 2. Type must be in VALID_TYPES array
// 3. Port must be valid number (1-65535) if present
// 4. Environments must be string array if present
// 5. SSH config must have required fields if useTunnel is true
// 6. ID must be valid UUID format
```

### 5. Component Architecture

```
+------------------------------------------------------------------+
|                    ConnectionExportImport                         |
|  (New component - orchestrates export/import UI)                  |
+------------------------------------------------------------------+
         |                    |                     |
         v                    v                     v
+----------------+  +------------------+  +------------------+
| ExportDialog   |  | ImportDialog     |  | ExportService    |
| - Select conns |  | - File upload    |  | - Sanitization   |
| - Password opt |  | - Preview list   |  | - File generation|
| - Download btn |  | - Conflict opts  |  | - Validation     |
+----------------+  +------------------+  +------------------+
                            |
                            v
                    +------------------+
                    | ImportService    |
                    | - Parse & validate|
                    | - Conflict detect |
                    | - Apply imports   |
                    +------------------+
```

### 6. UI Placement

The export/import feature will be available in **two locations**:

#### A. Settings Page (Primary)
Add a new "Data Management" card to `/frontend/src/pages/settings.tsx`:

```tsx
<Card>
  <CardHeader>
    <CardTitle className="flex items-center gap-2">
      <Database className="h-5 w-5" />
      Data Management
    </CardTitle>
    <CardDescription>
      Export and import your database connections
    </CardDescription>
  </CardHeader>
  <CardContent>
    <div className="space-y-4">
      {/* Export section */}
      <div className="flex items-center justify-between">
        <div>
          <p className="font-medium">Export Connections</p>
          <p className="text-sm text-muted-foreground">
            Download all connections as a JSON file
          </p>
        </div>
        <Button variant="outline" onClick={openExportDialog}>
          <Download className="h-4 w-4 mr-2" />
          Export
        </Button>
      </div>

      {/* Import section */}
      <div className="flex items-center justify-between">
        <div>
          <p className="font-medium">Import Connections</p>
          <p className="text-sm text-muted-foreground">
            Load connections from a JSON file
          </p>
        </div>
        <Button variant="outline" onClick={openImportDialog}>
          <Upload className="h-4 w-4 mr-2" />
          Import
        </Button>
      </div>
    </div>
  </CardContent>
</Card>
```

#### B. Connection Manager Dropdown (Secondary)
Add to the connection manager header as a dropdown option:

```tsx
<DropdownMenu>
  <DropdownMenuTrigger asChild>
    <Button variant="ghost" size="sm">
      <MoreHorizontal className="h-4 w-4" />
    </Button>
  </DropdownMenuTrigger>
  <DropdownMenuContent align="end">
    <DropdownMenuItem onClick={openExportDialog}>
      <Download className="h-4 w-4 mr-2" />
      Export Connections
    </DropdownMenuItem>
    <DropdownMenuItem onClick={openImportDialog}>
      <Upload className="h-4 w-4 mr-2" />
      Import Connections
    </DropdownMenuItem>
  </DropdownMenuContent>
</DropdownMenu>
```

### 7. File Structure

New files to create:

```
frontend/src/
  lib/
    export-import/
      types.ts                 # Type definitions
      export-service.ts        # Export logic
      import-service.ts        # Import logic
      validation.ts            # Validation functions
      index.ts                 # Public exports
  components/
    connection-manager/
      components/
        export-dialog.tsx      # Export UI modal
        import-dialog.tsx      # Import UI modal
```

### 8. Service Implementation Details

#### ExportService

```typescript
// /frontend/src/lib/export-import/export-service.ts

import { getSecureStorage } from '@/lib/secure-storage'
import { sanitizeConnection } from '@/lib/sanitization/connection-sanitizer'
import { DatabaseConnection } from '@/store/connection-store'
import { ConnectionExportFile, ExportOptions, ExportedConnection } from './types'

export async function exportConnections(
  connections: DatabaseConnection[],
  options: ExportOptions = { includePasswords: false }
): Promise<ConnectionExportFile> {
  const exportedConnections: ExportedConnection[] = []

  for (const conn of connections) {
    // Apply sanitization
    const sanitized = sanitizeConnection(conn)

    // Build exported connection
    const exported: ExportedConnection = {
      id: conn.id,
      name: conn.name,
      type: conn.type,
      host: conn.host,
      port: conn.port,
      database: conn.database,
      username: conn.username,
      sslMode: conn.sslMode,
      environments: conn.environments,
      useTunnel: conn.useTunnel,
      useVpc: conn.useVpc,
    }

    // Handle SSH tunnel (always sanitized)
    if (conn.sshTunnel) {
      exported.sshTunnel = {
        host: conn.sshTunnel.host,
        port: conn.sshTunnel.port,
        user: conn.sshTunnel.user,
        authMethod: conn.sshTunnel.authMethod,
        privateKeyPath: conn.sshTunnel.privateKeyPath,
        knownHostsPath: conn.sshTunnel.knownHostsPath,
        strictHostKeyChecking: conn.sshTunnel.strictHostKeyChecking,
        timeoutSeconds: conn.sshTunnel.timeoutSeconds,
        keepAliveIntervalSeconds: conn.sshTunnel.keepAliveIntervalSeconds,
      }
    }

    // Handle VPC config (always sanitized)
    if (conn.vpcConfig) {
      exported.vpcConfig = {
        vpcId: conn.vpcConfig.vpcId,
        subnetId: conn.vpcConfig.subnetId,
        securityGroupIds: conn.vpcConfig.securityGroupIds,
        privateLinkService: conn.vpcConfig.privateLinkService,
        endpointServiceName: conn.vpcConfig.endpointServiceName,
      }
    }

    // Handle parameters (sanitized)
    if (conn.parameters && sanitized.sanitizedConnection.parameters) {
      exported.parameters = sanitized.sanitizedConnection.parameters
    }

    // Handle password (only if explicitly requested)
    if (options.includePasswords) {
      const secureStorage = getSecureStorage()
      const credentials = await secureStorage.getCredentials(conn.id)
      if (credentials?.password) {
        exported.password = credentials.password
      }
    }

    exportedConnections.push(exported)
  }

  return {
    metadata: {
      version: '1.0.0',
      exportedAt: new Date().toISOString(),
      appVersion: import.meta.env.VITE_APP_VERSION || 'unknown',
      connectionCount: exportedConnections.length,
      includesPasswords: options.includePasswords,
    },
    connections: exportedConnections,
  }
}

export function downloadExportFile(exportData: ConnectionExportFile): void {
  const blob = new Blob(
    [JSON.stringify(exportData, null, 2)],
    { type: 'application/json' }
  )

  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `howlerops-connections-${new Date().toISOString().split('T')[0]}.json`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}
```

#### ImportService

```typescript
// /frontend/src/lib/export-import/import-service.ts

import { useConnectionStore } from '@/store/connection-store'
import { getSecureStorage } from '@/lib/secure-storage'
import {
  ConnectionExportFile,
  ExportedConnection,
  ImportOptions,
  ImportResult,
  ValidationResult
} from './types'
import { validateConnection, validateExportFile } from './validation'

export function parseExportFile(fileContent: string): ConnectionExportFile {
  try {
    const parsed = JSON.parse(fileContent)
    const validation = validateExportFile(parsed)

    if (!validation.isValid) {
      throw new Error(`Invalid file format: ${validation.errors.join(', ')}`)
    }

    return parsed as ConnectionExportFile
  } catch (error) {
    if (error instanceof SyntaxError) {
      throw new Error('Invalid JSON file')
    }
    throw error
  }
}

export async function importConnections(
  exportFile: ConnectionExportFile,
  options: ImportOptions
): Promise<ImportResult> {
  const store = useConnectionStore.getState()
  const secureStorage = getSecureStorage()
  const existingIds = new Set(store.connections.map(c => c.id))

  const result: ImportResult = {
    imported: 0,
    skipped: 0,
    overwritten: 0,
    failed: [],
  }

  for (const imported of exportFile.connections) {
    // Validate connection
    const validation = validateConnection(imported)
    if (!validation.isValid) {
      result.failed.push({
        connectionName: imported.name,
        originalId: imported.id,
        reason: validation.errors.join('; '),
      })
      continue
    }

    const isDuplicate = existingIds.has(imported.id)

    if (isDuplicate) {
      switch (options.conflictResolution) {
        case 'skip':
          result.skipped++
          continue

        case 'overwrite':
          // Remove existing and add new
          await store.removeConnection(imported.id)
          await addConnection(imported, store, secureStorage)
          result.overwritten++
          break

        case 'keep-both':
          // Generate new ID and import
          const newConnection = { ...imported, id: crypto.randomUUID() }
          await addConnection(newConnection, store, secureStorage)
          result.imported++
          break
      }
    } else {
      await addConnection(imported, store, secureStorage)
      result.imported++
    }
  }

  return result
}

async function addConnection(
  imported: ExportedConnection,
  store: ReturnType<typeof useConnectionStore.getState>,
  secureStorage: ReturnType<typeof getSecureStorage>
): Promise<void> {
  // Store password in secure storage if present
  if (imported.password) {
    await secureStorage.setCredentials(imported.id, {
      password: imported.password,
    })
  }

  // Add connection to store (without password in state)
  await store.addConnection({
    ...imported,
    password: undefined, // Never store in Zustand state
  })
}
```

### 9. User Experience Flow

#### Export Flow:
1. User clicks "Export Connections" button
2. Export dialog opens showing:
   - List of connections with checkboxes (all selected by default)
   - Option to "Include passwords" with warning icon
   - Preview of export metadata
3. If "Include passwords" is checked:
   - Show prominent warning dialog explaining security implications
   - Require explicit confirmation
4. User clicks "Export"
5. Browser downloads JSON file with timestamp in filename

#### Import Flow:
1. User clicks "Import Connections" button
2. Import dialog opens with file drop zone
3. User selects/drops JSON file
4. System validates and parses file
5. Preview shows:
   - Number of connections found
   - List of connections with names and types
   - Warning if file includes passwords
   - Conflict detection (connections with matching IDs)
6. User selects conflict resolution strategy if conflicts exist
7. User clicks "Import"
8. Results dialog shows:
   - Number imported/skipped/overwritten/failed
   - Details for any failures
9. Connection list refreshes automatically

### 10. Error Handling

| Error Type | User Message | Action |
|------------|--------------|--------|
| Invalid JSON | "The file is not valid JSON. Please select a valid export file." | Show error, keep dialog open |
| Wrong schema | "This file is not a HowlerOps connection export. Missing required fields." | Show error, keep dialog open |
| Version mismatch | "This export was created with a newer version of HowlerOps." | Show warning, allow import attempt |
| Invalid connection | "Connection '[name]' failed validation: [reason]" | Skip connection, continue with others |
| Storage error | "Failed to save connection '[name]'. Please try again." | Show error, report in results |

### 11. Security Considerations

1. **Password export requires explicit opt-in** - Default export excludes all credentials
2. **SSH credentials are NEVER exported** - Private keys and SSH passwords are explicitly excluded
3. **Parameters are sanitized** - Uses existing `connection-sanitizer.ts` infrastructure
4. **File is human-readable** - Users can inspect before sharing
5. **Import validation** - All imported data is validated before use
6. **No remote transmission** - Export/import is purely local file operations

### 12. Future Extensibility

The versioned schema allows for future enhancements:
- Schema version migrations for backward compatibility
- Additional metadata fields (tags, descriptions)
- Support for connection groups/folders
- Encrypted export option (password-protected)
- Cloud sync integration (if implemented)

## Consequences

### Positive
- Users can easily backup and restore connections
- Team members can share connection templates securely
- Human-readable format allows manual inspection
- Leverages existing sanitization infrastructure
- Clear separation of concerns (export/import services)

### Negative
- Additional code to maintain
- Users may accidentally share files with passwords included
- Import conflicts require user decisions

### Mitigations
- Strong default security (passwords excluded)
- Clear warnings for password inclusion
- Preview before import to catch issues early
- Validation prevents malformed data

## Implementation Priority

1. **Phase 1:** Core export functionality (no passwords)
2. **Phase 2:** Core import functionality with validation
3. **Phase 3:** Password export option with warnings
4. **Phase 4:** UI refinements (connection selection, preview)

## Related Documents

- `connection-sanitizer.ts` - Existing credential sanitization
- `connection-store.ts` - Connection state management
- `secure-storage.ts` - Password storage utilities
