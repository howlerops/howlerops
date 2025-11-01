/**
 * Complete Integration Example
 * This file shows how to integrate the Templates feature into your app
 */

// ============================================================================
// 1. ROUTER CONFIGURATION
// ============================================================================

// For React Router (Vite/React)
import { createBrowserRouter } from 'react-router-dom'
import { TemplatesPage } from './pages/TemplatesPage'
import { SchedulesPage } from './pages/SchedulesPage'

// This is an example file that intentionally exports multiple examples
// eslint-disable-next-line react-refresh/only-export-components
export const exampleRouter = createBrowserRouter([
  {
    path: '/',
    element: <Layout />,
    children: [
      {
        path: 'templates',
        element: <TemplatesPage />,
      },
      {
        path: 'schedules',
        element: <SchedulesPage />,
      },
    ],
  },
])

// ============================================================================
// 2. NAVIGATION MENU
// ============================================================================

import { Link } from 'react-router-dom'
import { FileCode, Calendar } from 'lucide-react'

function AppNavigation() {
  return (
    <nav className="flex gap-4">
      <Link
        to="/templates"
        className="flex items-center gap-2 px-4 py-2 rounded-lg hover:bg-accent"
      >
        <FileCode className="h-5 w-5" />
        Templates
      </Link>
      <Link
        to="/schedules"
        className="flex items-center gap-2 px-4 py-2 rounded-lg hover:bg-accent"
      >
        <Calendar className="h-5 w-5" />
        Schedules
      </Link>
    </nav>
  )
}

// ============================================================================
// 3. USING TEMPLATES IN YOUR COMPONENTS
// ============================================================================

import { useTemplatesStore } from '@/store/templates-store'
import { TemplateExecutor } from '@/components/templates'
import { useState, useEffect } from 'react'

function QueryEditorWithTemplates() {
  const { templates, fetchTemplates, loading } = useTemplatesStore()
  const [selectedTemplate, setSelectedTemplate] = useState(null)

  useEffect(() => {
    fetchTemplates()
  }, [fetchTemplates])

  return (
    <div>
      <h2>Available Templates</h2>

      {loading ? (
        <div>Loading templates...</div>
      ) : (
        <div className="grid gap-4">
          {templates.map((template) => (
            <div
              key={template.id}
              className="p-4 border rounded-lg cursor-pointer hover:shadow"
              onClick={() => setSelectedTemplate(template)}
            >
              <h3>{template.name}</h3>
              <p className="text-sm text-muted-foreground">
                {template.description}
              </p>
            </div>
          ))}
        </div>
      )}

      {/* Template Executor Modal */}
      {selectedTemplate && (
        <TemplateExecutor
          template={selectedTemplate}
          open={!!selectedTemplate}
          onClose={() => setSelectedTemplate(null)}
        />
      )}
    </div>
  )
}

// ============================================================================
// 4. CREATING A TEMPLATE PROGRAMMATICALLY
// ============================================================================

function CreateTemplateExample() {
  const { createTemplate } = useTemplatesStore()

  const handleCreateTemplate = async () => {
    try {
      const newTemplate = await createTemplate({
        name: 'Customer Revenue Report',
        description: 'Calculate total revenue per customer',
        sql_template: `
          SELECT
            customer_id,
            customer_name,
            SUM(order_total) as total_revenue,
            COUNT(*) as order_count,
            AVG(order_total) as avg_order_value
          FROM orders
          WHERE order_date >= {{startDate}}
            AND order_date <= {{endDate}}
          GROUP BY customer_id, customer_name
          ORDER BY total_revenue DESC
          LIMIT {{limit}}
        `,
        parameters: [
          {
            name: 'startDate',
            type: 'date',
            required: true,
            description: 'Start date for revenue calculation',
          },
          {
            name: 'endDate',
            type: 'date',
            required: true,
            description: 'End date for revenue calculation',
          },
          {
            name: 'limit',
            type: 'number',
            required: false,
            default: 50,
            description: 'Number of customers to return',
            validation: { min: 1, max: 500 },
          },
        ],
        tags: ['revenue', 'customers', 'reporting'],
        category: 'reporting',
        is_public: true,
      })

      console.log('Template created:', newTemplate)
      alert('Template created successfully!')
    } catch (error) {
      console.error('Failed to create template:', error)
      alert('Error creating template')
    }
  }

  return (
    <button onClick={handleCreateTemplate}>
      Create Revenue Report Template
    </button>
  )
}

