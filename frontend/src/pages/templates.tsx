/**
 * TemplatesPage Component
 * Main page for browsing and managing query templates
 */

import React, { useState, useEffect, useMemo } from 'react'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  Search,
  Plus,
  Filter,
  X,
  AlertCircle,
  FileCode,
} from 'lucide-react'
import { useTemplatesStore } from '@/store/templates-store'
import { TemplateCard, TemplateCardSkeleton } from '@/components/templates/TemplateCard'
import { TemplateExecutor } from '@/components/templates/TemplateExecutor'
import { ScheduleCreator } from '@/components/templates/ScheduleCreator'
import type { QueryTemplate, TemplateSortBy } from '@/types/templates'

const CATEGORIES = [
  { value: 'all', label: 'All Templates', count: 0 },
  { value: 'reporting', label: 'Reporting', count: 0 },
  { value: 'analytics', label: 'Analytics', count: 0 },
  { value: 'maintenance', label: 'Maintenance', count: 0 },
  { value: 'custom', label: 'Custom', count: 0 },
]

export function TemplatesPage() {
  const {
    templates,
    loading,
    error,
    filters: _filters,
    sortBy,
    fetchTemplates,
    deleteTemplate,
    duplicateTemplate,
    setFilters,
    setSortBy,
    getFilteredTemplates,
  } = useTemplatesStore()

  const [searchQuery, setSearchQuery] = useState('')
  const [selectedCategory, setSelectedCategory] = useState('all')
  const [selectedTags, setSelectedTags] = useState<string[]>([])
  const [executingTemplate, setExecutingTemplate] = useState<QueryTemplate | null>(null)
  const [schedulingTemplate, setSchedulingTemplate] = useState<QueryTemplate | null>(null)
  const [_showCreateDialog, setShowCreateDialog] = useState(false)

  // Load templates on mount
  useEffect(() => {
    fetchTemplates()
  }, [fetchTemplates])

  // Update filters when search/category/tags change
  useEffect(() => {
    setFilters({
      search: searchQuery,
      category: selectedCategory === 'all' ? undefined : selectedCategory,
      tags: selectedTags.length > 0 ? selectedTags : undefined,
    })
  }, [searchQuery, selectedCategory, selectedTags, setFilters])

  // Get filtered templates
  const filteredTemplates = useMemo(() => {
    return getFilteredTemplates()
  }, [getFilteredTemplates])

  // Get all unique tags from templates
  const allTags = useMemo(() => {
    const tagSet = new Set<string>()
    templates.forEach((t) => t.tags.forEach((tag) => tagSet.add(tag)))
    return Array.from(tagSet).sort()
  }, [templates])

  // Update category counts
  const categoriesWithCounts = useMemo(() => {
    return CATEGORIES.map((cat) => ({
      ...cat,
      count:
        cat.value === 'all'
          ? templates.length
          : templates.filter((t) => t.category === cat.value).length,
    }))
  }, [templates])

  const handleExecute = (template: QueryTemplate) => {
    setExecutingTemplate(template)
  }

  const handleSchedule = (template: QueryTemplate) => {
    setSchedulingTemplate(template)
  }

  const handleDelete = async (template: QueryTemplate) => {
    if (confirm(`Are you sure you want to delete "${template.name}"?`)) {
      try {
        await deleteTemplate(template.id)
      } catch (error) {
        console.error('Failed to delete template:', error)
      }
    }
  }

  const handleDuplicate = async (template: QueryTemplate) => {
    try {
      await duplicateTemplate(template.id)
    } catch (error) {
      console.error('Failed to duplicate template:', error)
    }
  }

  const toggleTag = (tag: string) => {
    setSelectedTags((prev) =>
      prev.includes(tag) ? prev.filter((t) => t !== tag) : [...prev, tag]
    )
  }

  const clearFilters = () => {
    setSearchQuery('')
    setSelectedCategory('all')
    setSelectedTags([])
  }

  const hasActiveFilters = searchQuery || selectedCategory !== 'all' || selectedTags.length > 0

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b bg-card">
        <div className="container mx-auto px-6 py-8">
          <div className="flex items-center justify-between mb-6">
            <div>
              <h1 className="text-3xl font-bold flex items-center gap-3">
                <FileCode className="h-8 w-8" />
                Query Templates
              </h1>
              <p className="text-muted-foreground mt-2">
                Browse, execute, and schedule reusable query templates
              </p>
            </div>

            <Button onClick={() => setShowCreateDialog(true)}>
              <Plus className="mr-2 h-4 w-4" />
              New Template
            </Button>
          </div>

          {/* Search and Filters */}
          <div className="flex flex-col lg:flex-row gap-4">
            {/* Search */}
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Search templates..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-10"
              />
            </div>

            {/* Sort */}
            <Select value={sortBy} onValueChange={(v: TemplateSortBy) => setSortBy(v)}>
              <SelectTrigger className="w-[180px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="usage">Most Used</SelectItem>
                <SelectItem value="newest">Newest First</SelectItem>
                <SelectItem value="updated">Recently Updated</SelectItem>
                <SelectItem value="name">Name (A-Z)</SelectItem>
              </SelectContent>
            </Select>

            {hasActiveFilters && (
              <Button variant="outline" onClick={clearFilters}>
                <X className="mr-2 h-4 w-4" />
                Clear Filters
              </Button>
            )}
          </div>

          {/* Category Tabs */}
          <Tabs
            value={selectedCategory}
            onValueChange={setSelectedCategory}
            className="mt-4"
          >
            <TabsList>
              {categoriesWithCounts.map((cat) => (
                <TabsTrigger key={cat.value} value={cat.value}>
                  {cat.label}
                  <Badge variant="secondary" className="ml-2">
                    {cat.count}
                  </Badge>
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>

          {/* Tag Filters */}
          {allTags.length > 0 && (
            <div className="mt-4">
              <div className="flex items-center gap-2 flex-wrap">
                <Filter className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm text-muted-foreground">Tags:</span>
                {allTags.slice(0, 10).map((tag) => (
                  <Badge
                    key={tag}
                    variant={selectedTags.includes(tag) ? 'default' : 'outline'}
                    className="cursor-pointer"
                    onClick={() => toggleTag(tag)}
                  >
                    {tag}
                  </Badge>
                ))}
                {allTags.length > 10 && (
                  <span className="text-xs text-muted-foreground">
                    +{allTags.length - 10} more
                  </span>
                )}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Content */}
      <div className="container mx-auto px-6 py-8">
        {/* Error Alert */}
        {error && (
          <Alert variant="destructive" className="mb-6">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        {/* Loading State */}
        {loading && templates.length === 0 && (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {Array.from({ length: 6 }).map((_, i) => (
              <TemplateCardSkeleton key={i} />
            ))}
          </div>
        )}

        {/* Empty State */}
        {!loading && filteredTemplates.length === 0 && (
          <div className="text-center py-12">
            <FileCode className="h-16 w-16 mx-auto text-muted-foreground mb-4" />
            <h3 className="text-lg font-semibold mb-2">No templates found</h3>
            <p className="text-muted-foreground mb-6">
              {hasActiveFilters
                ? 'Try adjusting your filters or search query'
                : 'Get started by creating your first template'}
            </p>
            {!hasActiveFilters && (
              <Button onClick={() => setShowCreateDialog(true)}>
                <Plus className="mr-2 h-4 w-4" />
                Create Template
              </Button>
            )}
          </div>
        )}

        {/* Templates Grid */}
        {!loading && filteredTemplates.length > 0 && (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {filteredTemplates.map((template) => (
              <TemplateCard
                key={template.id}
                template={template}
                onExecute={handleExecute}
                onSchedule={handleSchedule}
                onDelete={handleDelete}
                onDuplicate={handleDuplicate}
              />
            ))}
          </div>
        )}
      </div>

      {/* Modals */}
      {executingTemplate && (
        <TemplateExecutor
          template={executingTemplate}
          open={!!executingTemplate}
          onClose={() => setExecutingTemplate(null)}
        />
      )}

      {schedulingTemplate && (
        <ScheduleCreator
          template={schedulingTemplate}
          open={!!schedulingTemplate}
          onClose={() => setSchedulingTemplate(null)}
        />
      )}
    </div>
  )
}
