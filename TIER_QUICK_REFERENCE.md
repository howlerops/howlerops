# SQL Studio Tier System - Quick Reference

## 🎯 Quick Start

### 1. Initialize (in App.tsx)
```typescript
import { initializeTierStore } from '@/lib/tiers'

useEffect(() => {
  initializeTierStore()
}, [])
```

### 2. Check Features
```typescript
import { useFeatureGate } from '@/hooks/use-feature-gate'

const { allowed, showUpgrade } = useFeatureGate('sync')
```

### 3. Check Limits
```typescript
import { useTierLimit } from '@/hooks/use-tier-limit'

const limit = useTierLimit('connections', count)
// limit.allowed, limit.displayString, limit.percentage
```

### 4. Display Badge
```typescript
import { TierBadge } from '@/components/tier-badge'

<TierBadge variant="header" onClick={...} />
```

## 📊 Tier Limits

| Resource        | Local (Free) | Individual | Team      |
|----------------|--------------|------------|-----------|
| Connections    | 5            | ∞          | ∞         |
| Query History  | 50           | ∞          | ∞         |
| AI Memories    | 10           | ∞          | ∞         |
| Saved Queries  | 20           | ∞          | ∞         |
| Export Size    | 10 MB        | 100 MB     | 500 MB    |
| Team Members   | -            | -          | 5         |

## ✨ Features

| Feature           | Local | Individual | Team |
|-------------------|-------|------------|------|
| Cloud Sync        | ❌    | ✅         | ✅   |
| Multi-Device      | ❌    | ✅         | ✅   |
| AI Memory Sync    | ❌    | ✅         | ✅   |
| Team Sharing      | ❌    | ❌         | ✅   |
| RBAC              | ❌    | ❌         | ✅   |
| Audit Log         | ❌    | ❌         | ✅   |
| Priority Support  | ❌    | ✅         | ✅   |
| Custom Themes     | ❌    | ✅         | ✅   |

## 🔑 License Format

```
SQL-{TIER}-{UUID}-{CHECKSUM}
```

Example: `SQL-INDIVIDUAL-550e8400-e29b-41d4-a716-446655440000-A3F2`

## 📁 Key Files

### Import Path: `@/lib/tiers`
```typescript
import {
  // Store
  useTierStore,
  initializeTierStore,

  // Hooks
  useFeatureGate,
  useTierLimit,

  // Components
  TierBadge,

  // Utils
  validateLicenseKey,
  generateLicenseKey,

  // Config
  TIER_LIMITS,
  TIER_FEATURES,
} from '@/lib/tiers'
```

## 🎨 Component Variants

### TierBadge
```typescript
<TierBadge variant="header" />   // Compact, for toolbar
<TierBadge variant="inline" />   // Inline text badge
<TierBadge variant="card" />     // Detailed card
<TierBadge variant="minimal" />  // Text only
```

## 🪝 Hook Returns

### useFeatureGate
```typescript
{
  allowed: boolean           // Can use feature?
  tier: TierLevel           // Current tier
  requiredTier: TierLevel   // Required tier
  showUpgrade: () => void   // Show upgrade dialog
  isDevMode: boolean        // Dev mode active?
  isTeamFeature: boolean    // Team only?
  isIndividualFeature: boolean  // Individual+?
}
```

### useTierLimit
```typescript
{
  usage: number              // Current usage
  limit: number | null       // Max (null = unlimited)
  remaining: number | null   // Remaining capacity
  percentage: number         // Usage % (0-100)
  allowed: boolean           // Within limit?
  isNearLimit: boolean       // >= 80%
  isAtLimit: boolean         // >= 100%
  isUnlimited: boolean       // No limit
  displayString: string      // "3 / 5" or "3 / ∞"
  colorIndicator: string     // 'green' | 'yellow' | 'red'
  showUpgrade: () => void    // Show upgrade dialog
}
```

## 🎭 Common Patterns

### Guard Feature
```typescript
const { allowed, showUpgrade } = useFeatureGate('sync')

if (!allowed) {
  return <UpgradePrompt onClick={showUpgrade} />
}

return <SyncFeature />
```

### Show Limit Progress
```typescript
const limit = useTierLimit('connections', count)

<Progress
  value={limit.percentage}
  color={limit.colorIndicator}
/>
<Text>{limit.displayString}</Text>
```

