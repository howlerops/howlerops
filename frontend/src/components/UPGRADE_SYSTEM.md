# SQL Studio Upgrade Prompt System

A comprehensive, user-friendly upgrade prompt system that uses **soft nudges** instead of hard blocks.

## Philosophy: Gentle Nudges, Not Roadblocks

### Key Principles
1. **Never block** - Users can always continue working
2. **Show value** - Explain benefits, not just limits
3. **Be periodic** - Don't nag on every action
4. **Be contextual** - Show prompts at natural moments
5. **Be dismissible** - Easy to close and continue

## Installation & Setup

### 1. Wrap Your App with UpgradeProvider

```typescript
// App.tsx
import { UpgradeProvider } from '@/components/upgrade-provider'

function App() {
  return (
    <UpgradeProvider>
      <YourApp />
    </UpgradeProvider>
  )
}
```

### 2. Initialize Stores on App Launch

```typescript
// main.tsx or App.tsx
import { initializeTierStore } from '@/store/tier-store'
import { initializeUpgradePromptStore } from '@/store/upgrade-prompt-store'

// On app startup
initializeTierStore()
initializeUpgradePromptStore()
```

## Components Overview

### Core Components

#### 1. **UpgradeModal**
Beautiful, contextual upgrade modal with plan comparison.

```typescript
import { useUpgradeModal } from '@/components/upgrade-modal'

function MyComponent() {
  const { showUpgradeModal, UpgradeModalComponent } = useUpgradeModal()

  return (
    <>
      <Button onClick={() => showUpgradeModal('connections')}>
        Upgrade
      </Button>
      {UpgradeModalComponent}
    </>
  )
}
```

**Trigger Types:**
- `connections` - Reached connection limit
- `queryHistory` - Query history filling up
- `multiDevice` - New device detected
- `aiMemory` - AI memory limit
- `export` - Large export file
- `manual` - User clicked upgrade
- `periodic` - Periodic reminder
- `feature` - Locked feature clicked

#### 2. **Value Indicators**

Show usage status inline:

```typescript
import {
  ConnectionLimitIndicator,
  QueryHistoryIndicator,
  MultiDeviceBanner,
} from '@/components/value-indicators'

// In header
<ConnectionLimitIndicator variant="badge" />

// In query history panel
<QueryHistoryIndicator variant="banner" showThreshold={80} />

// At top of app (automatically shown)
<MultiDeviceBanner position="top" />
```

#### 3. **Soft Limits**

Non-blocking limit notifications:

```typescript
import { showSoftLimitToast, SoftLimitBanner } from '@/components/soft-limits'

// Toast notification
showSoftLimitToast({
  limitType: 'connections',
  usage: 6,
  softLimit: 5,
  onUpgrade: () => showUpgradeModal('connections'),
})

// Persistent banner
<SoftLimitBanner
  trigger="queryHistory"
  usage={52}
  softLimit={50}
  title="Query history limit reached"
  message="Upgrade for unlimited history"
  dismissible
  storageKey="query-banner-dismissed"
/>
```

#### 4. **Usage Stats**

Dashboard showing all usage metrics:

```typescript
import { UsageStats } from '@/components/usage-stats'

// In settings panel
<UsageStats showUpgradeCTA />

// Compact mode
<UsageStats compact />
```

#### 5. **Upgrade Flow**

Complete upgrade journey:

```typescript
import { PlanSelection, UpgradeSuccess } from '@/components/upgrade-flow'

// Plan selection
<PlanSelection
  selectedPlan={plan}
  onPlanChange={setPlan}
  billingPeriod={billing}
  onBillingPeriodChange={setBilling}
  onContinue={handleContinue}
/>

// Success screen
<UpgradeSuccess
  plan="individual"
  isTrial
  trialEndDate={trialEnd}
  onGetStarted={handleGetStarted}
/>
```

## Integration Examples

### Example 1: Connection Limit Check

