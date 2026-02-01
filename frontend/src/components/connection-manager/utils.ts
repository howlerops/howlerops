import { SSHAuthMethod } from "@/generated/database"

import type { ConnectionFormData, ConnectionPayload, DatabaseConnection, DatabaseTypeString, SSHTunnelConfig, VPCConfig } from "./types"

/**
 * Creates default form data for a new connection
 */
export const createDefaultFormData = (): ConnectionFormData => ({
  name: '',
  type: 'postgresql',
  host: 'localhost',
  port: '5432',
  database: '',
  username: '',
  password: '',
  sslMode: 'prefer',
  environments: [],

  // SSH Tunnel defaults
  useTunnel: false,
  sshHost: '',
  sshPort: '22',
  sshUser: '',
  sshAuthMethod: SSHAuthMethod.SSH_AUTH_METHOD_PASSWORD,
  sshPassword: '',
  sshPrivateKey: '',
  sshPrivateKeyPath: '',
  sshPrivateKeyPassphrase: '',
  sshKnownHostsPath: '',
  sshStrictHostKeyChecking: true,
  sshTimeoutSeconds: '30',
  sshKeepAliveIntervalSeconds: '0',

  // VPC defaults
  useVpc: false,
  vpcId: '',
  subnetId: '',
  securityGroupIds: '',
  privateLinkService: '',
  endpointServiceName: '',

  // Database-specific defaults
  mongoConnectionString: '',
  mongoAuthDatabase: '',
  elasticScheme: 'https',
  elasticApiKey: '',
  clickhouseNativeProtocol: false,
})

/**
 * Returns the default port for a database type
 */
export const getDefaultPort = (type: DatabaseTypeString): string => {
  const portMap: Record<DatabaseTypeString, string> = {
    postgresql: '5432',
    mysql: '3306',
    mariadb: '3306',
    tidb: '4000',
    clickhouse: '9000',
    mongodb: '27017',
    elasticsearch: '9200',
    opensearch: '9200',
    sqlite: '',
    mssql: '1433',
  }
  return portMap[type] ?? '5432'
}

/**
 * Checks if the database type requires host and port configuration
 */
export const requiresHostPort = (type: DatabaseTypeString): boolean => {
  return type !== 'sqlite'
}

/**
 * Checks if the database type supports SSL configuration
 */
export const supportsSSL = (type: DatabaseTypeString): boolean => {
  return ['postgresql', 'mysql', 'mariadb', 'tidb', 'clickhouse'].includes(type)
}

/**
 * Populates form data from an existing connection
 */
export const populateFormFromConnection = (connection: DatabaseConnection): ConnectionFormData => {
  return {
    name: connection.name,
    type: connection.type,
    host: connection.host || 'localhost',
    port: connection.port ? String(connection.port) : getDefaultPort(connection.type),
    database: connection.database || '',
    username: connection.username || '',
    password: connection.password || '',
    sslMode: connection.sslMode || 'prefer',
    environments: connection.environments || [],

    // SSH Tunnel
    useTunnel: connection.useTunnel || false,
    sshHost: connection.sshTunnel?.host || '',
    sshPort: connection.sshTunnel?.port ? String(connection.sshTunnel.port) : '22',
    sshUser: connection.sshTunnel?.user || '',
    sshAuthMethod: connection.sshTunnel?.authMethod || SSHAuthMethod.SSH_AUTH_METHOD_PASSWORD,
    sshPassword: connection.sshTunnel?.password || '',
    sshPrivateKey: connection.sshTunnel?.privateKey || '',
    sshPrivateKeyPath: connection.sshTunnel?.privateKeyPath || '',
    sshPrivateKeyPassphrase: '',
    sshKnownHostsPath: connection.sshTunnel?.knownHostsPath || '',
    sshStrictHostKeyChecking: connection.sshTunnel?.strictHostKeyChecking ?? true,
    sshTimeoutSeconds: connection.sshTunnel?.timeoutSeconds ? String(connection.sshTunnel.timeoutSeconds) : '30',
    sshKeepAliveIntervalSeconds: connection.sshTunnel?.keepAliveIntervalSeconds ? String(connection.sshTunnel.keepAliveIntervalSeconds) : '0',

    // VPC
    useVpc: connection.useVpc || false,
    vpcId: connection.vpcConfig?.vpcId || '',
    subnetId: connection.vpcConfig?.subnetId || '',
    securityGroupIds: connection.vpcConfig?.securityGroupIds?.join(', ') || '',
    privateLinkService: connection.vpcConfig?.privateLinkService || '',
    endpointServiceName: connection.vpcConfig?.endpointServiceName || '',

    // Database-specific parameters
    mongoConnectionString: connection.parameters?.connectionString || '',
    mongoAuthDatabase: connection.parameters?.authDatabase || '',
    elasticScheme: connection.parameters?.scheme || 'https',
    elasticApiKey: connection.parameters?.apiKey || '',
    clickhouseNativeProtocol: connection.parameters?.nativeProtocol === 'true',
  }
}

