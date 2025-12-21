# Upgrade System Quick Reference Card

## 30-Second Setup

```typescript
// 1. Wrap your app
import { UpgradeProvider } from '@/components/upgrade-provider'

<UpgradeProvider>
  <App />
</UpgradeProvider>

// 2. Initialize stores
import { initializeTierStore } from '@/store/tier-store'
import { initializeUpgradePromptStore } from '@/store/upgrade-prompt-store'

initializeTierStore()
initializeUpgradePromptStore()

// 3. Use in components
import { useUpgrade } from '@/components/upgrade-provider'

const { showUpgrade } = useUpgrade()
showUpgrade('connections') // Show modal
```

## Common Patterns

### Show Upgrade Modal
```typescript
import { useUpgrade } from '@/components/upgrade-provider'

const { showUpgrade } = useUpgrade()
showUpgrade('connections') // or 'queryHistory', 'multiDevice', etc.
```

### Check Limits
```typescript
import { useTierStore } from '@/store/tier-store'

const { checkLimit } = useTierStore()
const result = checkLimit('connections', currentCount)

if (!result.allowed) {
  // Show soft limit notification
}
```

### Soft Limit Toast
```typescript
import { showSoftLimitToast } from '@/components/soft-limits'

showSoftLimitToast({
  limitType: 'connections',
  usage: 6,
  softLimit: 5,
  onUpgrade: () => showUpgrade('connections'),
})
```

### Value Indicators
```typescript
import {
  ConnectionLimitIndicator,
  QueryHistoryIndicator,
} from '@/components/value-indicators'

// Header badge
<ConnectionLimitIndicator variant="badge" />

// Query panel banner
<QueryHistoryIndicator variant="banner" showThreshold={80} />
```

### Track Activity
```typescript
import { trackQueryExecution } from '@/lib/upgrade-reminders'

// After each query
trackQueryExecution()
```

## All Triggers

| Trigger | Use Case | Cooldown |
|---------|----------|----------|
| `connections` | Reached 5 connections | 24 hours |
| `queryHistory` | Query history filling up | 7 days |
| `multiDevice` | New device detected | 30 days |
| `aiMemory` | AI memory limit | 7 days |
| `export` | Large export file | 24 hours |
| `manual` | User clicked upgrade | None |
| `periodic` | Active user reminder | 7 days |
| `feature` | Locked feature clicked | 24 hours |

## All Components

### Core
- `UpgradeProvider` - Global provider
- `UpgradeModal` - Main upgrade modal
- `UsageStats` - Usage dashboard

### Indicators
- `ConnectionLimitIndicator` - Connection usage
- `QueryHistoryIndicator` - Query history usage
- `MultiDeviceBanner` - New device banner

### Soft Limits
- `showSoftLimitToast()` - Toast notification
- `SoftLimitBanner` - Persistent banner

### Upgrade Flow
- `PlanSelection` - Choose plan
- `UpgradeSuccess` - Success screen

## File Locations

```
frontend/src/
├── store/
│   └── upgrade-prompt-store.ts          # Prompt tracking
├── components/
│   ├── upgrade-modal.tsx                # Main modal
│   ├── upgrade-provider.tsx             # Global provider
│   ├── usage-stats.tsx                  # Usage dashboard
│   ├── value-indicators/                # Usage indicators
│   │   ├── connection-limit-indicator.tsx
│   │   ├── query-history-indicator.tsx
│   │   └── multi-device-banner.tsx
│   ├── soft-limits/                     # Soft limit UI
│   │   ├── soft-limit-toast.tsx
│   │   └── soft-limit-banner.tsx
│   └── upgrade-flow/                    # Upgrade journey
│       ├── plan-selection.tsx
│       └── upgrade-success.tsx
├── lib/
│   └── upgrade-reminders.ts             # Smart reminders
└── examples/
    └── upgrade-integration-example.tsx  # Code examples
```

## Variants

### ConnectionLimitIndicator
- `badge` - Small header badge
- `inline` - Text with icon
- `full` - Full card with progress

### QueryHistoryIndicator
- `badge` - Small badge
- `banner` - Full banner with progress
- `inline` - Inline text

## Colors

- Local: Gray `#64748B`
- Individual: Blue `#3B82F6`
- Team: Purple `#A855F7`
- Warning: Orange `#F97316`
- Success: Green `#10B981`

## Configuration

### Change Cooldowns
`upgrade-prompt-store.ts`:
```typescript
export const COOLDOWN_PERIODS = {
  connections: 24 * 60 * 60 * 1000, // 24h
  queryHistory: 7 * 24 * 60 * 60 * 1000, // 7d
}
```

### Change Reminder Settings
`upgrade-reminders.ts`:
```typescript
const DEFAULT_CONFIG = {
  minDaysBetween: 7,
  minQueriesInSession: 10,
  preferredDays: [5], // Friday
}
```

## Metrics

```typescript
const { getMetrics } = useUpgradePromptStore()
const {
  totalShown,
  totalDismissed,
  totalUpgrades,
  conversionRate,
  dismissRate,
} = getMetrics()
```

## Testing

```typescript
// Reset history
useUpgradePromptStore.getState().resetHistory()

// Clear dismissals
useUpgradePromptStore.getState().clearAllDismissals()

// Force show
const { shouldShowPrompt } = useUpgradePromptStore()
shouldShowPrompt('connections') // returns boolean
```

## Full Documentation

- Integration Guide: `/frontend/src/components/UPGRADE_SYSTEM.md`
- Examples: `/frontend/src/examples/upgrade-integration-example.tsx`
- Summary: `/frontend/UPGRADE_SYSTEM_SUMMARY.md`

---

**Remember:** Soft nudges, not roadblocks. Users can always continue working!
