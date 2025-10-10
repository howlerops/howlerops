import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
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
import { useConnectionStore } from "@/store/connection-store"
import { DatabaseConnection } from "@/store/connection-store"
import { Database, Plus, Trash2, Play, Square, Loader2 } from "lucide-react"
import { wailsEndpoints } from "@/lib/wails-api"

interface ConnectionFormData {
  name: string
  type: 'postgresql' | 'mysql' | 'sqlite' | 'mssql'
  host: string
  port: string
  database: string
  username: string
  password: string
}

const defaultFormData: ConnectionFormData = {
  name: '',
  type: 'postgresql',
  host: 'localhost',
  port: '5432',
  database: '',
  username: '',
  password: '',
}

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

  const buildConnectionPayload = () => {
    const port = formData.port ? parseInt(formData.port, 10) : 0

    return {
      name: formData.name,
      type: formData.type,
      host: formData.type === 'sqlite' ? '' : formData.host,
      port: formData.type === 'sqlite' ? 0 : port,
      database: formData.database,
      username: formData.type === 'sqlite' ? '' : formData.username,
      password: formData.type === 'sqlite' ? '' : formData.password,
      parameters: {},
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

  const getDefaultPort = (type: string) => {
    switch (type) {
      case 'postgresql': return '5432'
      case 'mysql': return '3306'
      case 'sqlite': return ''
      case 'mssql': return '1433'
      default: return '5432'
    }
  }

  const handleTypeChange = (type: 'postgresql' | 'mysql' | 'sqlite' | 'mssql') => {
    setFormData(prev => ({
      ...prev,
      type,
      port: getDefaultPort(type)
    }))
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
          <DialogContent className="sm:max-w-[425px]">
            <form onSubmit={handleSubmit}>
              <DialogHeader>
                <DialogTitle>Add New Connection</DialogTitle>
                <DialogDescription>
                  Enter the details for your database connection.
                </DialogDescription>
              </DialogHeader>

              <div className="grid gap-4 py-4">
                <div className="grid grid-cols-4 items-center gap-4">
                  <label htmlFor="name" className="text-right">
                    Name
                  </label>
                  <Input
                    id="name"
                    value={formData.name}
                    onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
                    className="col-span-3"
                    required
                  />
                </div>

                <div className="grid grid-cols-4 items-center gap-4">
                  <label htmlFor="type" className="text-right">
                    Type
                  </label>
                  <Select value={formData.type} onValueChange={handleTypeChange}>
                    <SelectTrigger className="col-span-3">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="postgresql">PostgreSQL</SelectItem>
                      <SelectItem value="mysql">MySQL</SelectItem>
                      <SelectItem value="sqlite">SQLite</SelectItem>
                      <SelectItem value="mssql">SQL Server</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                {formData.type !== 'sqlite' && (
                  <>
                    <div className="grid grid-cols-4 items-center gap-4">
                      <label htmlFor="host" className="text-right">
                        Host
                      </label>
                      <Input
                        id="host"
                        value={formData.host}
                        onChange={(e) => setFormData(prev => ({ ...prev, host: e.target.value }))}
                        className="col-span-3"
                        required
                      />
                    </div>

                    <div className="grid grid-cols-4 items-center gap-4">
                      <label htmlFor="port" className="text-right">
                        Port
                      </label>
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

                <div className="grid grid-cols-4 items-center gap-4">
                  <label htmlFor="database" className="text-right">
                    Database
                  </label>
                  <Input
                    id="database"
                    value={formData.database}
                    onChange={(e) => setFormData(prev => ({ ...prev, database: e.target.value }))}
                    className="col-span-3"
                    required
                  />
                </div>

                {formData.type !== 'sqlite' && (
                  <>
                    <div className="grid grid-cols-4 items-center gap-4">
                      <label htmlFor="username" className="text-right">
                        Username
                      </label>
                      <Input
                        id="username"
                        value={formData.username}
                        onChange={(e) => setFormData(prev => ({ ...prev, username: e.target.value }))}
                        className="col-span-3"
                        required
                      />
                    </div>

                    <div className="grid grid-cols-4 items-center gap-4">
                      <label htmlFor="password" className="text-right">
                        Password
                      </label>
                      <Input
                        id="password"
                        type="password"
                        value={formData.password}
                        onChange={(e) => setFormData(prev => ({ ...prev, password: e.target.value }))}
                        className="col-span-3"
                      />
                    </div>
                  </>
                )}
              </div>

              <DialogFooter>
                <div className="flex flex-col items-start gap-2">
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
                <div>Type: {connection.type}</div>
                <div>Database: {connection.database}</div>
                {connection.host && <div>Host: {connection.host}:{connection.port}</div>}
                {connection.username && <div>User: {connection.username}</div>}
              </div>

              <div className="mt-4 flex items-center justify-between">
                <div className="flex items-center space-x-2">
                  <div
                    className={`h-2 w-2 rounded-full ${
                      connection.isConnected ? 'bg-green-500' : 'bg-gray-400'
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
