import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import { Label } from "@/components/ui/label"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import { SecretInput } from "@/components/secret-input"
import { PemKeyUpload } from "@/components/pem-key-upload"
import { useConnectionStore, DatabaseTypeString, SSHTunnelConfig, VPCConfig } from "@/store/connection-store"
import { DatabaseConnection } from "@/store/connection-store"
import { SSHAuthMethod } from "@/generated/database"
import { Database, Plus, Trash2, Play, Square, Loader2, ChevronDown, ChevronRight, Lock, Server, Cloud } from "lucide-react"
import { wailsEndpoints } from "@/lib/wails-api"

interface ConnectionFormData {
  name: string
  type: DatabaseTypeString
  host: string
  port: string
  database: string
  username: string
  password: string
  sslMode: string

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

const defaultFormData: ConnectionFormData = {
  name: '',
  type: 'postgresql',
  host: 'localhost',
  port: '5432',
  database: '',
  username: '',
  password: '',
  sslMode: 'prefer',

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
}

const DATABASE_TYPE_OPTIONS = [
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

export function ConnectionManager() {
  const {
    connections,
    addConnection,
    removeConnection,
    connectToDatabase,
    disconnectFromDatabase,
    isConnecting,
  } = useConnectionStore()

  const [isDialogOpen, setIsDialogOpen] = useState(false)
  const [formData, setFormData] = useState<ConnectionFormData>(defaultFormData)
  const [submitError, setSubmitError] = useState<string | null>(null)
  const [isTestingConnection, setIsTestingConnection] = useState(false)
  const [isSshSectionOpen, setIsSshSectionOpen] = useState(false)
  const [isVpcSectionOpen, setIsVpcSectionOpen] = useState(false)
  const [isAdvancedSshOpen, setIsAdvancedSshOpen] = useState(false)

  const buildConnectionPayload = () => {
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
      useTunnel: formData.useTunnel,
      sshTunnel,
      useVpc: formData.useVpc,
      vpcConfig,
      parameters,
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setSubmitError(null)
    setIsTestingConnection(true)

    const connectionData = buildConnectionPayload()

    try {
      const result = await wailsEndpoints.connections.test({
        ...connectionData,
        connection_timeout: 30,
      })

      if (!result.success) {
        throw new Error(result.message || 'Connection test failed')
      }

      addConnection(connectionData)
      setFormData(defaultFormData)
      setIsDialogOpen(false)
      setIsSshSectionOpen(false)
      setIsVpcSectionOpen(false)
      setIsAdvancedSshOpen(false)
    } catch (error) {
      setSubmitError(error instanceof Error ? error.message : 'Failed to validate connection')
    } finally {
      setIsTestingConnection(false)
    }
  }

  const handleConnect = async (connection: DatabaseConnection) => {
    try {
      if (connection.isConnected) {
        await disconnectFromDatabase(connection.id)
      } else {
        await connectToDatabase(connection.id)
      }
    } catch (error) {
      console.error('Connection toggle failed:', error)
    }
  }

  const getDefaultPort = (type: DatabaseTypeString): string => {
    switch (type) {
      case 'postgresql': return '5432'
      case 'mysql': return '3306'
      case 'mariadb': return '3306'
      case 'tidb': return '4000'
      case 'clickhouse': return '9000'
      case 'mongodb': return '27017'
      case 'elasticsearch': return '9200'
      case 'opensearch': return '9200'
      case 'sqlite': return ''
      case 'mssql': return '1433'
      default: return '5432'
    }
  }

  const handleTypeChange = (type: DatabaseTypeString) => {
    setFormData(prev => ({
      ...prev,
      type,
      port: getDefaultPort(type)
    }))
  }

  const requiresHostPort = (type: DatabaseTypeString): boolean => {
    return type !== 'sqlite'
  }

  const supportsSSL = (type: DatabaseTypeString): boolean => {
    return ['postgresql', 'mysql', 'mariadb', 'tidb', 'clickhouse'].includes(type)
  }

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold">Database Connections</h1>
          <p className="text-muted-foreground">Manage your database connections</p>
        </div>

        <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              Add Connection
            </Button>
          </DialogTrigger>
          <DialogContent className="sm:max-w-[600px] max-h-[90vh] overflow-y-auto">
            <form onSubmit={handleSubmit}>
              <DialogHeader>
                <DialogTitle>Add New Connection</DialogTitle>
                <DialogDescription>
                  Enter the details for your database connection.
                </DialogDescription>
              </DialogHeader>

              <div className="grid gap-4 py-4">
                {/* Basic Connection Info */}
                <div className="grid grid-cols-4 items-center gap-4">
                  <Label htmlFor="name" className="text-right">
                    Name
                  </Label>
                  <Input
                    id="name"
                    value={formData.name}
                    onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
                    className="col-span-3"
                    required
                  />
                </div>

                <div className="grid grid-cols-4 items-center gap-4">
                  <Label htmlFor="type" className="text-right">
                    Type
                  </Label>
                  <Select value={formData.type} onValueChange={handleTypeChange}>
                    <SelectTrigger className="col-span-3">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {DATABASE_TYPE_OPTIONS.map(option => (
                        <SelectItem key={option.value} value={option.value}>
                          {option.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                {/* Host and Port */}
                {requiresHostPort(formData.type) && (
                  <>
                    <div className="grid grid-cols-4 items-center gap-4">
                      <Label htmlFor="host" className="text-right">
                        Host
                      </Label>
                      <Input
                        id="host"
                        value={formData.host}
                        onChange={(e) => setFormData(prev => ({ ...prev, host: e.target.value }))}
                        className="col-span-3"
                        required
                      />
                    </div>

                    <div className="grid grid-cols-4 items-center gap-4">
                      <Label htmlFor="port" className="text-right">
                        Port
                      </Label>
                      <Input
                        id="port"
                        type="number"
                        value={formData.port}
                        onChange={(e) => setFormData(prev => ({ ...prev, port: e.target.value }))}
                        className="col-span-3"
                        required
                      />
                    </div>
                  </>
                )}

                {/* Database */}
                <div className="grid grid-cols-4 items-center gap-4">
                  <Label htmlFor="database" className="text-right">
                    {formData.type === 'mongodb' ? 'Database (optional)' :
                     formData.type === 'elasticsearch' || formData.type === 'opensearch' ? 'Index Pattern (optional)' :
                     'Database'}
                  </Label>
                  <Input
                    id="database"
                    value={formData.database}
                    onChange={(e) => setFormData(prev => ({ ...prev, database: e.target.value }))}
                    className="col-span-3"
                    required={formData.type !== 'mongodb' && formData.type !== 'elasticsearch' && formData.type !== 'opensearch'}
                  />
                </div>

                {/* Username and Password */}
                {requiresHostPort(formData.type) && (
                  <>
                    <div className="grid grid-cols-4 items-center gap-4">
                      <Label htmlFor="username" className="text-right">
                        Username
                      </Label>
                      <Input
                        id="username"
                        value={formData.username}
                        onChange={(e) => setFormData(prev => ({ ...prev, username: e.target.value }))}
                        className="col-span-3"
                        required={formData.type !== 'mongodb'}
                      />
                    </div>

                    <div className="grid grid-cols-4 items-center gap-4">
                      <Label htmlFor="password" className="text-right">
                        Password
                      </Label>
                      <div className="col-span-3">
                        <SecretInput
                          id="password"
                          value={formData.password}
                          onChange={(value) => setFormData(prev => ({ ...prev, password: value }))}
                          placeholder="Enter database password"
                        />
                      </div>
                    </div>
                  </>
                )}

                {/* SSL Mode */}
                {supportsSSL(formData.type) && (
                  <div className="grid grid-cols-4 items-center gap-4">
                    <Label htmlFor="sslMode" className="text-right">
                      SSL Mode
                    </Label>
                    <Select value={formData.sslMode} onValueChange={(value) => setFormData(prev => ({ ...prev, sslMode: value }))}>
                      <SelectTrigger className="col-span-3">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="disable">Disable</SelectItem>
                        <SelectItem value="allow">Allow</SelectItem>
                        <SelectItem value="prefer">Prefer</SelectItem>
                        <SelectItem value="require">Require</SelectItem>
                        <SelectItem value="verify-ca">Verify CA</SelectItem>
                        <SelectItem value="verify-full">Verify Full</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                )}

                {/* MongoDB-specific fields */}
                {formData.type === 'mongodb' && (
                  <>
                    <div className="grid grid-cols-4 items-center gap-4">
                      <Label htmlFor="mongoConnectionString" className="text-right">
                        Connection String (optional)
                      </Label>
                      <Input
                        id="mongoConnectionString"
                        value={formData.mongoConnectionString}
                        onChange={(e) => setFormData(prev => ({ ...prev, mongoConnectionString: e.target.value }))}
                        className="col-span-3"
                        placeholder="mongodb://..."
                      />
                    </div>
                    <div className="grid grid-cols-4 items-center gap-4">
                      <Label htmlFor="mongoAuthDatabase" className="text-right">
                        Auth Database
                      </Label>
                      <Input
                        id="mongoAuthDatabase"
                        value={formData.mongoAuthDatabase}
                        onChange={(e) => setFormData(prev => ({ ...prev, mongoAuthDatabase: e.target.value }))}
                        className="col-span-3"
                        placeholder="admin"
                      />
                    </div>
                  </>
                )}

                {/* Elasticsearch/OpenSearch-specific fields */}
                {(formData.type === 'elasticsearch' || formData.type === 'opensearch') && (
                  <>
                    <div className="grid grid-cols-4 items-center gap-4">
                      <Label htmlFor="elasticScheme" className="text-right">
                        Scheme
                      </Label>
                      <Select value={formData.elasticScheme} onValueChange={(value) => setFormData(prev => ({ ...prev, elasticScheme: value }))}>
                        <SelectTrigger className="col-span-3">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="http">HTTP</SelectItem>
                          <SelectItem value="https">HTTPS</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="grid grid-cols-4 items-center gap-4">
                      <Label htmlFor="elasticApiKey" className="text-right">
                        API Key (optional)
                      </Label>
                      <Input
                        id="elasticApiKey"
                        type="password"
                        value={formData.elasticApiKey}
                        onChange={(e) => setFormData(prev => ({ ...prev, elasticApiKey: e.target.value }))}
                        className="col-span-3"
                      />
                    </div>
                  </>
                )}

                {/* ClickHouse-specific fields */}
                {formData.type === 'clickhouse' && (
                  <div className="grid grid-cols-4 items-center gap-4">
                    <Label htmlFor="clickhouseNativeProtocol" className="text-right">
                      Native Protocol
                    </Label>
                    <div className="col-span-3 flex items-center space-x-2">
                      <Checkbox
                        id="clickhouseNativeProtocol"
                        checked={formData.clickhouseNativeProtocol}
                        onCheckedChange={(checked) => setFormData(prev => ({ ...prev, clickhouseNativeProtocol: checked === true }))}
                      />
                      <Label htmlFor="clickhouseNativeProtocol" className="text-sm text-muted-foreground">
                        Use native protocol (port 9000) instead of HTTP
                      </Label>
                    </div>
                  </div>
                )}

                {/* SSH Tunnel Configuration */}
                {requiresHostPort(formData.type) && (
                  <Collapsible open={isSshSectionOpen} onOpenChange={setIsSshSectionOpen}>
                    <div className="border rounded-lg p-4 mt-2">
                      <CollapsibleTrigger className="flex items-center justify-between w-full">
                        <div className="flex items-center space-x-2">
                          <Lock className="h-4 w-4" />
                          <Label className="text-sm font-semibold">SSH Tunnel Configuration</Label>
                        </div>
                        {isSshSectionOpen ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
                      </CollapsibleTrigger>

                      <CollapsibleContent className="mt-4 space-y-4">
                        <div className="flex items-center space-x-2">
                          <Checkbox
                            id="useTunnel"
                            checked={formData.useTunnel}
                            onCheckedChange={(checked) => setFormData(prev => ({ ...prev, useTunnel: checked === true }))}
                          />
                          <Label htmlFor="useTunnel">Enable SSH tunnel</Label>
                        </div>

                        {formData.useTunnel && (
                          <>
                            <div className="grid grid-cols-4 items-center gap-4">
                              <Label htmlFor="sshHost" className="text-right">
                                SSH Host
                              </Label>
                              <Input
                                id="sshHost"
                                value={formData.sshHost}
                                onChange={(e) => setFormData(prev => ({ ...prev, sshHost: e.target.value }))}
                                className="col-span-3"
                                required
                              />
                            </div>

                            <div className="grid grid-cols-4 items-center gap-4">
                              <Label htmlFor="sshPort" className="text-right">
                                SSH Port
                              </Label>
                              <Input
                                id="sshPort"
                                type="number"
                                value={formData.sshPort}
                                onChange={(e) => setFormData(prev => ({ ...prev, sshPort: e.target.value }))}
                                className="col-span-3"
                                required
                              />
                            </div>

                            <div className="grid grid-cols-4 items-center gap-4">
                              <Label htmlFor="sshUser" className="text-right">
                                SSH User
                              </Label>
                              <Input
                                id="sshUser"
                                value={formData.sshUser}
                                onChange={(e) => setFormData(prev => ({ ...prev, sshUser: e.target.value }))}
                                className="col-span-3"
                                required
                              />
                            </div>

                            <div className="grid grid-cols-4 items-center gap-4">
                              <Label htmlFor="sshAuthMethod" className="text-right">
                                Auth Method
                              </Label>
                              <Select
                                value={formData.sshAuthMethod}
                                onValueChange={(value) => setFormData(prev => ({ ...prev, sshAuthMethod: value as SSHAuthMethod }))}
                              >
                                <SelectTrigger className="col-span-3">
                                  <SelectValue />
                                </SelectTrigger>
                                <SelectContent>
                                  <SelectItem value={SSHAuthMethod.SSH_AUTH_METHOD_PASSWORD}>Password</SelectItem>
                                  <SelectItem value={SSHAuthMethod.SSH_AUTH_METHOD_PRIVATE_KEY}>Private Key</SelectItem>
                                </SelectContent>
                              </Select>
                            </div>

                            {formData.sshAuthMethod === SSHAuthMethod.SSH_AUTH_METHOD_PASSWORD && (
                              <div className="grid grid-cols-4 items-center gap-4">
                                <Label htmlFor="sshPassword" className="text-right">
                                  SSH Password
                                </Label>
                                <div className="col-span-3">
                                  <SecretInput
                                    id="sshPassword"
                                    value={formData.sshPassword}
                                    onChange={(value) => setFormData(prev => ({ ...prev, sshPassword: value }))}
                                    placeholder="Enter SSH password"
                                    required
                                  />
                                </div>
                              </div>
                            )}

                            {formData.sshAuthMethod === SSHAuthMethod.SSH_AUTH_METHOD_PRIVATE_KEY && (
                              <div className="grid grid-cols-4 items-start gap-4">
                                <Label className="text-right pt-2">
                                  Private Key
                                </Label>
                                <div className="col-span-3">
                                  <PemKeyUpload
                                    onUpload={(keyContent) => setFormData(prev => ({ ...prev, sshPrivateKey: keyContent }))}
                                    onError={(error) => console.error('PEM key error:', error)}
                                  />
                                  <div className="mt-2">
                                    <SecretInput
                                      value={formData.sshPrivateKeyPassphrase}
                                      onChange={(value) => setFormData(prev => ({ ...prev, sshPrivateKeyPassphrase: value }))}
                                      placeholder="Key passphrase (if encrypted)"
                                      label="Key Passphrase (Optional)"
                                    />
                                  </div>
                                </div>
                              </div>
                            )}

                            {/* Advanced SSH Options */}
                            <Collapsible open={isAdvancedSshOpen} onOpenChange={setIsAdvancedSshOpen}>
                              <CollapsibleTrigger className="flex items-center space-x-2 text-sm text-muted-foreground hover:text-foreground">
                                {isAdvancedSshOpen ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
                                <span>Advanced SSH Options</span>
                              </CollapsibleTrigger>

                              <CollapsibleContent className="mt-4 space-y-4">
                                <div className="grid grid-cols-4 items-center gap-4">
                                  <Label htmlFor="sshKnownHostsPath" className="text-right">
                                    Known Hosts Path
                                  </Label>
                                  <Input
                                    id="sshKnownHostsPath"
                                    value={formData.sshKnownHostsPath}
                                    onChange={(e) => setFormData(prev => ({ ...prev, sshKnownHostsPath: e.target.value }))}
                                    className="col-span-3"
                                    placeholder="~/.ssh/known_hosts"
                                  />
                                </div>

                                <div className="grid grid-cols-4 items-center gap-4">
                                  <Label htmlFor="sshStrictHostKeyChecking" className="text-right">
                                    Strict Host Key Checking
                                  </Label>
                                  <div className="col-span-3 flex items-center space-x-2">
                                    <Checkbox
                                      id="sshStrictHostKeyChecking"
                                      checked={formData.sshStrictHostKeyChecking}
                                      onCheckedChange={(checked) => setFormData(prev => ({ ...prev, sshStrictHostKeyChecking: checked === true }))}
                                    />
                                  </div>
                                </div>

                                <div className="grid grid-cols-4 items-center gap-4">
                                  <Label htmlFor="sshTimeoutSeconds" className="text-right">
                                    Timeout (seconds)
                                  </Label>
                                  <Input
                                    id="sshTimeoutSeconds"
                                    type="number"
                                    value={formData.sshTimeoutSeconds}
                                    onChange={(e) => setFormData(prev => ({ ...prev, sshTimeoutSeconds: e.target.value }))}
                                    className="col-span-3"
                                  />
                                </div>

                                <div className="grid grid-cols-4 items-center gap-4">
                                  <Label htmlFor="sshKeepAliveIntervalSeconds" className="text-right">
                                    Keep-Alive (seconds)
                                  </Label>
                                  <Input
                                    id="sshKeepAliveIntervalSeconds"
                                    type="number"
                                    value={formData.sshKeepAliveIntervalSeconds}
                                    onChange={(e) => setFormData(prev => ({ ...prev, sshKeepAliveIntervalSeconds: e.target.value }))}
                                    className="col-span-3"
                                  />
                                </div>
                              </CollapsibleContent>
                            </Collapsible>
                          </>
                        )}
                      </CollapsibleContent>
                    </div>
                  </Collapsible>
                )}

                {/* VPC Configuration */}
                {requiresHostPort(formData.type) && (
                  <Collapsible open={isVpcSectionOpen} onOpenChange={setIsVpcSectionOpen}>
                    <div className="border rounded-lg p-4 mt-2">
                      <CollapsibleTrigger className="flex items-center justify-between w-full">
                        <div className="flex items-center space-x-2">
                          <Cloud className="h-4 w-4" />
                          <Label className="text-sm font-semibold">VPC Configuration</Label>
                        </div>
                        {isVpcSectionOpen ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
                      </CollapsibleTrigger>

                      <CollapsibleContent className="mt-4 space-y-4">
                        <div className="flex items-center space-x-2">
                          <Checkbox
                            id="useVpc"
                            checked={formData.useVpc}
                            onCheckedChange={(checked) => setFormData(prev => ({ ...prev, useVpc: checked === true }))}
                          />
                          <Label htmlFor="useVpc">Enable VPC configuration</Label>
                        </div>

                        {formData.useVpc && (
                          <>
                            <div className="grid grid-cols-4 items-center gap-4">
                              <Label htmlFor="vpcId" className="text-right">
                                VPC ID
                              </Label>
                              <Input
                                id="vpcId"
                                value={formData.vpcId}
                                onChange={(e) => setFormData(prev => ({ ...prev, vpcId: e.target.value }))}
                                className="col-span-3"
                                placeholder="vpc-xxxxxxxxx"
                                required
                              />
                            </div>

                            <div className="grid grid-cols-4 items-center gap-4">
                              <Label htmlFor="subnetId" className="text-right">
                                Subnet ID
                              </Label>
                              <Input
                                id="subnetId"
                                value={formData.subnetId}
                                onChange={(e) => setFormData(prev => ({ ...prev, subnetId: e.target.value }))}
                                className="col-span-3"
                                placeholder="subnet-xxxxxxxxx"
                                required
                              />
                            </div>

                            <div className="grid grid-cols-4 items-center gap-4">
                              <Label htmlFor="securityGroupIds" className="text-right">
                                Security Group IDs
                              </Label>
                              <Input
                                id="securityGroupIds"
                                value={formData.securityGroupIds}
                                onChange={(e) => setFormData(prev => ({ ...prev, securityGroupIds: e.target.value }))}
                                className="col-span-3"
                                placeholder="sg-xxx, sg-yyy"
                                required
                              />
                            </div>

                            <div className="grid grid-cols-4 items-center gap-4">
                              <Label htmlFor="privateLinkService" className="text-right">
                                Private Link Service
                              </Label>
                              <Input
                                id="privateLinkService"
                                value={formData.privateLinkService}
                                onChange={(e) => setFormData(prev => ({ ...prev, privateLinkService: e.target.value }))}
                                className="col-span-3"
                              />
                            </div>

                            <div className="grid grid-cols-4 items-center gap-4">
                              <Label htmlFor="endpointServiceName" className="text-right">
                                Endpoint Service Name
                              </Label>
                              <Input
                                id="endpointServiceName"
                                value={formData.endpointServiceName}
                                onChange={(e) => setFormData(prev => ({ ...prev, endpointServiceName: e.target.value }))}
                                className="col-span-3"
                              />
                            </div>
                          </>
                        )}
                      </CollapsibleContent>
                    </div>
                  </Collapsible>
                )}
              </div>

              <DialogFooter>
                <div className="flex flex-col items-start gap-2 w-full">
                  {submitError && (
                    <p className="text-sm text-destructive">{submitError}</p>
                  )}
                  <Button type="submit" disabled={isTestingConnection}>
                    {isTestingConnection ? (
                      <>
                        <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                        Testing...
                      </>
                    ) : (
                      'Add Connection'
                    )}
                  </Button>
                </div>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {connections.map((connection) => (
          <Card key={connection.id}>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium flex items-center">
                <Database className="h-4 w-4 mr-2" />
                {connection.name}
              </CardTitle>
              <Button
                variant="ghost"
                size="sm"
                onClick={async () => {
                  if (connection.isConnected) {
                    await disconnectFromDatabase(connection.id)
                  }
                  removeConnection(connection.id)
                }}
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </CardHeader>
            <CardContent>
              <div className="text-xs text-muted-foreground space-y-1">
                <div className="flex items-center gap-1">
                  <span className="font-medium">Type:</span> {connection.type}
                </div>
                <div className="flex items-center gap-1">
                  <span className="font-medium">Database:</span> {connection.database || 'N/A'}
                </div>
                {connection.host && (
                  <div className="flex items-center gap-1">
                    <Server className="h-3 w-3" />
                    <span>{connection.host}:{connection.port}</span>
                  </div>
                )}
                {connection.username && (
                  <div className="flex items-center gap-1">
                    <span className="font-medium">User:</span> {connection.username}
                  </div>
                )}
                {connection.useTunnel && (
                  <div className="flex items-center gap-1 text-primary">
                    <Lock className="h-3 w-3" />
                    <span>SSH Tunnel</span>
                  </div>
                )}
                {connection.useVpc && (
                  <div className="flex items-center gap-1 text-primary">
                    <Cloud className="h-3 w-3" />
                    <span>VPC</span>
                  </div>
                )}
              </div>

              <div className="mt-4 flex items-center justify-between">
                <div className="flex items-center space-x-2">
                  <div
                    className={`h-2 w-2 rounded-full ${
                      connection.isConnected ? 'bg-primary' : 'bg-muted-foreground'
                    }`}
                  />
                  <span className="text-xs">
                    {connection.isConnected ? 'Connected' : 'Disconnected'}
                  </span>
                </div>

                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => handleConnect(connection)}
                  disabled={isConnecting}
                >
                  {connection.isConnected ? (
                    <Square className="h-4 w-4" />
                  ) : (
                    <Play className="h-4 w-4" />
                  )}
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}

        {connections.length === 0 && (
          <Card className="col-span-full">
            <CardContent className="flex flex-col items-center justify-center py-12">
              <Database className="h-12 w-12 text-muted-foreground mb-4" />
              <CardTitle className="mb-2">No connections configured</CardTitle>
              <CardDescription className="mb-4">
                Add your first database connection to get started
              </CardDescription>
              <Button onClick={() => setIsDialogOpen(true)}>
                <Plus className="h-4 w-4 mr-2" />
                Add Connection
              </Button>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  )
}
