import { Calendar,Database, Edit, Eye, Lock, Plus, Trash2 } from 'lucide-react'
import React, { useEffect,useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'

interface WailsApp {
  ListSyntheticViews: () => Promise<ViewSummary[]>
  GetSyntheticView: (viewId: string) => Promise<ViewDefinition>
  DeleteSyntheticView: (viewId: string) => Promise<void>
}

interface ViewDefinition {
  id: string
  name: string
  description: string
  version: string
  columns: Array<{ name: string; type: string }>
  sources: Array<{ connectionIdOrName: string; schema: string; table: string }>
  createdAt: string
  updatedAt: string
}

interface ViewSummary {
  id: string
  name: string
  description: string
  version: string
  createdAt: string
  updatedAt: string
}

interface SyntheticViewsManagerProps {
  onViewSelect?: (view: ViewDefinition) => void
  onViewEdit?: (view: ViewDefinition) => void
  onViewDelete?: (viewId: string) => void
}

export const SyntheticViewsManager: React.FC<SyntheticViewsManagerProps> = ({
  onViewSelect,
  onViewEdit,
  onViewDelete
}) => {
  const [views, setViews] = useState<ViewSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [selectedView, setSelectedView] = useState<ViewDefinition | null>(null)

  // Load synthetic views
  const loadViews = async () => {
    try {
      setLoading(true)
      setError(null)
      
      // Load synthetic views
      try {
        // @ts-expect-error - Wails API may not be available in development
        const App = await import('../../wailsjs/go/main/App') as WailsApp
        const viewsList = await App.ListSyntheticViews()
        setViews(viewsList)
      } catch (importError) {
        console.warn('Wails API not available, using mock data:', importError)
        // Fallback to mock data for development
        setViews([])
      }
    } catch (err) {
      console.error('Failed to load synthetic views:', err)
      setError(err instanceof Error ? err.message : 'Failed to load views')
    } finally {
      setLoading(false)
    }
  }

  // Load view details
  const loadViewDetails = async (viewId: string) => {
    try {
      try {
        // @ts-expect-error - Wails API may not be available in development
        const App = await import('../../wailsjs/go/main/App') as WailsApp
        const view = await App.GetSyntheticView(viewId)
        setSelectedView(view)
        if (onViewSelect) {
          onViewSelect(view)
        }
      } catch (importError) {
        console.warn('Wails API not available:', importError)
        setError('Wails API not available')
      }
    } catch (err) {
      console.error('Failed to load view details:', err)
      setError(err instanceof Error ? err.message : 'Failed to load view details')
    }
  }

  // Delete view
  const handleDeleteView = async (viewId: string) => {
    if (!window.confirm('Are you sure you want to delete this synthetic view? This action cannot be undone.')) {
      return
    }

    try {
      try {
        // @ts-expect-error - Wails API may not be available in development
        const App = await import('../../wailsjs/go/main/App') as WailsApp
        await App.DeleteSyntheticView(viewId)
        
        // Remove from local state
        setViews(prev => prev.filter(v => v.id !== viewId))
        
        if (onViewDelete) {
          onViewDelete(viewId)
        }
      } catch (importError) {
        console.warn('Wails API not available:', importError)
        setError('Wails API not available')
      }
    } catch (err) {
      console.error('Failed to delete view:', err)
      setError(err instanceof Error ? err.message : 'Failed to delete view')
    }
  }

  // Edit view
  const handleEditView = async (view: ViewSummary) => {
    try {
      try {
        // @ts-expect-error - Wails API may not be available in development
        const App = await import('../../wailsjs/go/main/App') as WailsApp
        const viewDetails = await App.GetSyntheticView(view.id)
        if (onViewEdit) {
          onViewEdit(viewDetails)
        }
      } catch (importError) {
        console.warn('Wails API not available:', importError)
        setError('Wails API not available')
      }
  } catch (err) {
      console.error('Failed to load view for editing:', err)
      setError(err instanceof Error ? err.message : 'Failed to load view for editing')
    }
  }

  // View details
  const handleViewDetails = async (view: ViewSummary) => {
    await loadViewDetails(view.id)
  }

  useEffect(() => {
    loadViews()
  }, [])

  if (loading) {
    return (
      <div className="flex items-center justify-center p-8">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-4"></div>
          <p className="text-muted-foreground">Loading synthetic views...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold">Synthetic Views</h2>
          <p className="text-muted-foreground">
            Federated views that join tables across multiple databases
          </p>
        </div>
        
        <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              Create View
            </Button>
          </DialogTrigger>
          <DialogContent className="max-w-2xl">
            <DialogHeader>
              <DialogTitle>Create Synthetic View</DialogTitle>
              <DialogDescription>
                Create a new federated view that joins tables across multiple databases
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4">
              <div className="text-center py-8">
                <Database className="h-12 w-12 mx-auto mb-4 text-muted-foreground" />
                <p className="text-muted-foreground">
                  Visual query builder coming soon. For now, you can create views programmatically.
                </p>
              </div>
            </div>
          </DialogContent>
        </Dialog>
      </div>

      {/* Error State */}
      {error && (
        <div className="bg-destructive/10 border border-destructive/20 rounded-lg p-4">
          <p className="text-destructive text-sm">{error}</p>
        </div>
      )}

      {/* Views Grid */}
      {views.length === 0 ? (
        <div className="text-center py-12">
          <Database className="h-12 w-12 mx-auto mb-4 text-muted-foreground" />
          <h3 className="text-lg font-semibold mb-2">No Synthetic Views</h3>
          <p className="text-muted-foreground mb-4">
            Create your first federated view to join tables across multiple databases
          </p>
          <Button onClick={() => setShowCreateDialog(true)}>
            <Plus className="h-4 w-4 mr-2" />
            Create Your First View
          </Button>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {views.map((view) => (
            <Card key={view.id} className="hover:shadow-md transition-shadow">
              <CardHeader className="pb-3">
                <div className="flex items-start justify-between">
                  <div className="space-y-1">
                    <CardTitle className="text-lg">{view.name}</CardTitle>
                    <CardDescription className="text-sm">
                      {view.description || 'No description'}
                    </CardDescription>
                  </div>
                  <Badge variant="secondary" className="ml-2">
                    <Lock className="h-3 w-3 mr-1" />
                    Read-only
                  </Badge>
                </div>
              </CardHeader>
              
              <CardContent className="pt-0">
                <div className="space-y-3">
                  <div className="flex items-center text-xs text-muted-foreground">
                    <Calendar className="h-3 w-3 mr-1" />
                    Updated {new Date(view.updatedAt).toLocaleDateString()}
                  </div>
                  
                  <div className="flex items-center gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleViewDetails(view)}
                      className="flex-1"
                    >
                      <Eye className="h-3 w-3 mr-1" />
                      View
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleEditView(view)}
                      className="flex-1"
                    >
                      <Edit className="h-3 w-3 mr-1" />
                      Edit
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleDeleteView(view.id)}
                      className="text-destructive hover:text-destructive"
                    >
                      <Trash2 className="h-3 w-3" />
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* View Details Dialog */}
      {selectedView && (
        <Dialog open={!!selectedView} onOpenChange={() => setSelectedView(null)}>
          <DialogContent className="max-w-4xl">
            <DialogHeader>
              <DialogTitle>{selectedView.name}</DialogTitle>
              <DialogDescription>
                {selectedView.description || 'Synthetic federated view'}
              </DialogDescription>
            </DialogHeader>
            
            <div className="space-y-6">
              {/* View Info */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label className="text-sm font-medium">Version</Label>
                  <p className="text-sm text-muted-foreground">{selectedView.version}</p>
                </div>
                <div>
                  <Label className="text-sm font-medium">Schema</Label>
                  <p className="text-sm text-muted-foreground">synthetic</p>
                </div>
              </div>

              {/* Columns */}
              <div>
                <Label className="text-sm font-medium mb-2 block">Columns</Label>
                <div className="space-y-2">
                  {selectedView.columns.map((col, index) => (
                    <div key={index} className="flex items-center justify-between p-2 bg-muted/50 rounded">
                      <span className="font-mono text-sm">{col.name}</span>
                      <Badge variant="outline" className="text-xs">
                        {col.type}
                      </Badge>
                    </div>
                  ))}
                </div>
              </div>

              {/* Sources */}
              <div>
                <Label className="text-sm font-medium mb-2 block">Source Tables</Label>
                <div className="space-y-2">
                  {selectedView.sources.map((source, index) => (
                    <div key={index} className="flex items-center justify-between p-2 bg-muted/50 rounded">
                      <span className="font-mono text-sm">
                        {source.connectionIdOrName}.{source.schema}.{source.table}
                      </span>
                    </div>
                  ))}
                </div>
              </div>

              {/* Actions */}
              <div className="flex justify-end gap-2">
                <Button variant="outline" onClick={() => setSelectedView(null)}>
                  Close
                </Button>
                <Button onClick={() => {
                  setSelectedView(null)
                  handleEditView({ id: selectedView.id, name: selectedView.name, description: selectedView.description, version: selectedView.version, createdAt: '', updatedAt: '' })
                }}>
                  <Edit className="h-4 w-4 mr-2" />
                  Edit View
                </Button>
              </div>
            </div>
          </DialogContent>
        </Dialog>
      )}
    </div>
  )
}
