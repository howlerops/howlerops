# Query Templates - Quick Start Guide

Get up and running with the Templates feature in under 5 minutes!

## ⚡ 5-Minute Quick Start

### Step 1: Install Dependencies (30 seconds)

```bash
cd frontend
npm install date-fns
```

### Step 2: Add Routes (1 minute)

**Option A: React Router**
```tsx
// src/App.tsx
import { TemplatesPage } from './pages/TemplatesPage'
import { SchedulesPage } from './pages/SchedulesPage'

const routes = [
  { path: '/templates', element: <TemplatesPage /> },
  { path: '/schedules', element: <SchedulesPage /> },
]
```

**Option B: Next.js**
```tsx
// app/templates/page.tsx
import { TemplatesPage } from '@/pages/TemplatesPage'
export default function Templates() {
  return <TemplatesPage />
}
```

### Step 3: Configure API (30 seconds)

Create `.env` or `.env.local`:
```bash
NEXT_PUBLIC_API_URL=http://localhost:8080
```

### Step 4: Add Navigation (1 minute)

```tsx
import { Link } from 'react-router-dom'
import { FileCode, Calendar } from 'lucide-react'

<nav>
  <Link to="/templates">
    <FileCode /> Templates
  </Link>
  <Link to="/schedules">
    <Calendar /> Schedules
  </Link>
</nav>
```

### Step 5: Run and Test (2 minutes)

```bash
npm run dev
```

Visit:
- http://localhost:5173/templates
- http://localhost:5173/schedules

**Done!** 🎉

---

## 🧪 Testing Without Backend

Use mock data while developing:

```tsx
import { mockTemplates } from '@/lib/mocks/templates-mock-data'
import { useTemplatesStore } from '@/store/templates-store'

// In your component or test setup
useEffect(() => {
  useTemplatesStore.setState({
    templates: mockTemplates,
    loading: false,
  })
}, [])
```

---

## 🎯 Common Use Cases

### 1. Execute a Template

```tsx
import { useTemplatesStore } from '@/store/templates-store'

const { executeTemplate } = useTemplatesStore()

const result = await executeTemplate('template-id', {
  startDate: '2024-01-01',
  endDate: '2024-12-31',
})

console.log(result.rows)
```

### 2. Create a Schedule

```tsx
import { useTemplatesStore } from '@/store/templates-store'

const { createSchedule } = useTemplatesStore()

await createSchedule({
  template_id: 'template-id',
  name: 'Daily Report',
  frequency: '0 9 * * *', // Daily at 9 AM
  parameters: { startDate: '2024-01-01' },
  notification_email: 'team@company.com',
})
```

### 3. Search Templates

```tsx
import { useTemplatesStore } from '@/store/templates-store'

const { setFilters, getFilteredTemplates } = useTemplatesStore()

setFilters({
  search: 'sales',
  category: 'reporting',
  tags: ['revenue'],
})

const filtered = getFilteredTemplates()
```

---

## 📋 Checklist

Copy this checklist to track your integration:

```
Setup
□ Install date-fns dependency
□ Add routes for /templates and /schedules
□ Configure NEXT_PUBLIC_API_URL in .env
□ Add navigation links

Testing
□ Visit /templates page - should load
□ Search works
□ Category filters work
□ Click "Use Template" - modal opens
□ Execute a template - results display
□ Visit /schedules page - should load
□ Create a schedule - success
□ View execution history - timeline shows

Polish
□ Test on mobile device
□ Test in dark mode
□ Test keyboard navigation
□ Test with screen reader (optional)

Production
□ Backend API endpoints implemented
□ Environment variables set for production
□ Error tracking configured
□ Analytics added (optional)
```

---

## 🔧 Troubleshooting

### Templates not loading?
```tsx
// Check API configuration
console.log('API URL:', process.env.NEXT_PUBLIC_API_URL)

// Check network requests in DevTools
// Look for calls to /api/v1/templates
```

### Type errors?
```bash
# Run type check
npm run typecheck

# Common fix: restart TypeScript server in VS Code
# Cmd+Shift+P > "TypeScript: Restart TS Server"
```

### Styling looks broken?
```bash
# Ensure Tailwind is configured
# Check tailwind.config.js includes:
content: [
  "./src/**/*.{js,jsx,ts,tsx}",
]
```

---

## 📚 Next Steps

Once you have the basics working:

1. **Read the full guides:**
   - `TEMPLATES_README.md` - Complete reference
   - `TEMPLATES_IMPLEMENTATION.md` - Technical details
   - `TEMPLATES_VISUAL_GUIDE.md` - Design system

2. **Customize:**
   - Update category colors in `TemplateCard.tsx`
   - Add your own presets to `CronBuilder.tsx`
   - Customize validation rules in templates

3. **Extend:**
   - Add template folders/categories
   - Implement template versioning
   - Add export/import functionality
   - Create template marketplace

---

## 🆘 Need Help?

1. Check the documentation files
2. Review integration examples in `TEMPLATES_INTEGRATION_EXAMPLE.tsx`
3. Test with mock data first
4. Check browser console for errors
5. Verify API endpoints are correct

---

## 🎉 Success!

You should now have:
- ✅ Working template library
- ✅ Template execution with parameters
- ✅ Schedule management
- ✅ Execution history

**Happy querying!** 🚀
