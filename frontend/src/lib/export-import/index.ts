/**
 * Connection Export/Import Module
 *
 * Provides functionality to export database connections to JSON files
 * and import them back. Handles credential sanitization and validation.
 *
 * @module lib/export-import
 *
 * @example
 * // Export connections
 * import { exportConnections, downloadExportFile } from '@/lib/export-import'
 *
 * const connections = useConnectionStore.getState().connections
 * const exportData = await exportConnections(connections, { includePasswords: false })
 * downloadExportFile(exportData)
 *
 * @example
 * // Import connections
 * import { parseExportFile, importConnections, validateExportFile } from '@/lib/export-import'
 *
 * const fileContent = await file.text()
 * const parsed = parseExportFile(fileContent)
 * const validation = validateExportFile(parsed)
 *
 * if (validation.isValid) {
 *   const result = await importConnections(parsed, { conflictResolution: 'skip' })
 *   console.log(`Imported ${result.imported} connections`)
 * }
 */

// Type exports
export type {
  ConnectionExportFile,
  ConflictResolution,
  ExportDialogState,
  ExportedConnection,
  ExportedSSHTunnelConfig,
  ExportedVPCConfig,
  ExportMetadata,
  ExportOptions,
  ImportDialogState,
  ImportFailure,
  ImportOptions,
  ImportResult,
  ValidationResult,
} from './types'

// Constant exports
export {
  CURRENT_SCHEMA_VERSION,
  REQUIRED_CONNECTION_FIELDS,
  VALID_DATABASE_TYPES,
} from './types'

// Validation exports
export {
  fileContainsPasswords,
  findConflictingConnections,
  getValidationSummary,
  validateAllConnections,
  validateConnection,
  validateExportFile,
} from './validation'

// Export service
export {
  buildExportFile,
  exportConnections,
  getExportableConnections,
} from './export-service'

// Import service
export {
  getConflictingIds,
  importConnections,
  parseExportFile,
  previewImport,
  readExportFile,
} from './import-service'
