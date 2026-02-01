import { SSHAuthMethod } from "@/generated/database"
import { DatabaseConnection,DatabaseTypeString, SSHTunnelConfig, VPCConfig } from "@/store/connection-store"

/**
 * Form data structure for creating/editing database connections
 */
export interface ConnectionFormData {
  name: string
  type: DatabaseTypeString
  host: string
  port: string
  database: string
  username: string
  password: string
  sslMode: string
  environments: string[]

  // SSH Tunnel
  useTunnel: boolean
  sshHost: string
  sshPort: string
  sshUser: string
  sshAuthMethod: SSHAuthMethod
  sshPassword: string
  sshPrivateKey: string
  sshPrivateKeyPath: string
  sshPrivateKeyPassphrase: string
  sshKnownHostsPath: string
  sshStrictHostKeyChecking: boolean
  sshTimeoutSeconds: string
  sshKeepAliveIntervalSeconds: string

  // VPC
  useVpc: boolean
  vpcId: string
  subnetId: string
  securityGroupIds: string
  privateLinkService: string
  endpointServiceName: string

  // Database-specific parameters
  mongoConnectionString: string
  mongoAuthDatabase: string
  elasticScheme: string
  elasticApiKey: string
  clickhouseNativeProtocol: boolean
}

/**
 * Database type option for dropdown selection
 */
export interface DatabaseTypeOption {
  value: DatabaseTypeString
  label: string
}

/**
 * Grouped connections by environment
 */
export interface ConnectionGroup {
  key: string
  label: string
  connections: DatabaseConnection[]
}

/**
 * Connection payload for API calls
 */
export interface ConnectionPayload {
  name: string
  type: DatabaseTypeString
  host: string
  port: number
  database: string
  username: string
  password: string
  sslMode: string
  environments: string[]
  useTunnel: boolean
  sshTunnel?: SSHTunnelConfig
  useVpc: boolean
  vpcConfig?: VPCConfig
  parameters: Record<string, string>
}

/**
 * Available database types with their display labels
 */
export const DATABASE_TYPE_OPTIONS: readonly DatabaseTypeOption[] = [
  { value: 'postgresql', label: 'PostgreSQL' },
  { value: 'mysql', label: 'MySQL' },
  { value: 'mariadb', label: 'MariaDB' },
  { value: 'sqlite', label: 'SQLite' },
  { value: 'mssql', label: 'SQL Server' },
  { value: 'tidb', label: 'TiDB' },
  { value: 'clickhouse', label: 'ClickHouse' },
  { value: 'mongodb', label: 'MongoDB' },
  { value: 'elasticsearch', label: 'Elasticsearch' },
  { value: 'opensearch', label: 'OpenSearch' },
] as const

/**
 * Special filter value for "all environments"
 */
export const ALL_ENVIRONMENTS_FILTER = '__all__'

/**
 * Label for connections without assigned environment
 */
export const UNASSIGNED_ENVIRONMENT_LABEL = 'No Environment'

// Re-export commonly used types from the store
export type { DatabaseConnection, DatabaseTypeString, SSHTunnelConfig, VPCConfig }
export { SSHAuthMethod }
