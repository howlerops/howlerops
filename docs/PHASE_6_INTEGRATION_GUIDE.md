# Phase 6: Quick Integration Guide

This guide helps you integrate the onboarding and tutorial system into Howlerops.

## Step 1: Install Dependencies

```bash
cd frontend
npm install shepherd.js react-joyride react-confetti
```

## Step 2: Add Missing UI Component (Slider)

The video player uses a Slider component. If not already present, add it:

```bash
npx shadcn-ui@latest add slider
```

Or create manually:

```tsx
// frontend/src/components/ui/slider.tsx
import * as React from "react"
import * as SliderPrimitive from "@radix-ui/react-slider"
import { cn } from "@/lib/utils"

const Slider = React.forwardRef<
  React.ElementRef<typeof SliderPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof SliderPrimitive.Root>
>(({ className, ...props }, ref) => (
  <SliderPrimitive.Root
    ref={ref}
    className={cn(
      "relative flex w-full touch-none select-none items-center",
      className
    )}
    {...props}
  >
    <SliderPrimitive.Track className="relative h-2 w-full grow overflow-hidden rounded-full bg-secondary">
      <SliderPrimitive.Range className="absolute h-full bg-primary" />
    </SliderPrimitive.Track>
    <SliderPrimitive.Thumb className="block h-5 w-5 rounded-full border-2 border-primary bg-background ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50" />
  </SliderPrimitive.Root>
))
Slider.displayName = SliderPrimitive.Root.displayName

export { Slider }
```

## Step 3: Integrate Onboarding into App

### Option A: Add to Main App Component

```tsx
// frontend/src/App.tsx (or your main app file)
import { useState, useEffect } from "react"
import { OnboardingWizard } from "@/components/onboarding"
import { HelpWidget } from "@/components/docs"

function App() {
  const [showOnboarding, setShowOnboarding] = useState(false)

  useEffect(() => {
    // Check if user has completed onboarding
    const onboardingState = localStorage.getItem("sql-studio-onboarding")
    if (onboardingState) {
      const state = JSON.parse(onboardingState)
      setShowOnboarding(!state.isComplete)
    } else {
      // New user, show onboarding
      setShowOnboarding(true)
    }
  }, [])

  const handleOnboardingComplete = (selectedPath: string) => {
    console.log("User selected path:", selectedPath)

    // Navigate based on selected path
    switch (selectedPath) {
      case "templates":
        // navigate("/templates")
        break
      case "ai":
        // navigate("/ai-assistant")
        break
      case "team":
        // navigate("/organization/invite")
        break
      case "explore":
      default:
        // navigate("/")
        break
    }
  }

  return (
    <>
      {/* Your existing app */}
      <YourAppContent />

      {/* Onboarding wizard */}
      <OnboardingWizard
        open={showOnboarding}
        onOpenChange={setShowOnboarding}
        onComplete={handleOnboardingComplete}
      />

      {/* Help widget (floating button) */}
      <HelpWidget />
    </>
  )
}
```

### Option B: Add to Specific Route

```tsx
// For apps using React Router
import { Routes, Route } from "react-router-dom"
import { OnboardingWizard } from "@/components/onboarding"

function AppRoutes() {
  const [showOnboarding, setShowOnboarding] = useState(true)

  return (
    <>
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/editor" element={<QueryEditor />} />
        {/* ... other routes */}
      </Routes>

      <OnboardingWizard
        open={showOnboarding}
        onOpenChange={setShowOnboarding}
      />
    </>
  )
}
```

## Step 4: Add Tutorial to Query Editor