### Pre-check Action
```typescript
const { allowed, showUpgrade } = useCanExceedLimit(
  'connections',
  currentCount,
  1
)

if (!allowed) {
  showUpgrade()
  return
}

addConnection(...)
```

### License Activation
```typescript
const { activateLicense } = useTierStore()

const result = await activateLicense(key)

if (result.success) {
  toast.success(`Welcome to ${result.tier}!`)
} else {
  toast.error(result.error)
}
```

## 🔧 Development

### Enable Dev Mode
```bash
# .env
VITE_TIER_DEV_MODE=true
```

Or:
```typescript
useTierStore.getState().enableDevMode()
```

### Test Licenses
```typescript
import { devLicenses } from '@/lib/tiers'

console.log(devLicenses.local)
console.log(devLicenses.individual)
console.log(devLicenses.team)
```

### Generate License
```typescript
import { generateLicenseKey } from '@/lib/tiers'

const key = generateLicenseKey('individual')
// SQL-INDIVIDUAL-{uuid}-{checksum}
```

## 🎪 Events

### Listen for Upgrade Requests
```typescript
useEffect(() => {
  const handler = (e: CustomEvent) => {
    const { feature, limit, currentTier } = e.detail
    showUpgradeModal(e.detail)
  }

  window.addEventListener('showUpgradeDialog', handler)
  return () => window.removeEventListener('showUpgradeDialog', handler)
}, [])
```

## 🧪 Testing

### Check Current Tier
```typescript
const { currentTier } = useTierStore()
// 'local' | 'individual' | 'team'
```

### Check Specific Tier
```typescript
const { isAtLeastTier } = useTierStore()

if (isAtLeastTier('individual')) {
  // Has Individual or Team tier
}
```

### Get All Features
```typescript
const { getFeatures } = useTierStore()
const features = getFeatures()
// { sync: true, multiDevice: true, ... }
```

### Get All Limits
```typescript
const { getLimits } = useTierStore()
const limits = getLimits()
// { connections: 5, queryHistory: 50, ... }
```

## 📝 Type Definitions

```typescript
type TierLevel = 'local' | 'individual' | 'team'

type FeatureName =
  | 'sync'
  | 'multiDevice'
  | 'teamSharing'
  | 'rbac'
  | 'auditLog'
  | 'prioritySupport'
  | 'customThemes'
  // ... etc

type LimitName =
  | 'connections'
  | 'queryHistory'
  | 'aiMemories'
  | 'savedQueries'
  | 'queryExecutions'
  | 'teamMembers'
```

## 🚀 Integration Checklist

1. ✅ Initialize tier store in app entry point
2. ✅ Add tier badge to header/toolbar
3. ✅ Create settings page with license activation
4. ⬜ Add upgrade dialog/modal
5. ⬜ Implement feature guards in components
6. ⬜ Add limit warnings in UI
7. ⬜ Set up server-side license validation
8. ⬜ Configure payment integration
9. ⬜ Add usage analytics
10. ⬜ Test all three tiers

## 📚 Documentation

- Full Docs: `/frontend/src/lib/tiers/README.md`
- Summary: `/TIER_SYSTEM_SUMMARY.md`
- This Reference: `/TIER_QUICK_REFERENCE.md`

## 🆘 Troubleshooting

### License Won't Activate
- Check format: `SQL-{TIER}-{UUID}-{CHECKSUM}`
- Verify UUID is RFC 4122 format
- Ensure checksum is correct (4 hex chars)
- Check network connectivity

### Limit Not Enforced
- Verify tier store is initialized
- Check dev mode isn't enabled
- Review limit configuration
- Check for race conditions

### Feature Not Available
- Verify feature name is correct
- Check tier has the feature
- Ensure license hasn't expired
- Check dev mode status

### Tier Store Not Persisting
- Check localStorage is available
- Verify no browser privacy modes
- Check for localStorage quota
- Review persistence middleware

---

**Quick Imports:**
```typescript
import { useTierStore, initializeTierStore } from '@/store/tier-store'
import { useFeatureGate } from '@/hooks/use-feature-gate'
import { useTierLimit } from '@/hooks/use-tier-limit'
import { TierBadge } from '@/components/tier-badge'
```

**Or use central export:**
```typescript
import {
  useTierStore,
  useFeatureGate,
  useTierLimit,
  TierBadge,
} from '@/lib/tiers'
```