```typescript
// connection-manager.tsx
import { useConnectionStore } from '@/store/connection-store'
import { useTierStore } from '@/store/tier-store'
import { useUpgrade } from '@/components/upgrade-provider'
import { showSoftLimitToast } from '@/components/soft-limits'

function ConnectionManager() {
  const { connections, addConnection } = useConnectionStore()
  const { checkLimit } = useTierStore()
  const { showUpgrade } = useUpgrade()

  const handleAddConnection = async (conn: Connection) => {
    const limitCheck = checkLimit('connections', connections.length + 1)

    if (!limitCheck.allowed) {
      // Show soft limit toast (non-blocking)
      showSoftLimitToast({
        limitType: 'connections',
        usage: connections.length,
        softLimit: limitCheck.limit || 5,
        onUpgrade: () => showUpgrade('connections'),
      })

      // Still allow the action (soft limit)
      // Or block it if you want a hard limit:
      // return
    }

    try {
      await addConnection(conn)
    } catch (error) {
      // Handle error
    }
  }

  return (
    <div>
      <ConnectionLimitIndicator variant="badge" />
      {/* Rest of component */}
    </div>
  )
}
```

### Example 2: Query History Indicator

```typescript
// query-history-panel.tsx
import { QueryHistoryIndicator } from '@/components/value-indicators'

function QueryHistoryPanel() {
  return (
    <div>
      <div className="header">
        <h2>Query History</h2>
        <QueryHistoryIndicator variant="inline" showUpgradeCTA />
      </div>

      {/* Show banner when approaching limit */}
      <QueryHistoryIndicator
        variant="banner"
        showThreshold={80}
      />

      {/* History list */}
    </div>
  )
}
```

### Example 3: Tracking Query Execution

```typescript
// query-executor.ts
import { trackQueryExecution } from '@/lib/upgrade-reminders'

export async function executeQuery(query: string) {
  // Execute query
  const result = await api.executeQuery(query)

  // Track for activity monitoring
  trackQueryExecution()

  return result
}
```

### Example 4: Settings Panel with Usage Stats

```typescript
// settings.tsx
import { UsageStats } from '@/components/usage-stats'
import { TierSettingsPanel } from '@/components/tier-settings-panel'

function Settings() {
  return (
    <div className="space-y-6">
      <TierSettingsPanel />
      <UsageStats showUpgradeCTA />
    </div>
  )
}
```

### Example 5: Locked Feature Preview

```typescript
// feature-component.tsx
import { useTierStore } from '@/store/tier-store'
import { useUpgrade } from '@/components/upgrade-provider'
import { Lock } from 'lucide-react'

function PremiumFeature() {
  const { hasFeature } = useTierStore()
  const { showUpgrade } = useUpgrade()

  const canUseFeature = hasFeature('teamSharing')

  if (!canUseFeature) {
    return (
      <div
        className="relative cursor-pointer opacity-60"
        onClick={() => showUpgrade('feature')}
      >
        <div className="absolute inset-0 flex items-center justify-center z-10 bg-black/20 backdrop-blur-sm rounded-lg">
          <div className="text-center">
            <Lock className="w-8 h-8 text-white mx-auto mb-2" />
            <p className="text-white font-semibold">Team Plan Required</p>
          </div>
        </div>
        {/* Feature preview (grayed out) */}
        <div className="pointer-events-none">
          <FeaturePreview />
        </div>
      </div>
    )
  }

  return <FullFeature />
}
```

## User Experience Flow

### First Time Soft Limit Hit
1. User reaches limit (e.g., 5 connections)
2. Gentle toast notification appears
3. Action is still allowed (soft limit)
4. Toast shows upgrade value

### Second Time
1. Persistent banner in relevant view
2. Shows usage progress bar
3. "Start Free Trial" CTA
4. Dismissible for 24 hours

### Third Time
1. Small badge reminder in UI
2. Non-intrusive
3. Click to see upgrade benefits

### Periodic Reminders
1. After 7 days for active users
2. Only at natural pauses
3. Never during critical workflows
4. Friday afternoon (good upgrade time!)

