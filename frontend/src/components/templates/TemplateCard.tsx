/**
 * TemplateCard Component
 * Displays a single query template in a card layout
 */

import React from 'react'
import { useRouter } from 'next/navigation'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Play,
  Copy,
  Edit,
  Trash2,
  Calendar,
  MoreVertical,
  Users,
  Lock,
  TrendingUp,
} from 'lucide-react'
import type { QueryTemplate } from '@/types/templates'
import { formatDistanceToNow } from 'date-fns'

interface TemplateCardProps {
  template: QueryTemplate
  onExecute?: (template: QueryTemplate) => void
  onEdit?: (template: QueryTemplate) => void
  onDelete?: (template: QueryTemplate) => void
  onDuplicate?: (template: QueryTemplate) => void
  onSchedule?: (template: QueryTemplate) => void
}

const CATEGORY_COLORS = {
  reporting: 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300',
  analytics: 'bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-300',
  maintenance: 'bg-orange-100 text-orange-700 dark:bg-orange-900 dark:text-orange-300',
  custom: 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300',
}

export function TemplateCard({
  template,
  onExecute,
  onEdit,
  onDelete,
  onDuplicate,
  onSchedule,
}: TemplateCardProps) {
  const _router = useRouter()

  const categoryColor = CATEGORY_COLORS[template.category] || CATEGORY_COLORS.custom

  return (
    <Card className="group hover:shadow-lg transition-all duration-200 flex flex-col h-full">
      <CardHeader>
        <div className="flex items-start justify-between gap-2">
          <div className="flex-1 min-w-0">
            <CardTitle className="text-lg truncate" title={template.name}>
              {template.name}
            </CardTitle>
            <CardDescription className="line-clamp-2 mt-1">
              {template.description || 'No description'}
            </CardDescription>
          </div>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="ghost"
                size="sm"
                className="h-8 w-8 p-0 opacity-0 group-hover:opacity-100 transition-opacity"
              >
                <MoreVertical className="h-4 w-4" />
                <span className="sr-only">Open menu</span>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              {onExecute && (
                <DropdownMenuItem onClick={() => onExecute(template)}>
                  <Play className="mr-2 h-4 w-4" />
                  Execute Template
                </DropdownMenuItem>
              )}
              {onSchedule && (
                <DropdownMenuItem onClick={() => onSchedule(template)}>
                  <Calendar className="mr-2 h-4 w-4" />
                  Schedule Query
                </DropdownMenuItem>
              )}
              {onDuplicate && (
                <DropdownMenuItem onClick={() => onDuplicate(template)}>
                  <Copy className="mr-2 h-4 w-4" />
                  Duplicate
                </DropdownMenuItem>
              )}
              <DropdownMenuSeparator />
              {onEdit && (
                <DropdownMenuItem onClick={() => onEdit(template)}>
                  <Edit className="mr-2 h-4 w-4" />
                  Edit
                </DropdownMenuItem>
              )}
              {onDelete && (
                <DropdownMenuItem
                  onClick={() => onDelete(template)}
                  className="text-red-600 dark:text-red-400"
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  Delete
                </DropdownMenuItem>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        <div className="flex flex-wrap gap-2 mt-3">
          <Badge variant="secondary" className={categoryColor}>
            {template.category}
          </Badge>
          {template.is_public ? (
            <Badge variant="outline" className="gap-1">
              <Users className="h-3 w-3" />
              Public
            </Badge>
          ) : (
            <Badge variant="outline" className="gap-1">
              <Lock className="h-3 w-3" />
              Private
            </Badge>
          )}
        </div>
      </CardHeader>

      <CardContent className="flex-1">
        {/* Tags */}
        {template.tags.length > 0 && (
          <div className="flex flex-wrap gap-1.5 mb-3">
            {template.tags.slice(0, 3).map((tag) => (
              <Badge key={tag} variant="outline" className="text-xs">
                {tag}
              </Badge>
            ))}
            {template.tags.length > 3 && (
              <Badge variant="outline" className="text-xs">
                +{template.tags.length - 3} more
              </Badge>
            )}
          </div>
        )}

        {/* SQL Preview */}
        <div className="bg-muted rounded-md p-3 font-mono text-xs overflow-hidden">
          <code className="line-clamp-3 text-muted-foreground">
            {template.sql_template}
          </code>
        </div>

        {/* Parameters Count */}
        {template.parameters.length > 0 && (
          <p className="text-xs text-muted-foreground mt-2">
            {template.parameters.length} parameter{template.parameters.length !== 1 ? 's' : ''}
          </p>
        )}
      </CardContent>

      <CardFooter className="flex items-center justify-between text-xs text-muted-foreground border-t pt-4">
        <div className="flex items-center gap-1">
          <TrendingUp className="h-3 w-3" />
          <span>{template.usage_count} uses</span>
        </div>

        <div title={new Date(template.updated_at).toLocaleString()}>
          Updated {formatDistanceToNow(new Date(template.updated_at), { addSuffix: true })}
        </div>
      </CardFooter>

      {/* Primary Action Button */}
      {onExecute && (
        <div className="px-6 pb-4">
          <Button
            onClick={() => onExecute(template)}
            className="w-full"
            variant="default"
          >
            <Play className="mr-2 h-4 w-4" />
            Use Template
          </Button>
        </div>
      )}
    </Card>
  )
}

// Skeleton loader for template cards
export function TemplateCardSkeleton() {
  return (
    <Card className="flex flex-col h-full">
      <CardHeader>
        <div className="space-y-2">
          <div className="h-5 bg-muted rounded animate-pulse w-3/4" />
          <div className="h-4 bg-muted rounded animate-pulse w-full" />
          <div className="h-4 bg-muted rounded animate-pulse w-2/3" />
        </div>
        <div className="flex gap-2 mt-3">
          <div className="h-5 bg-muted rounded animate-pulse w-20" />
          <div className="h-5 bg-muted rounded animate-pulse w-16" />
        </div>
      </CardHeader>

      <CardContent className="flex-1">
        <div className="bg-muted rounded-md h-20 animate-pulse" />
      </CardContent>

      <CardFooter className="border-t pt-4">
        <div className="h-4 bg-muted rounded animate-pulse w-full" />
      </CardFooter>
    </Card>
  )
}