/**
 * Builds connection payload from form data for API submission
 */
export const buildConnectionPayload = (formData: ConnectionFormData): ConnectionPayload => {
  const port = formData.port ? parseInt(formData.port, 10) : 0
  const parameters: Record<string, string> = {}

  // MongoDB-specific parameters
  if (formData.type === 'mongodb') {
    if (formData.mongoConnectionString) {
      parameters.connectionString = formData.mongoConnectionString
    }
    if (formData.mongoAuthDatabase) {
      parameters.authDatabase = formData.mongoAuthDatabase
    }
  }

  // Elasticsearch/OpenSearch parameters
  if (formData.type === 'elasticsearch' || formData.type === 'opensearch') {
    parameters.scheme = formData.elasticScheme
    if (formData.elasticApiKey) {
      parameters.apiKey = formData.elasticApiKey
    }
  }

  // ClickHouse parameters
  if (formData.type === 'clickhouse') {
    parameters.nativeProtocol = formData.clickhouseNativeProtocol.toString()
  }

  // Build SSH tunnel config if enabled
  let sshTunnel: SSHTunnelConfig | undefined
  if (formData.useTunnel) {
    sshTunnel = {
      host: formData.sshHost,
      port: parseInt(formData.sshPort, 10) || 22,
      user: formData.sshUser,
      authMethod: formData.sshAuthMethod,
      password: formData.sshAuthMethod === SSHAuthMethod.SSH_AUTH_METHOD_PASSWORD ? formData.sshPassword : undefined,
      privateKey: formData.sshAuthMethod === SSHAuthMethod.SSH_AUTH_METHOD_PRIVATE_KEY ? formData.sshPrivateKey : undefined,
      privateKeyPath: formData.sshPrivateKeyPath || undefined,
      knownHostsPath: formData.sshKnownHostsPath || undefined,
      strictHostKeyChecking: formData.sshStrictHostKeyChecking,
      timeoutSeconds: parseInt(formData.sshTimeoutSeconds, 10) || 30,
      keepAliveIntervalSeconds: parseInt(formData.sshKeepAliveIntervalSeconds, 10) || 0,
    }
  }

  // Build VPC config if enabled
  let vpcConfig: VPCConfig | undefined
  if (formData.useVpc) {
    vpcConfig = {
      vpcId: formData.vpcId,
      subnetId: formData.subnetId,
      securityGroupIds: formData.securityGroupIds.split(',').map(id => id.trim()).filter(Boolean),
      privateLinkService: formData.privateLinkService || undefined,
      endpointServiceName: formData.endpointServiceName || undefined,
    }
  }

  return {
    name: formData.name,
    type: formData.type,
    host: formData.type === 'sqlite' ? '' : formData.host,
    port: formData.type === 'sqlite' ? 0 : port,
    database: formData.database,
    username: formData.type === 'sqlite' ? '' : formData.username,
    password: formData.type === 'sqlite' ? '' : formData.password,
    sslMode: formData.sslMode,
    environments: formData.environments,
    useTunnel: formData.useTunnel,
    sshTunnel,
    useVpc: formData.useVpc,
    vpcConfig,
    parameters,
  }
}

/**
 * Gets the database label for display
 */
export const getDatabaseLabel = (type: DatabaseTypeString): string => {
  if (type === 'mongodb') return 'Database (optional)'
  if (type === 'elasticsearch' || type === 'opensearch') return 'Index Pattern (optional)'
  return 'Database'
}

/**
 * Checks if database field is required for the given type
 */
export const isDatabaseRequired = (type: DatabaseTypeString): boolean => {
  return type !== 'mongodb' && type !== 'elasticsearch' && type !== 'opensearch'
}

/**
 * Checks if username is required for the given type
 */
export const isUsernameRequired = (type: DatabaseTypeString): boolean => {
  return type !== 'mongodb'
}
