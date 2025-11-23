# Report Organization System - Implementation Guide

## Overview

This document outlines the implementation of a comprehensive folder/tag organization system and report templates for HowlerOps. The system transforms report management from a flat list to an organized, searchable library similar to Notion/Google Drive.

## Implementation Status

### ✅ Completed

#### Backend (Go)

1. **Storage Layer** (`backend-go/pkg/storage/`)
   - ✅ `report_folders.go` - Complete folder hierarchy management
   - ✅ `reports.go` - Updated with `starred` and `starred_at` fields
   - ✅ Database schema updates for folders, tags, templates, and starred reports

2. **Data Models**
   - ✅ `ReportFolder` - Folder structure with parent/child relationships
   - ✅ `Tag` - Tag with usage counts
   - ✅ `ReportTemplate` - Template with category, icon, and usage tracking
   - ✅ `Report` - Extended with `Starred` and `StarredAt` fields

3. **Storage Methods**
   - ✅ `CreateFolder`, `ListFolders`, `UpdateFolder`, `DeleteFolder`
   - ✅ `MoveReportToFolder` - Drag-and-drop support
   - ✅ `ListTags`, `CreateOrUpdateTag` - Tag management
   - ✅ `SaveTemplate`, `ListTemplates`, `GetTemplate`, `IncrementTemplateUsage`
   - ✅ `ToggleStarred`, `ListStarredReports` - Favorites functionality

#### Frontend (TypeScript/React)

1. **Type Definitions** (`frontend/src/types/reports.ts`)
   - ✅ `ReportFolder`, `FolderNode` - Folder types with tree structure
   - ✅ `Tag` - Tag with count
   - ✅ `ReportTemplate` - Template types
   - ✅ `ReportSummary` - Extended with `starred` and `starredAt`

2. **Components**
   - ✅ `folder-tree.tsx` - Hierarchical folder tree with drag-and-drop

### 🚧 Remaining Work

#### Backend API Endpoints (Needs Integration)

The storage layer is complete but needs HTTP API endpoints. Create these routes (suggested location: `backend-go/internal/server/report_routes.go`):

```go
// Folders
POST   /api/folders                  // Create folder
GET    /api/folders                  // List all folders
PUT    /api/folders/:id              // Update folder
DELETE /api/folders/:id              // Delete folder
POST   /api/reports/:id/move         // Move report to folder

// Tags
GET    /api/tags                     // List all tags with counts
POST   /api/tags                     // Create/update tag
PUT    /api/reports/:id/tags         // Update report tags

// Templates
GET    /api/templates                // List templates (optional category filter)
GET    /api/templates/:id            // Get template
POST   /api/templates                // Create template from report
POST   /api/templates/:id/instantiate // Create report from template

// Favorites
POST   /api/reports/:id/star         // Toggle star status
GET    /api/reports/starred          // List starred reports
```

**Example API Handler Implementation:**

```go
package server

import (
    "encoding/json"
    "net/http"
    "github.com/gorilla/mux"
)

// In your HTTP server setup:
func (s *HTTPServer) registerReportRoutes(r *mux.Router) {
    // Folders
    r.HandleFunc("/api/folders", s.handleListFolders).Methods("GET")
    r.HandleFunc("/api/folders", s.handleCreateFolder).Methods("POST")
    r.HandleFunc("/api/folders/{id}", s.handleUpdateFolder).Methods("PUT")
    r.HandleFunc("/api/folders/{id}", s.handleDeleteFolder).Methods("DELETE")
    r.HandleFunc("/api/reports/{id}/move", s.handleMoveReport).Methods("POST")

    // Tags
    r.HandleFunc("/api/tags", s.handleListTags).Methods("GET")
    r.HandleFunc("/api/tags", s.handleCreateTag).Methods("POST")

    // Templates
    r.HandleFunc("/api/templates", s.handleListTemplates).Methods("GET")
    r.HandleFunc("/api/templates/{id}", s.handleGetTemplate).Methods("GET")
    r.HandleFunc("/api/templates", s.handleCreateTemplate).Methods("POST")
    r.HandleFunc("/api/templates/{id}/instantiate", s.handleInstantiateTemplate).Methods("POST")

    // Starred
    r.HandleFunc("/api/reports/{id}/star", s.handleToggleStar).Methods("POST")
    r.HandleFunc("/api/reports/starred", s.handleListStarred).Methods("GET")
}

func (s *HTTPServer) handleListFolders(w http.ResponseWriter, r *http.Request) {
    folders, err := s.folderStorage.ListFolders()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(folders)
}

func (s *HTTPServer) handleToggleStar(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]

    newStarred, err := s.reportStorage.ToggleStarred(id)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(map[string]bool{"starred": newStarred})
}

// Add similar handlers for other endpoints...
```

#### Frontend Components (Need Implementation)

1. **Tag Selector** (`frontend/src/components/reports/tag-selector.tsx`)
   - Multi-select tag picker
   - Tag creation dialog
   - Color picker for tags
   - Tag count display

