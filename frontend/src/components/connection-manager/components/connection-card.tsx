import { Activity, Cloud, Database, Lock, Pencil, Play, Server, Square, Tag, Trash2 } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"

import type { DatabaseConnection } from "../types"

interface ConnectionCardProps {
  connection: DatabaseConnection
  isConnecting: boolean
  onEdit: (connection: DatabaseConnection) => void
  onDelete: (connection: DatabaseConnection) => void
  onConnect: (connection: DatabaseConnection) => void
  onDiagnostics: (connection: DatabaseConnection) => void
}

/**
 * Card component displaying a single database connection
 */
export function ConnectionCard({
  connection,
  isConnecting,
  onEdit,
  onDelete,
  onConnect,
  onDiagnostics,
}: ConnectionCardProps) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium flex items-center">
          <Database className="h-4 w-4 mr-2" />
          {connection.name}
        </CardTitle>
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onEdit(connection)}
            title="Edit connection"
          >
            <Pencil className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onDelete(connection)}
            title="Delete connection"
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
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

        {/* Environment tags */}
        <div className="mt-3">
          <div className="flex items-center gap-2 text-xs font-medium text-muted-foreground">
            <Tag className="h-3 w-3" />
            <span>Environment</span>
          </div>
          {connection.environments && connection.environments.length > 0 ? (
            <div className="mt-1 flex flex-wrap gap-2">
              {connection.environments.map((env) => (
                <Badge key={`${connection.id}-${env}`} variant="secondary" className="text-xs">
                  {env}
                </Badge>
              ))}
            </div>
          ) : (
            <p className="mt-1 text-xs text-muted-foreground italic">Not assigned</p>
          )}
        </div>

        {/* Connection status and actions */}
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

          <div className="flex items-center gap-1">
            {connection.isConnected && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onDiagnostics(connection)}
                title="View diagnostics"
              >
                <Activity className="h-4 w-4" />
              </Button>
            )}
            <Button
              variant="outline"
              size="sm"
              onClick={() => onConnect(connection)}
              disabled={isConnecting}
            >
              {connection.isConnected ? (
                <Square className="h-4 w-4" />
              ) : (
                <Play className="h-4 w-4" />
              )}
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