### New Device Detection
1. Detects new device fingerprint
2. Shows multi-device banner
3. Friendly "Welcome back!" message
4. Sync workspace CTA

## Customization

### Cooldown Periods

Adjust in `upgrade-prompt-store.ts`:

```typescript
export const COOLDOWN_PERIODS: Record<UpgradeTrigger, number> = {
  connections: 24 * 60 * 60 * 1000,      // 24 hours
  queryHistory: 7 * 24 * 60 * 60 * 1000, // 7 days
  multiDevice: 30 * 24 * 60 * 60 * 1000, // 30 days
  // ... customize as needed
}
```

### Dismissal Durations

```typescript
export const DISMISSAL_DURATIONS = {
  short: 24,        // 1 day
  medium: 24 * 7,   // 1 week
  long: 24 * 30,    // 1 month
  permanent: 24 * 365, // 1 year
}
```

### Reminder Configuration

In `upgrade-reminders.ts`:

```typescript
const DEFAULT_CONFIG: ReminderConfig = {
  minDaysBetween: 7,
  minQueriesInSession: 10,
  preferredDays: [5], // Friday
  preferredHours: [14, 18], // 2 PM - 6 PM
  minDaysSinceFirstLaunch: 7,
}
```

## Analytics & Metrics

Track upgrade prompt effectiveness:

```typescript
import { useUpgradePromptStore } from '@/store/upgrade-prompt-store'

function Analytics() {
  const { getMetrics } = useUpgradePromptStore()
  const metrics = getMetrics()

  console.log('Conversion Rate:', metrics.conversionRate)
  console.log('Dismiss Rate:', metrics.dismissRate)
  console.log('Total Shown:', metrics.totalShown)
  console.log('Total Upgrades:', metrics.totalUpgrades)
}
```

## Testing

### Test Prompts Locally

```typescript
// In browser console
import { useUpgradePromptStore } from '@/store/upgrade-prompt-store'

const store = useUpgradePromptStore.getState()

// Reset all prompt history
store.resetHistory()

// Clear dismissals
store.clearAllDismissals()

// Manually show prompt
store.shouldShowPrompt('connections') // true
```

### Test Different Triggers

```typescript
import { useUpgradeModal } from '@/components/upgrade-modal'

function TestComponent() {
  const { showUpgradeModal } = useUpgradeModal()

  return (
    <div>
      <Button onClick={() => showUpgradeModal('connections')}>
        Test Connections
      </Button>
      <Button onClick={() => showUpgradeModal('multiDevice')}>
        Test Multi-Device
      </Button>
      <Button onClick={() => showUpgradeModal('periodic')}>
        Test Periodic
      </Button>
    </div>
  )
}
```

## Best Practices

### DO:
- Show prompts at natural pauses (after query completes, app launch)
- Use contextual messaging (explain WHY they need to upgrade)
- Always make prompts dismissible
- Track metrics to optimize conversion
- Use soft limits (allow action, show value)
- Respect user's workflow

### DON'T:
- Block users from working
- Show prompts during critical tasks (mid-edit, mid-query)
- Nag repeatedly if dismissed
- Use generic "Upgrade now!" messages
- Force upgrades for basic features
- Interrupt user's flow

## Design System

### Colors
- Local tier: Gray (#64748B)
- Individual tier: Blue (#3B82F6)
- Team tier: Purple (#A855F7)
- Warning: Orange (#F97316)
- Success: Green (#10B981)

### Icons
- Sparkles: Upgrade/premium
- Cloud: Sync
- Users: Team
- Lock: Locked features
- TrendingUp: Growth
- Zap: Fast/powerful

## Future Enhancements

- [ ] Stripe integration for payment
- [ ] OAuth login (Google, GitHub)
- [ ] Email verification
- [ ] Team member invitations
- [ ] Usage charts and trends
- [ ] A/B testing different messages
- [ ] Smart ML-based prompt timing
- [ ] In-app guided tours for new features

## Support

Questions? Check the [main README](../../../README.md) or open an issue on GitHub.