2. **Template Gallery** (`frontend/src/components/reports/template-gallery.tsx`)
   - Visual grid of templates
   - Category filters
   - Template preview cards
   - Search functionality
   - "Create from Template" flow

3. **Report Navigator** (`frontend/src/components/reports/report-navigator.tsx`)
   - Enhanced sidebar replacing simple list
   - Tabs: Folders / Tags / Recent / Starred
   - Search with filtering
   - Quick actions (New Report, From Template)

4. **Built-in Templates** (Need creation)
   - Executive Dashboard
   - Sales Pipeline
   - User Cohort Analysis
   - Custom Analytics
   - Blank Canvas

#### Frontend Services (Need Extension)

Extend `frontend/src/services/reports-service.ts` with:

```typescript
// Folders
async createFolder(folder: Partial<ReportFolder>): Promise<ReportFolder>
async listFolders(): Promise<ReportFolder[]>
async updateFolder(id: string, updates: Partial<ReportFolder>): Promise<ReportFolder>
async deleteFolder(id: string): Promise<void>
async moveReport(reportId: string, folderId?: string): Promise<void>

// Tags
async listTags(): Promise<Tag[]>
async createTag(name: string, color?: string): Promise<Tag>
async updateReportTags(reportId: string, tags: string[]): Promise<void>

// Templates
async listTemplates(category?: string): Promise<ReportTemplate[]>
async getTemplate(id: string): Promise<ReportTemplate>
async createTemplate(report: ReportRecord, meta: TemplateMetadata): Promise<ReportTemplate>
async instantiateTemplate(templateId: string): Promise<ReportRecord>

// Starred
async toggleStar(reportId: string): Promise<boolean>
async listStarred(): Promise<ReportSummary[]>
```

#### State Management Updates

Extend `frontend/src/store/report-store.ts`:

```typescript
interface ReportStoreState {
  // Existing fields...

  // New fields
  folders: FolderNode[]
  tags: Tag[]
  templates: ReportTemplate[]
  activeFolder?: FolderNode
  selectedTags: string[]
  showStarredOnly: boolean

  // New actions
  fetchFolders: () => Promise<void>
  createFolder: (name: string, parentId?: string) => Promise<void>
  deleteFolder: (id: string) => Promise<void>
  moveReportToFolder: (reportId: string, folderId?: string) => Promise<void>

  fetchTags: () => Promise<void>
  updateReportTags: (reportId: string, tags: string[]) => Promise<void>

  fetchTemplates: () => Promise<void>
  createFromTemplate: (templateId: string) => Promise<void>
  saveAsTemplate: (report: ReportRecord, metadata: TemplateMetadata) => Promise<void>

  toggleStar: (reportId: string) => Promise<void>
  setShowStarredOnly: (value: boolean) => void
}
```

#### Database Migration

Add migration script to update existing databases:

```sql
-- Migration: Add report organization features
-- Version: 20250122_report_organization

-- Add starred fields to reports table
ALTER TABLE reports ADD COLUMN starred BOOLEAN DEFAULT FALSE;
ALTER TABLE reports ADD COLUMN starred_at DATETIME;

-- Create index for starred reports
CREATE INDEX IF NOT EXISTS idx_reports_starred ON reports(starred DESC, starred_at DESC);

-- Create folders table
CREATE TABLE IF NOT EXISTS report_folders (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    parent_id TEXT,
    color TEXT,
    icon TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (parent_id) REFERENCES report_folders(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_folders_parent_id ON report_folders(parent_id);

-- Create tags table
CREATE TABLE IF NOT EXISTS report_tags (
    name TEXT PRIMARY KEY,
    color TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create templates table
CREATE TABLE IF NOT EXISTS report_templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    category TEXT NOT NULL,
    thumbnail TEXT,
    icon TEXT,
    tags TEXT,
    definition TEXT NOT NULL,
    filter TEXT,
    featured BOOLEAN DEFAULT FALSE,
    usage_count INTEGER DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_templates_category ON report_templates(category);
CREATE INDEX IF NOT EXISTS idx_templates_featured ON report_templates(featured);

-- Seed default templates
INSERT INTO report_templates (id, name, description, category, icon, tags, definition, filter, featured)
VALUES
(
    'exec-dashboard',
    'Executive Dashboard',
    'High-level KPIs and trends for leadership',
    'analytics',
    '📈',
    '["kpi", "metrics", "executive"]',
    '{"layout":[],"components":[]}',
    '{"fields":[]}',
    true
);
```

## Architecture Decisions

### Folder Structure

- **Hierarchical**: Folders can have parent folders (unlimited nesting)
- **Single parent**: Each folder has 0 or 1 parent (tree structure)
- **Cascade delete**: Deleting a folder cascades to children
- **Reports**: Assigned to folders via `folder` field (stores folder ID or NULL for root)

### Tag System

