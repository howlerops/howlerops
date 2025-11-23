import { ChevronDown, ChevronRight, Folder, FolderOpen, FolderPlus, MoreHorizontal, Trash2 } from 'lucide-react'
import { useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { cn } from '@/lib/utils'
import type { FolderNode } from '@/types/reports'

interface FolderTreeProps {
  folders: FolderNode[]
  activeFolder?: FolderNode
  onSelectFolder: (folder: FolderNode | undefined) => void
  onCreateFolder: () => void
  onRenameFolder: (folder: FolderNode) => void
  onDeleteFolder: (folder: FolderNode) => void
  onDropReport?: (reportId: string, folderId: string | undefined) => void
}

export function FolderTree({
  folders,
  activeFolder,
  onSelectFolder,
  onCreateFolder,
  onRenameFolder,
  onDeleteFolder,
  onDropReport,
}: FolderTreeProps) {
  return (
    <div className="space-y-1">
      {/* All Reports (root) */}
      <button
        className={cn(
          'w-full flex items-center gap-2 px-2 py-1.5 rounded-md transition',
          'hover:bg-muted text-left',
          !activeFolder && 'bg-primary/10 border border-primary/20'
        )}
        onClick={() => onSelectFolder(undefined)}
        onDrop={(e) => {
          if (onDropReport) {
            e.preventDefault()
            const reportId = e.dataTransfer.getData('reportId')
            if (reportId) onDropReport(reportId, undefined)
          }
        }}
        onDragOver={(e) => {
          if (onDropReport) e.preventDefault()
        }}
      >
        <Folder className="h-4 w-4" />
        <span className="flex-1 font-medium">All Reports</span>
      </button>

      {/* Root folders */}
      {folders.map((folder) => (
        <FolderNodeComponent
          key={folder.id}
          folder={folder}
          level={0}
          active={activeFolder?.id === folder.id}
          onSelect={() => onSelectFolder(folder)}
          onRename={() => onRenameFolder(folder)}
          onDelete={() => onDeleteFolder(folder)}
          onDrop={onDropReport}
        />
      ))}

      {/* Add folder button */}
      <Button variant="ghost" className="w-full justify-start" onClick={onCreateFolder}>
        <FolderPlus className="mr-2 h-4 w-4" /> New Folder
      </Button>
    </div>
  )
}

interface FolderNodeProps {
  folder: FolderNode
  level: number
  active: boolean
  onSelect: () => void
  onRename: () => void
  onDelete: () => void
  onDrop?: (reportId: string, folderId: string | undefined) => void
}

function FolderNodeComponent({ folder, level, active, onSelect, onRename, onDelete, onDrop }: FolderNodeProps) {
  const [expanded, setExpanded] = useState(folder.expanded ?? true)
  const [isDragOver, setIsDragOver] = useState(false)
  const hasChildren = folder.children.length > 0

  return (
    <div>
      <div
        className={cn(
          'group flex items-center gap-2 px-2 py-1.5 rounded-md cursor-pointer transition',
          'hover:bg-muted',
          active && 'bg-primary/10 border border-primary/20',
          isDragOver && 'bg-primary/5 border-2 border-primary/40',
        )}
        style={{ paddingLeft: `${0.5 + level}rem` }}
        onClick={onSelect}
        onDrop={(e) => {
          setIsDragOver(false)
          if (onDrop) {
            e.preventDefault()
            e.stopPropagation()
            const reportId = e.dataTransfer.getData('reportId')
            if (reportId) onDrop(reportId, folder.id)
          }
        }}
        onDragOver={(e) => {
          if (onDrop) {
            e.preventDefault()
            e.stopPropagation()
            setIsDragOver(true)
          }
        }}
        onDragLeave={() => setIsDragOver(false)}
      >
        {hasChildren ? (
          <button
            onClick={(e) => {
              e.stopPropagation()
              setExpanded(!expanded)
            }}
            className="p-0.5 hover:bg-accent rounded"
          >
            {expanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
          </button>
        ) : (
          <div className="w-5" />
        )}

        <span className="text-lg">
          {folder.icon || (expanded && hasChildren ? <FolderOpen className="h-4 w-4" /> : <Folder className="h-4 w-4" />)}
        </span>
        <span className="flex-1 font-medium truncate" title={folder.name}>
          {folder.name}
        </span>
        <Badge variant="secondary" className="text-xs tabular-nums">
          {folder.reportCount}
        </Badge>

        <DropdownMenu>
          <DropdownMenuTrigger asChild onClick={(e) => e.stopPropagation()}>
            <Button variant="ghost" size="icon" className="h-6 w-6 opacity-0 group-hover:opacity-100">
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={onRename}>Rename</DropdownMenuItem>
            <DropdownMenuItem>Change Icon</DropdownMenuItem>
            <DropdownMenuItem>New Subfolder</DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={onDelete} className="text-destructive">
              <Trash2 className="mr-2 h-4 w-4" /> Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      {/* Render children if expanded */}
      {expanded && hasChildren && (
        <div>
          {folder.children.map((child) => (
            <FolderNodeComponent
              key={child.id}
              folder={child}
              level={level + 1}
              active={active}
              onSelect={onSelect}
              onRename={onRename}
              onDelete={onDelete}
              onDrop={onDrop}
            />
          ))}
        </div>
      )}
    </div>
  )
}