// ============================================================================
// 5. EXECUTING A TEMPLATE
// ============================================================================

function ExecuteTemplateExample() {
  const { executeTemplate } = useTemplatesStore()
  const [result, setResult] = useState(null)

  const handleExecute = async () => {
    try {
      const queryResult = await executeTemplate('template-id-here', {
        startDate: '2024-01-01',
        endDate: '2024-12-31',
        limit: 100,
      })

      setResult(queryResult)
      console.log('Query results:', queryResult)
    } catch (error) {
      console.error('Execution failed:', error)
    }
  }

  return (
    <div>
      <button onClick={handleExecute}>Execute Template</button>

      {result && (
        <div>
          <h3>Results ({result.rowCount} rows in {result.executionTime}ms)</h3>
          <table>
            <thead>
              <tr>
                {result.columns.map((col) => (
                  <th key={col}>{col}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {result.rows.map((row, i) => (
                <tr key={i}>
                  {result.columns.map((col) => (
                    <td key={col}>{row[col]}</td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

// ============================================================================
// 6. CREATING A SCHEDULE
// ============================================================================

function CreateScheduleExample() {
  const { createSchedule } = useTemplatesStore()

  const handleCreateSchedule = async () => {
    try {
      const schedule = await createSchedule({
        template_id: 'template-id-here',
        name: 'Daily Revenue Report',
        frequency: '0 9 * * *', // Daily at 9 AM
        parameters: {
          startDate: '2024-01-01',
          endDate: '2024-12-31',
          limit: 50,
        },
        notification_email: 'reports@company.com',
        result_storage: 'database',
      })

      console.log('Schedule created:', schedule)
      alert('Schedule created successfully!')
    } catch (error) {
      console.error('Failed to create schedule:', error)
    }
  }

  return (
    <button onClick={handleCreateSchedule}>
      Create Daily Schedule
    </button>
  )
}

// ============================================================================
// 7. USING CRON BUILDER STANDALONE
// ============================================================================

import { CronBuilder } from '@/components/templates'

function ScheduleFormExample() {
  const [frequency, setFrequency] = useState('0 9 * * *')

  const handleSubmit = () => {
    console.log('Schedule frequency:', frequency)
    // Use frequency in schedule creation
  }

  return (
    <form onSubmit={handleSubmit}>
      <h3>Configure Schedule</h3>

      <CronBuilder value={frequency} onChange={setFrequency} />

      <button type="submit">Save Schedule</button>
    </form>
  )
}

// ============================================================================
// 8. FILTERING AND SEARCHING TEMPLATES
// ============================================================================

function TemplateSearchExample() {
  const {
    _templates,
    setFilters,
    setSortBy,
    getFilteredTemplates,
  } = useTemplatesStore()

  const [searchQuery, setSearchQuery] = useState('')
  const [category, setCategory] = useState('all')

  useEffect(() => {
    setFilters({
      search: searchQuery,
      category: category === 'all' ? undefined : category,
    })
  }, [searchQuery, category, setFilters])

  const filteredTemplates = getFilteredTemplates()

  return (
    <div>
      {/* Search Input */}
      <input
        type="text"
        placeholder="Search templates..."
        value={searchQuery}
        onChange={(e) => setSearchQuery(e.target.value)}
      />

      {/* Category Filter */}
      <select value={category} onChange={(e) => setCategory(e.target.value)}>
        <option value="all">All Categories</option>
        <option value="reporting">Reporting</option>
        <option value="analytics">Analytics</option>
        <option value="maintenance">Maintenance</option>
        <option value="custom">Custom</option>
      </select>

      {/* Sort Options */}
      <select onChange={(e) => setSortBy(e.target.value)}>
        <option value="usage">Most Used</option>
        <option value="newest">Newest</option>
        <option value="name">Name (A-Z)</option>
        <option value="updated">Recently Updated</option>
      </select>

      {/* Results */}
      <div>
        {filteredTemplates.length} templates found
        {filteredTemplates.map((template) => (
          <div key={template.id}>{template.name}</div>
        ))}
      </div>
    </div>
  )
}

// ============================================================================
// 9. VIEWING EXECUTION HISTORY
// ============================================================================

import { ExecutionHistory } from '@/components/templates'

function ScheduleDetailExample() {
  const [viewingHistory, setViewingHistory] = useState(false)
  const scheduleId = 'schedule-id-here'

  return (
    <div>
      <button onClick={() => setViewingHistory(true)}>
        View Execution History
      </button>

      <ExecutionHistory
        scheduleId={scheduleId}
        open={viewingHistory}
        onClose={() => setViewingHistory(false)}
      />
    </div>
  )
}

// ============================================================================
// 10. COMPLETE DASHBOARD EXAMPLE
// ============================================================================

function TemplatesDashboard() {
  const {
    templates,
    schedules,
    loading,
    error,
    fetchTemplates,
    fetchSchedules,
  } = useTemplatesStore()

  useEffect(() => {
    fetchTemplates()
    fetchSchedules()
  }, [fetchTemplates, fetchSchedules])

  if (loading) return <div>Loading...</div>
  if (error) return <div>Error: {error}</div>

  const stats = {
    totalTemplates: templates.length,
    publicTemplates: templates.filter((t) => t.is_public).length,
    activeSchedules: schedules.filter((s) => s.status === 'active').length,
    totalExecutions: schedules.reduce((sum, _s) => {
      // Sum up executions (would come from API)
      return sum
    }, 0),
  }

  return (
    <div className="p-6">
      {/* Stats Cards */}
      <div className="grid grid-cols-4 gap-4 mb-6">
        <StatCard label="Total Templates" value={stats.totalTemplates} />
        <StatCard label="Public Templates" value={stats.publicTemplates} />
        <StatCard label="Active Schedules" value={stats.activeSchedules} />
        <StatCard label="Total Executions" value={stats.totalExecutions} />
      </div>

      {/* Quick Actions */}
      <div className="flex gap-4 mb-6">
        <Link to="/templates">
          <button>Browse Templates</button>
        </Link>
        <Link to="/schedules">
          <button>Manage Schedules</button>
        </Link>
      </div>

      {/* Recent Templates */}
      <div>
        <h3>Recent Templates</h3>
        <div className="grid grid-cols-3 gap-4">
          {templates.slice(0, 6).map((template) => (
            <TemplateCard key={template.id} template={template} />
          ))}
        </div>
      </div>
    </div>
  )
}

function StatCard({ label, value }) {
  return (
    <div className="border rounded-lg p-4">
      <div className="text-sm text-muted-foreground">{label}</div>
      <div className="text-3xl font-bold mt-2">{value}</div>
    </div>
  )
}

// ============================================================================
// 11. TESTING WITH MOCK DATA
// ============================================================================

import { mockTemplates, mockSchedules, mockExecutions } from '@/lib/mocks/templates-mock-data'

function TestingExample() {
  // Use mock data in development
  useEffect(() => {
    if (process.env.NODE_ENV === 'development') {
      // Initialize store with mock data
      useTemplatesStore.setState({
        templates: mockTemplates,
        schedules: mockSchedules,
        executions: new Map([
          ['sch-1', mockExecutions.filter(e => e.schedule_id === 'sch-1')],
        ]),
      })
    }
  }, [])

  return <TemplatesPage />
}

// ============================================================================
// 12. ERROR HANDLING EXAMPLE
// ============================================================================

function ErrorHandlingExample() {
  const { executeTemplate, error, clearError } = useTemplatesStore()
  const [localError, setLocalError] = useState(null)

  const handleExecute = async () => {
    try {
      clearError()
      setLocalError(null)

      const result = await executeTemplate('template-id', {
        startDate: '2024-01-01',
      })

      console.log('Success:', result)
    } catch (err) {
      setLocalError(err.message)
      console.error('Execution failed:', err)
    }
  }

  return (
    <div>
      <button onClick={handleExecute}>Execute</button>

      {(error || localError) && (
        <div className="bg-red-100 text-red-700 p-4 rounded">
          {error || localError}
        </div>
      )}
    </div>
  )
}

// ============================================================================
// EXPORT ALL EXAMPLES
// ============================================================================

export {
  AppNavigation,
  QueryEditorWithTemplates,
  CreateTemplateExample,
  ExecuteTemplateExample,
  CreateScheduleExample,
  ScheduleFormExample,
  TemplateSearchExample,
  ScheduleDetailExample,
  TemplatesDashboard,
  TestingExample,
  ErrorHandlingExample,
}
