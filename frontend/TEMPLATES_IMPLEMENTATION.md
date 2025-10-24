# Query Templates & Scheduling - Implementation Guide

## Overview

Complete frontend implementation for query templates and scheduled queries in SQL Studio. This provides a full-featured template library, execution engine, and scheduling system with cron-based automation.

## Features Implemented

### 1. Query Templates
- **Template Library**: Browse, search, and filter templates
- **Category Management**: Reporting, Analytics, Maintenance, Custom
- **Tag System**: Multi-tag filtering and search
- **Template Execution**: Dynamic parameter input with validation
- **Template CRUD**: Create, edit, duplicate, delete templates
- **Usage Tracking**: Track template usage and popularity

### 2. Scheduled Queries
- **Cron Scheduling**: Visual cron builder with presets
- **Schedule Management**: Create, pause, resume, delete schedules
- **Execution History**: Timeline view with detailed logs
- **Status Monitoring**: Active, paused, failed states
- **Notifications**: Email alerts for execution results
- **Result Storage**: Database or S3 storage options

### 3. User Experience
- **Responsive Design**: Mobile-first layout
- **Advanced Search**: Fuzzy search across name, description, SQL, tags
- **Real-time Validation**: Parameter validation with error messages
- **SQL Preview**: Syntax highlighting with interpolated parameters
- **Execution Results**: Tabular display with export options
- **Loading States**: Skeletons and progressive loading

## File Structure

```
frontend/src/
├── types/
│   └── templates.ts              # TypeScript type definitions
├── lib/
│   ├── api/
│   │   └── templates.ts          # API client with all operations
│   └── utils/
│       └── cron.ts               # Cron utilities and parsing
├── store/
│   └── templates-store.ts        # Zustand state management
├── components/
│   └── templates/
│       ├── TemplateCard.tsx      # Template display card
│       ├── TemplateExecutor.tsx  # Execution modal with params
│       ├── CronBuilder.tsx       # Visual cron expression builder
│       ├── ScheduleCreator.tsx   # Schedule creation modal
│       ├── ExecutionHistory.tsx  # Execution timeline viewer
│       └── index.ts              # Component exports
└── pages/
    ├── TemplatesPage.tsx         # Main templates library
    └── SchedulesPage.tsx         # Schedules management
```

## Usage Examples

### 1. Add Templates Page to App

```tsx
// app/templates/page.tsx
import { TemplatesPage } from '@/pages/TemplatesPage'

export default function Templates() {
  return <TemplatesPage />
}
```

### 2. Add Schedules Page to App

```tsx
// app/schedules/page.tsx
import { SchedulesPage } from '@/pages/SchedulesPage'

export default function Schedules() {
  return <SchedulesPage />
}
```

### 3. Using Templates Store

```tsx
import { useTemplatesStore } from '@/store/templates-store'

function MyComponent() {
  const {
    templates,
    loading,
    fetchTemplates,
    executeTemplate
  } = useTemplatesStore()

  useEffect(() => {
    fetchTemplates()
  }, [])

  const handleExecute = async (templateId: string) => {
    const result = await executeTemplate(templateId, {
      startDate: '2024-01-01',
      limit: 100
    })
    console.log(result)
  }

  return <div>...</div>
}
```

### 4. Standalone Template Executor

```tsx
import { TemplateExecutor } from '@/components/templates'

function QueryPage() {
  const [template, setTemplate] = useState<QueryTemplate | null>(null)

  return (
    <>
      <Button onClick={() => setTemplate(myTemplate)}>
        Execute Template
      </Button>

      {template && (
        <TemplateExecutor
          template={template}
          open={!!template}
          onClose={() => setTemplate(null)}
        />
      )}
    </>
  )
}
```

### 5. Inline Cron Builder

```tsx
import { CronBuilder } from '@/components/templates'

function ScheduleForm() {
  const [frequency, setFrequency] = useState('0 9 * * *')

  return (
    <CronBuilder
      value={frequency}
      onChange={setFrequency}
    />
  )
}
```

## API Integration

The implementation expects the following backend endpoints:

### Templates
- `GET /api/v1/templates` - List templates (with filters)
- `GET /api/v1/templates/:id` - Get template details
- `POST /api/v1/templates` - Create template
- `PUT /api/v1/templates/:id` - Update template
- `DELETE /api/v1/templates/:id` - Delete template
- `POST /api/v1/templates/:id/execute` - Execute with params
- `POST /api/v1/templates/:id/duplicate` - Duplicate template