- **Denormalized counts**: Tag counts computed on-the-fly from report tags
- **Flexible creation**: Tags created on-demand when used
- **Color coding**: Optional color per tag for visual distinction
- **No hierarchy**: Tags are flat (not nested)

### Template System

- **Categories**: analytics, operations, sales, custom
- **Usage tracking**: Templates track how many times instantiated
- **Featured flag**: Highlight popular templates
- **Connection agnostic**: Templates don't store connection IDs (user selects at instantiation)

### Starred/Favorites

- **Boolean + timestamp**: `starred` flag + `starred_at` for sorting
- **User-independent**: Current implementation doesn't filter by user (add `user_id` for multi-tenancy)

## UI/UX Design Principles

### Information Architecture

1. **Three-level hierarchy**: Folders → Reports → Components
2. **Multiple views**: Support folders, tags, recent, and starred views
3. **Search across all**: Full-text search across names, descriptions, and tags
4. **Visual distinction**: Icons, colors, badges for quick scanning

### Interaction Patterns

1. **Drag-and-drop**: Move reports between folders
2. **Quick actions**: Context menus on right-click
3. **Keyboard shortcuts**: Navigate folders, search, create
4. **Progressive disclosure**: Collapse/expand folder trees

### Scalability

- **Virtualization**: Use react-window for 1000+ items
- **Lazy loading**: Load folder contents on expand
- **Debounced search**: 300ms debounce on search input
- **Pagination**: For templates and large tag lists

## Testing Checklist

### Backend

- [ ] Folder CRUD operations
- [ ] Folder hierarchy (parent/child relationships)
- [ ] Cascade delete on folder removal
- [ ] Move report between folders
- [ ] Tag creation and counting
- [ ] Template save/load/instantiate
- [ ] Toggle starred status
- [ ] List starred reports
- [ ] Database migrations

### Frontend

- [ ] Folder tree rendering
- [ ] Folder expand/collapse
- [ ] Drag-and-drop reports to folders
- [ ] Create/rename/delete folders
- [ ] Tag multi-select
- [ ] Tag filtering
- [ ] Template gallery display
- [ ] Create report from template
- [ ] Save report as template
- [ ] Star/unstar reports
- [ ] Starred reports view
- [ ] Search functionality
- [ ] Performance with 1000+ reports

## Performance Considerations

### Backend

- **Indexes**: Already created on folder_id, parent_id, starred, category
- **Denormalized counts**: Compute tag/folder counts in queries, not cached
- **Pagination**: For large template/tag lists

### Frontend

- **Folder tree**: Load full tree once, cache in state
- **Tag counts**: Update on save (no real-time sync needed)
- **Template thumbnails**: Lazy load images
- **Search**: Client-side for <100 reports, server-side for 100+

## Next Steps

1. **Create API routes** - Wire up storage methods to HTTP endpoints
2. **Implement tag selector** - Multi-select component with color picker
3. **Build template gallery** - Visual grid with category filters
4. **Create report navigator** - Enhanced sidebar with tabs
5. **Add built-in templates** - Create 5-10 example templates
6. **Database migration** - Script to update existing deployments
7. **Integration testing** - End-to-end tests for all workflows
8. **Documentation** - User guide for organization features

## Files Modified/Created

### Backend (Go)

- ✅ `backend-go/pkg/storage/report_folders.go` (NEW)
- ✅ `backend-go/pkg/storage/reports.go` (MODIFIED)
- ⏳ `backend-go/internal/server/report_routes.go` (NEED TO CREATE)

### Frontend (TypeScript/React)

- ✅ `frontend/src/types/reports.ts` (MODIFIED)
- ✅ `frontend/src/components/reports/folder-tree.tsx` (NEW)
- ⏳ `frontend/src/components/reports/tag-selector.tsx` (NEED TO CREATE)
- ⏳ `frontend/src/components/reports/template-gallery.tsx` (NEED TO CREATE)
- ⏳ `frontend/src/components/reports/report-navigator.tsx` (NEED TO CREATE)
- ⏳ `frontend/src/services/reports-service.ts` (NEED TO EXTEND)
- ⏳ `frontend/src/store/report-store.ts` (NEED TO EXTEND)
- ⏳ `frontend/src/pages/reports.tsx` (NEED TO MODIFY)

## Questions for Review

1. **Multi-user support**: Should starred reports and folders be user-specific?
2. **Permissions**: Do folders need access control?
3. **Template sharing**: Should users share templates with team?
4. **Folder colors**: Auto-assign or user-selected?
5. **Tag autocomplete**: Pre-populate common tags?

## Resources

- Folder Tree: `/Users/jacob/projects/amplifier/ai_working/howlerops/frontend/src/components/reports/folder-tree.tsx`
- Storage Layer: `/Users/jacob/projects/amplifier/ai_working/howlerops/backend-go/pkg/storage/report_folders.go`
- Type Definitions: `/Users/jacob/projects/amplifier/ai_working/howlerops/frontend/src/types/reports.ts`
