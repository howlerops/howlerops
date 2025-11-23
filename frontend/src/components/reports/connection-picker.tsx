import { Database } from 'lucide-react'
import { useEffect } from 'react'

import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useConnectionsStore } from '@/store/connections-store'

interface ConnectionPickerProps {
  value?: string
  onChange: (connectionId: string) => void
  disabled?: boolean
  label?: string
}

const DATABASE_TYPE_LABELS: Record<string, string> = {
  postgres: 'PostgreSQL',
  mysql: 'MySQL',
  sqlite: 'SQLite',
  turso: 'Turso',
}

const DATABASE_TYPE_COLORS: Record<string, string> = {
  postgres: 'text-blue-600',
  mysql: 'text-orange-600',
  sqlite: 'text-gray-600',
  turso: 'text-green-600',
}

export function ConnectionPicker({
  value,
  onChange,
  disabled,
  label = 'Database Connection',
}: ConnectionPickerProps) {
  const connections = useConnectionsStore((state) => state.connections)
  const loading = useConnectionsStore((state) => state.loading)
  const fetchConnections = useConnectionsStore((state) => state.fetchConnections)

  useEffect(() => {
    fetchConnections().catch(console.error)
  }, [fetchConnections])

  return (
    <div className="space-y-2">
      <Label>{label}</Label>
      <Select value={value} onValueChange={onChange} disabled={disabled || loading}>
        <SelectTrigger>
          <SelectValue placeholder={loading ? 'Loading connections...' : 'Select database connection...'} />
        </SelectTrigger>
        <SelectContent>
          {connections.length === 0 && !loading && (
            <div className="px-2 py-6 text-center text-sm text-muted-foreground">
              No connections available. Create one in the Connections page.
            </div>
          )}
          {connections.map((conn) => (
            <SelectItem key={conn.id} value={conn.id}>
              <div className="flex items-center gap-2">
                <Database
                  className={`h-4 w-4 ${DATABASE_TYPE_COLORS[conn.database_type] || 'text-gray-600'}`}
                />
                <div className="flex flex-col">
                  <span className="font-medium">{conn.name}</span>
                  <span className="text-xs text-muted-foreground">
                    {DATABASE_TYPE_LABELS[conn.database_type] || conn.database_type}
                    {conn.description && ` • ${conn.description}`}
                  </span>
                </div>
              </div>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}