### Schedules
- `GET /api/v1/schedules` - List schedules
- `GET /api/v1/schedules/:id` - Get schedule details
- `POST /api/v1/schedules` - Create schedule
- `PUT /api/v1/schedules/:id` - Update schedule
- `DELETE /api/v1/schedules/:id` - Delete schedule
- `POST /api/v1/schedules/:id/pause` - Pause schedule
- `POST /api/v1/schedules/:id/resume` - Resume schedule
- `POST /api/v1/schedules/:id/run` - Run immediately
- `GET /api/v1/schedules/:id/executions` - Get execution history

## Template Parameter System

Templates use `{{parameter}}` placeholders in SQL:

```sql
SELECT * FROM orders
WHERE created_at >= {{startDate}}
  AND status = {{status}}
  AND amount > {{minAmount}}
LIMIT {{limit}}
```

Parameter definitions:

```typescript
{
  name: "startDate",
  type: "date",
  required: true,
  description: "Filter orders from this date"
},
{
  name: "status",
  type: "string",
  required: false,
  default: "completed",
  validation: {
    options: ["pending", "completed", "cancelled"]
  }
},
{
  name: "minAmount",
  type: "number",
  required: false,
  default: 0,
  validation: {
    min: 0,
    max: 10000
  }
},
{
  name: "limit",
  type: "number",
  default: 100,
  validation: {
    min: 1,
    max: 1000
  }
}
```

## Cron Expression Examples

The CronBuilder supports these patterns:

- **Every hour**: `0 * * * *`
- **Daily at 9 AM**: `0 9 * * *`
- **Weekly on Monday**: `0 9 * * 1`
- **Monthly on 1st**: `0 9 1 * *`
- **Weekdays at 9 AM**: `0 9 * * 1-5`
- **Every 6 hours**: `0 */6 * * *`

Format: `minute hour dayOfMonth month dayOfWeek`

## Accessibility Features

- **Keyboard Navigation**: Full keyboard support for all interactions
- **Screen Reader Support**: ARIA labels and descriptions
- **Focus Management**: Proper focus trapping in modals
- **Color Contrast**: WCAG AA compliant colors
- **Error Messages**: Clear, actionable error states
- **Loading States**: Announced to screen readers

## Performance Optimizations

1. **Memoization**: Filtered templates and computed values
2. **Virtual Scrolling**: For large execution histories
3. **Lazy Loading**: Components loaded on-demand
4. **Debounced Search**: 300ms debounce on search input
5. **Optimistic Updates**: UI updates before API confirms
6. **Local Caching**: Templates cached in Zustand store

## Styling & Theming

Built with:
- **Tailwind CSS**: Utility-first styling
- **shadcn/ui**: Accessible component primitives
- **Dark Mode**: Full dark theme support
- **Responsive**: Mobile-first breakpoints (sm/md/lg/xl)

Custom theme variables in `globals.css`:
```css
--primary
--secondary
--accent
--muted
--destructive
```

## Testing Checklist

- [ ] Templates load and display correctly
- [ ] Search filters templates by name/description/SQL/tags
- [ ] Category tabs filter properly
- [ ] Tag chips toggle on/off
- [ ] Sort options work (usage/newest/name/updated)
- [ ] Template executor validates parameters
- [ ] SQL preview updates with parameter changes
- [ ] Execute button disabled when invalid
- [ ] Results display in table format
- [ ] Cron builder generates valid expressions
- [ ] Preset crons populate correctly
- [ ] Custom cron validation works
- [ ] Schedule creation succeeds
- [ ] Schedules list displays all schedules
- [ ] Pause/resume updates status
- [ ] Run now triggers execution
- [ ] Execution history loads and displays
- [ ] Timeline shows correct status badges
- [ ] Error messages display properly
- [ ] Mobile responsive layout works
- [ ] Dark mode looks correct
- [ ] Keyboard navigation works

## Dependencies

Required packages (already installed in most Next.js projects):

```json
{
  "dependencies": {
    "zustand": "^4.x",
    "date-fns": "^2.x",
    "lucide-react": "^0.x",
    "react-syntax-highlighter": "^15.x",
    "@radix-ui/react-*": "Various"
  }
}
```

## Future Enhancements

Potential additions:

1. **Template Sharing**: Share templates across organizations
2. **Version History**: Track template changes over time
3. **Template Folders**: Organize templates into folders
4. **Advanced Scheduling**: Dependencies between schedules
5. **Result Caching**: Cache execution results
6. **Export Templates**: Export/import template definitions
7. **Batch Execution**: Run multiple templates at once
8. **Query Optimization**: Suggest index improvements
9. **Cost Estimates**: Estimate query execution cost
10. **Collaborative Editing**: Real-time template editing

## Support

For issues or questions:
1. Check the implementation guide
2. Review type definitions in `types/templates.ts`
3. Test with mock data first
4. Verify API endpoints are correct
5. Check browser console for errors

## License

Part of SQL Studio - MIT License