```tsx
// frontend/src/pages/QueryEditor.tsx
import { TutorialTrigger } from "@/components/tutorials"

export function QueryEditor() {
  return (
    <TutorialTrigger
      tutorialId="query-editor-basics"
      trigger="page_visit"
      maxTriggerCount={1}
      delay={2000}
    >
      <div className="flex h-screen">
        {/* Add data-tutorial attributes to enable highlighting */}
        <aside data-tutorial="sidebar">
          <Sidebar />
        </aside>

        <main data-tutorial="query-editor" className="flex-1">
          <div data-tutorial="query-input">
            <CodeEditor />
          </div>

          <button data-tutorial="run-button">
            Run Query
          </button>

          <div data-tutorial="query-results">
            <ResultsPanel />
          </div>
        </main>

        <aside data-tutorial="schema">
          <SchemaExplorer />
        </aside>
      </div>
    </TutorialTrigger>
  )
}
```

## Step 5: Add Empty States

```tsx
// frontend/src/components/SavedQueriesList.tsx
import { EmptyState, emptyStates } from "@/components/empty-states"
import { useNavigate } from "react-router-dom"

export function SavedQueriesList({ queries }) {
  const navigate = useNavigate()

  if (queries.length === 0) {
    return (
      <EmptyState
        {...emptyStates.noSavedQueries}
        primaryAction={{
          label: "Create Your First Query",
          onClick: () => navigate("/editor")
        }}
        secondaryAction={{
          label: "Learn About Saved Queries",
          onClick: () => {
            // Open tutorial or help
            window.open("/docs/saved-queries", "_blank")
          }
        }}
      />
    )
  }

  return (
    <div>
      {queries.map(query => (
        <QueryCard key={query.id} query={query} />
      ))}
    </div>
  )
}
```

## Step 6: Add Feature Tooltips for New Features

```tsx
// frontend/src/components/Navbar.tsx
import { FeatureTooltip } from "@/components/feature-discovery"

export function Navbar() {
  return (
    <nav>
      {/* Regular buttons */}
      <Button>Queries</Button>
      <Button>Connections</Button>

      {/* New feature with tooltip */}
      <FeatureTooltip
        feature="query-templates"
        title="New: Query Templates!"
        description="Create reusable query templates with parameters"
        ctaText="Try Templates"
        ctaLink="/templates"
        dismissible
      >
        <Button>
          <FileText className="mr-2" />
          Templates
          <Badge className="ml-2">New</Badge>
        </Button>
      </FeatureTooltip>
    </nav>
  )
}
```

## Step 7: Add Contextual Help

```tsx
// frontend/src/components/ScheduleQueryForm.tsx
import { QuickHelp } from "@/components/docs"

export function ScheduleQueryForm() {
  return (
    <form>
      <FormField>
        <Label className="flex items-center gap-2">
          Cron Expression
          <QuickHelp topic="cron-expressions" />
        </Label>
        <Input placeholder="0 9 * * *" />
      </FormField>

      {/* Other form fields */}
    </form>
  )
}
```

## Step 8: Configure Analytics (Optional)

Replace the console.log in `/frontend/src/lib/analytics/onboarding-tracking.ts`:

```typescript
// For PostHog
track(event: OnboardingEvent, properties: OnboardingEventProperties = {}) {
  const enrichedProperties = {
    ...properties,
    timestamp: new Date().toISOString(),
    session_id: this.sessionId,
  }

  if (window.posthog) {
    window.posthog.capture(event, enrichedProperties)
  }
}

// For Mixpanel
track(event: OnboardingEvent, properties: OnboardingEventProperties = {}) {
  const enrichedProperties = {
    ...properties,
    timestamp: new Date().toISOString(),
    session_id: this.sessionId,
  }

  if (window.mixpanel) {
    window.mixpanel.track(event, enrichedProperties)
  }
}

// For Google Analytics
track(event: OnboardingEvent, properties: OnboardingEventProperties = {}) {
  const enrichedProperties = {
    ...properties,
    timestamp: new Date().toISOString(),
    session_id: this.sessionId,
  }

  if (window.gtag) {
    window.gtag('event', event, enrichedProperties)
  }
}
```

## Step 9: Add Tutorial Library Page (Optional)

