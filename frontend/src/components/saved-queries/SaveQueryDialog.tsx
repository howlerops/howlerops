/**
 * Save Query Dialog
 *
 * Modal dialog for saving and editing saved queries
 *
 * Features:
 * - Create new saved query
 * - Edit existing query
 * - Folder and tag organization
 * - Favorite toggle
 * - Validation
 *
 * @module components/saved-queries/SaveQueryDialog
 */

import { Loader2,Plus, Star, X } from 'lucide-react'
import { useEffect,useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { useSavedQueriesStore } from '@/store/saved-queries-store'
import type { SavedQueryRecord } from '@/types/storage'

interface SaveQueryDialogProps {
  /** Whether dialog is open */
  open: boolean

  /** Callback when dialog should close */
  onClose: () => void

  /** User ID for ownership */
  userId: string

  /** Initial query text (for new queries) */
  initialQuery?: string

  /** Existing query to edit */
  existingQuery?: SavedQueryRecord

  /** Callback after successful save */
  onSaved?: (query: SavedQueryRecord) => void
}

export function SaveQueryDialog({
  open,
  onClose,
  userId,
  initialQuery,
  existingQuery,
  onSaved,
}: SaveQueryDialogProps) {
  const { folders, tags, saveQuery, updateQuery } = useSavedQueriesStore()

  // Form state
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [queryText, setQueryText] = useState('')
  const [selectedFolder, setSelectedFolder] = useState<string>('')
  const [selectedTags, setSelectedTags] = useState<string[]>([])
  const [isFavorite, setIsFavorite] = useState(false)
  const [newFolderInput, setNewFolderInput] = useState('')
  const [newTagInput, setNewTagInput] = useState('')
  const [showNewFolder, setShowNewFolder] = useState(false)
  const [isSaving, setIsSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const isEditMode = !!existingQuery

  // Initialize form when dialog opens
  useEffect(() => {
    let mounted = true

    if (open && mounted) {
      if (existingQuery) {
        // Edit mode: populate with existing data
        setTitle(existingQuery.title)
        setDescription(existingQuery.description || '')
        setQueryText(existingQuery.query_text)
        setSelectedFolder(existingQuery.folder || '')
        setSelectedTags(existingQuery.tags)
        setIsFavorite(existingQuery.is_favorite)
      } else {
        // Create mode: use initial query or empty
        setTitle('')
        setDescription('')
        setQueryText(initialQuery || '')
        setSelectedFolder('')
        setSelectedTags([])
        setIsFavorite(false)
      }
      setNewFolderInput('')
      setNewTagInput('')
      setShowNewFolder(false)
      setError(null)
    }

    return () => {
      mounted = false
    }
  }, [open, existingQuery, initialQuery])

  const handleSave = async () => {
    // Validation
    if (!title.trim()) {
      setError('Title is required')
      return
    }

    if (title.trim().length > 200) {
      setError('Title must be 200 characters or less')
      return
    }

    if (!queryText.trim()) {
      setError('Query text is required')
      return
    }

    if (description && description.length > 1000) {
      setError('Description must be 1000 characters or less')
      return
    }

    setIsSaving(true)
    setError(null)

    try {
      const folder = showNewFolder && newFolderInput.trim()
        ? newFolderInput.trim()
        : selectedFolder || undefined

      if (isEditMode && existingQuery) {
        // Update existing query
        await updateQuery(existingQuery.id, {
          title: title.trim(),
          description: description.trim() || undefined,
          query_text: queryText,
          tags: selectedTags,
          folder,
          is_favorite: isFavorite,
        })

        const updated = { ...existingQuery, title: title.trim() }
        onSaved?.(updated as SavedQueryRecord)
      } else {
        // Create new query
        const query = await saveQuery({
          user_id: userId,
          title: title.trim(),
          description: description.trim() || undefined,
          query_text: queryText,
          tags: selectedTags,
          folder,
          is_favorite: isFavorite,
        })

        onSaved?.(query)
      }

      onClose()
    } catch (err) {
      console.error('Failed to save query:', err)
      setError(err instanceof Error ? err.message : 'Failed to save query')
    } finally {
      setIsSaving(false)
    }
  }

  const handleAddTag = () => {
    const tag = newTagInput.trim()
    if (tag && !selectedTags.includes(tag)) {
      setSelectedTags([...selectedTags, tag])
      setNewTagInput('')
    }
  }

  const handleRemoveTag = (tag: string) => {
    setSelectedTags(selectedTags.filter((t) => t !== tag))
  }

  return (
    <Dialog open={open} onOpenChange={(isOpen) => !isOpen && onClose()}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            {isEditMode ? 'Edit Saved Query' : 'Save Query'}
          </DialogTitle>
          <DialogDescription>
            {isEditMode
              ? 'Update the details of your saved query'
              : 'Save this query for easy access later'}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          {/* Title */}
          <div className="space-y-2">
            <Label htmlFor="query-title" className="flex items-center gap-2">
              Title <span className="text-destructive">*</span>
            </Label>
            <Input
              id="query-title"
              placeholder="e.g., Top 10 Users by Revenue"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              autoFocus
            />
          </div>

          {/* Description */}
          <div className="space-y-2">
            <Label htmlFor="query-description">Description</Label>
            <Textarea
              id="query-description"
              placeholder="Optional description or notes about this query"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
            />
          </div>

          {/* Query Text (read-only display) */}
          <div className="space-y-2">
            <Label>Query</Label>
            <div className="border rounded-md p-3 bg-muted/30 max-h-40 overflow-y-auto">
              <pre className="text-xs font-mono whitespace-pre-wrap">
                {queryText || 'No query text'}
              </pre>
            </div>
          </div>

          {/* Folder */}
          <div className="space-y-2">
            <Label htmlFor="query-folder">Folder</Label>
            {showNewFolder ? (
              <div className="flex gap-2">
                <Input
                  placeholder="New folder name"
                  value={newFolderInput}
                  onChange={(e) => setNewFolderInput(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      setShowNewFolder(false)
                    }
                  }}
                />
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    setShowNewFolder(false)
                    setNewFolderInput('')
                  }}
                >
                  Cancel
                </Button>
              </div>
            ) : (
              <div className="flex gap-2">
                <Select value={selectedFolder} onValueChange={setSelectedFolder}>
                  <SelectTrigger id="query-folder">
                    <SelectValue placeholder="No folder" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="">No folder</SelectItem>
                    {folders.map((folder) => (
                      <SelectItem key={folder} value={folder}>
                        {folder}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Button
                  variant="outline"
                  size="icon"
                  onClick={() => setShowNewFolder(true)}
                  title="Create new folder"
                >
                  <Plus className="h-4 w-4" />
                </Button>
              </div>
            )}
          </div>

          {/* Tags */}
          <div className="space-y-2">
            <Label>Tags</Label>
            <div className="flex flex-wrap gap-2 mb-2">
              {selectedTags.map((tag) => (
                <Badge key={tag} variant="secondary" className="gap-1">
                  {tag}
                  <button
                    onClick={() => handleRemoveTag(tag)}
                    className="ml-1 hover:text-destructive"
                  >
                    <X className="h-3 w-3" />
                  </button>
                </Badge>
              ))}
            </div>
            <div className="flex gap-2">
              <Input
                placeholder="Add a tag"
                value={newTagInput}
                onChange={(e) => setNewTagInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault()
                    handleAddTag()
                  }
                }}
              />
              <Button variant="outline" size="sm" onClick={handleAddTag}>
                Add
              </Button>
            </div>
            {tags.length > 0 && (
              <div className="flex flex-wrap gap-1 pt-2">
                <span className="text-xs text-muted-foreground">
                  Existing tags:
                </span>
                {tags.map((tag) => (
                  <Badge
                    key={tag}
                    variant="outline"
                    className="cursor-pointer text-xs"
                    onClick={() => {
                      if (!selectedTags.includes(tag)) {
                        setSelectedTags([...selectedTags, tag])
                      }
                    }}
                  >
                    {tag}
                  </Badge>
                ))}
              </div>
            )}
          </div>

          {/* Favorite */}
          <div className="flex items-center space-x-2">
            <Checkbox
              id="query-favorite"
              checked={isFavorite}
              onCheckedChange={(checked) => setIsFavorite(checked === true)}
            />
            <Label
              htmlFor="query-favorite"
              className="flex items-center gap-2 cursor-pointer"
            >
              <Star className={`h-4 w-4 ${isFavorite ? 'fill-yellow-400 text-yellow-400' : ''}`} />
              Mark as favorite
            </Label>
          </div>

          {/* Error message */}
          {error && (
            <div className="text-sm text-destructive bg-destructive/10 px-3 py-2 rounded-md">
              {error}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={isSaving}>
            Cancel
          </Button>
          <Button onClick={handleSave} disabled={isSaving}>
            {isSaving ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                Saving...
              </>
            ) : (
              <>{isEditMode ? 'Update' : 'Save'}</>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