```tsx
// frontend/src/pages/TutorialsPage.tsx
import { useState } from "react"
import { TutorialLibrary, TutorialEngine } from "@/components/tutorials"
import { Tutorial } from "@/types/tutorial"

export function TutorialsPage() {
  const [activeTutorial, setActiveTutorial] = useState<Tutorial | null>(null)
  const [completedTutorials, setCompletedTutorials] = useState<string[]>([])

  return (
    <div className="container py-8">
      <TutorialLibrary
        onStartTutorial={(tutorial) => setActiveTutorial(tutorial)}
        completedTutorials={completedTutorials}
      />

      {activeTutorial && (
        <TutorialEngine
          tutorial={activeTutorial}
          open={activeTutorial !== null}
          onOpenChange={(open) => !open && setActiveTutorial(null)}
          onComplete={() => {
            setCompletedTutorials([...completedTutorials, activeTutorial.id])
            setActiveTutorial(null)
          }}
        />
      )}
    </div>
  )
}
```

## Step 10: Add Video Library Page (Optional)

```tsx
// frontend/src/pages/VideosPage.tsx
import { VideoLibrary } from "@/components/videos"

export function VideosPage() {
  return (
    <div className="container py-8">
      <VideoLibrary />
    </div>
  )
}
```

## Step 11: Add Interactive Examples Page (Optional)

```tsx
// frontend/src/pages/ExamplesPage.tsx
import { ExampleGallery } from "@/components/examples"

export function ExamplesPage() {
  return (
    <div className="container py-8">
      <ExampleGallery />
    </div>
  )
}
```

## Testing Checklist

After integration, test:

- [ ] Onboarding wizard appears for new users
- [ ] Onboarding can be skipped and resumed
- [ ] Onboarding tracks progress in localStorage
- [ ] Help widget appears (bottom-right)
- [ ] Help panel opens with `?` keyboard shortcut
- [ ] Tutorials highlight correct elements
- [ ] Tutorial navigation (next/back) works
- [ ] Empty states show when appropriate
- [ ] Feature tooltips appear and dismiss
- [ ] Quick help tooltips work
- [ ] Analytics events fire (check console)

## Troubleshooting

### Onboarding doesn't appear
- Check localStorage: `sql-studio-onboarding`
- Clear it to force show: `localStorage.removeItem("sql-studio-onboarding")`

### Tutorial highlighting doesn't work
- Verify `data-tutorial` attributes are on elements
- Check CSS selector in tutorial definition
- Ensure elements are visible when tutorial starts

### Help widget not showing
- Check z-index conflicts
- Verify HelpWidget is rendered
- Check console for errors

### Slider component missing
- Install: `npm install @radix-ui/react-slider`
- Create slider.tsx in ui components (see Step 2)

## Performance Tips

### Lazy Load Components

```tsx
import { lazy, Suspense } from "react"

const OnboardingWizard = lazy(() =>
  import("@/components/onboarding").then(m => ({
    default: m.OnboardingWizard
  }))
)

function App() {
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <OnboardingWizard />
    </Suspense>
  )
}
```

### Debounce Analytics

```typescript
// In onboarding-tracking.ts
private eventQueue: Array<{event: string, properties: any}> = []

track(event: OnboardingEvent, properties: OnboardingEventProperties = {}) {
  this.eventQueue.push({ event, properties })

  // Flush queue every 5 seconds
  if (!this.flushTimer) {
    this.flushTimer = setTimeout(() => this.flush(), 5000)
  }
}

private flush() {
  // Send batched events
  this.eventQueue.forEach(({ event, properties }) => {
    // Your analytics call
  })
  this.eventQueue = []
  this.flushTimer = null
}
```

## Need Help?

- Review `/docs/ONBOARDING_IMPLEMENTATION.md` for detailed documentation
- Check component source code for implementation details
- Review `/docs/user-guides/` for user-facing documentation
- Check example integrations in this guide

**Happy building! 🎉**
