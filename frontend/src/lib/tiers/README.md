# Howlerops Tier Management System

A comprehensive tier detection and management system for Howlerops's 3-tier product structure. This system provides type-safe tier management, license validation, feature gating, and usage limit enforcement.

## Overview

Howlerops offers three tiers:

- **Local (Free)**: 5 connections, 50 query history, no cloud sync
- **Individual ($9/mo)**: Unlimited resources, cloud sync, multi-device support
- **Team ($29/mo)**: All Individual features plus team collaboration, RBAC, and audit logging

## Architecture

### Core Components

1. **Types** (`/types/tiers.ts`)
   - TypeScript interfaces for tiers, features, limits, and licenses
   - Type-safe tier system with compile-time checking

2. **Configuration** (`/config/tier-limits.ts`)
   - Centralized tier limits and feature definitions
   - Easy tier comparison and validation utilities

3. **Store** (`/store/tier-store.ts`)
   - Zustand store with localStorage persistence
   - License activation and validation
   - Feature and limit checking

4. **License Validator** (`/lib/tiers/license-validator.ts`)
   - License key format: `SQL-{TIER}-{UUID}-{CHECKSUM}`
   - CRC16 checksum validation
   - Server-side validation support (optional)

5. **React Hooks**
   - `useFeatureGate`: Feature availability checking
   - `useTierLimit`: Usage limit monitoring

6. **UI Components** (`/components/tier-badge.tsx`)
   - Visual tier indicators
   - Multiple display variants

## Quick Start

### 1. Initialize the Tier Store

In your main app entry point:

```typescript
import { initializeTierStore } from '@/lib/tiers'

function App() {
  useEffect(() => {
    initializeTierStore()
  }, [])

  return <YourApp />
}
```

### 2. Check Feature Availability

```typescript
import { useFeatureGate } from '@/hooks/use-feature-gate'

function CloudSyncButton() {
  const { allowed, showUpgrade } = useFeatureGate('sync')

  if (!allowed) {
    return (
      <button onClick={showUpgrade}>
        Upgrade to enable Cloud Sync
      </button>
    )
  }

  return <button onClick={handleSync}>Sync to Cloud</button>
}
```

### 3. Monitor Usage Limits

```typescript
import { useTierLimit } from '@/hooks/use-tier-limit'

function ConnectionList() {
  const connections = useConnections()
  const limit = useTierLimit('connections', connections.length)

  return (
    <div>
      <h3>Connections: {limit.displayString}</h3>
      <Progress
        value={limit.percentage}
        color={limit.colorIndicator}
      />

      {limit.isNearLimit && (
        <Alert>You're approaching your connection limit!</Alert>
      )}

      {!limit.allowed && (
        <button onClick={limit.showUpgrade}>
          Upgrade for more connections
        </button>
      )}
    </div>
  )
}
```

### 4. Display Tier Badge

```typescript
import { TierBadge } from '@/components/tier-badge'

// In header
<TierBadge
  variant="header"
  onClick={() => navigate('/settings?tab=tier')}
/>

// In settings
<TierBadge variant="card" showExpiration />
```

## Integration Examples

### Existing Store Integration

#### Connection Store

The connection store automatically enforces connection limits:

```typescript
// In connection-store.ts
addConnection: (connectionData) => {
  const currentConnections = get().connections.length
  const tierStore = useTierStore.getState()
  const limitCheck = tierStore.checkLimit('connections', currentConnections + 1)

  if (!limitCheck.allowed) {
    throw new Error(
      `Connection limit reached. Current tier allows ${limitCheck.limit} connections.`
    )
  }

  // ... add connection
}
```

#### Query History Repository

Query history automatically prunes old entries when limits are reached:

```typescript
// In query-history-repository.ts
async create(data) {
  const currentCount = await this.count()
  const limitCheck = tierStore.checkLimit('queryHistory', currentCount + 1)

  if (!limitCheck.allowed && tierStore.currentTier === 'local') {
    await this.pruneOldest(1) // Auto-cleanup for free tier
  }

  // ... create record
}
```

## License Management

### Activating a License

```typescript
import { useTierStore } from '@/store/tier-store'

function LicenseForm() {
  const { activateLicense } = useTierStore()

  const handleActivate = async (key: string) => {
    const result = await activateLicense(key)

    if (result.success) {
      console.log('License activated:', result.tier)
    } else {
      console.error('Activation failed:', result.error)
    }
  }

  return (
    <form onSubmit={(e) => {
      e.preventDefault()
      handleActivate(e.currentTarget.licenseKey.value)
    }}>
      <input name="licenseKey" placeholder="SQL-INDIVIDUAL-..." />
      <button type="submit">Activate</button>
    </form>
  )
}
```

### Generating Development Licenses

In development mode, test licenses are automatically generated:

```typescript
import { devLicenses } from '@/lib/tiers'

// Available in console and code:
console.log(devLicenses.local)
console.log(devLicenses.individual)
console.log(devLicenses.team)
```

## Development Mode

Enable unlimited access for development:

```bash
# In .env
VITE_TIER_DEV_MODE=true
```

Or programmatically:

```typescript
const { enableDevMode } = useTierStore()
enableDevMode() // Only works in development
```

## API Reference

### Tier Store

```typescript
const {
  currentTier,           // Current active tier
  licenseKey,            // Active license key
  hasFeature,            // (feature) => boolean
  checkLimit,            // (limit, usage) => LimitCheckResult
  activateLicense,       // (key) => Promise<Result>
  deactivateLicense,     // () => void
  getFeatures,           // () => TierFeatures
  getLimits,             // () => TierLimits
  isAtLeastTier,         // (tier) => boolean
} = useTierStore()
```

### Feature Gate Hook

```typescript
const {
  allowed,              // Whether feature is available
  tier,                 // Current tier
  requiredTier,         // Required tier for feature
  showUpgrade,          // () => void - Show upgrade dialog
  isDevMode,            // Whether in dev mode
  isTeamFeature,        // Whether team-only
  isIndividualFeature,  // Whether individual+
} = useFeatureGate('sync')
```

### Tier Limit Hook

```typescript
const {
  usage,                // Current usage
  limit,                // Maximum allowed (null = unlimited)
  remaining,            // Remaining capacity
  percentage,           // Usage percentage (0-100)
  allowed,              // Whether within limit
  isNearLimit,          // >= 80% used
  isAtLimit,            // >= 100% used
  isUnlimited,          // No limit
  displayString,        // e.g., "3 / 5" or "3 / ∞"
  colorIndicator,       // 'green' | 'yellow' | 'red'
  showUpgrade,          // () => void
} = useTierLimit('connections', connectionCount)
```

## Upgrade Dialog Integration

The tier system dispatches custom events when upgrade is needed:

```typescript
// Listen for upgrade events
useEffect(() => {
  const handleUpgrade = (e: CustomEvent) => {
    const { feature, limit, currentTier, requiredTier } = e.detail

    // Show your upgrade modal/dialog
    showUpgradeModal({
      reason: feature || limit,
      currentTier,
      requiredTier,
    })
  }

  window.addEventListener('showUpgradeDialog', handleUpgrade)
  return () => window.removeEventListener('showUpgradeDialog', handleUpgrade)
}, [])
```

## Testing

### Unit Tests

```typescript
import { validateLicenseKey, generateLicenseKey } from '@/lib/tiers'

describe('License Validation', () => {
  it('validates correct license keys', async () => {
    const key = generateLicenseKey('individual')
    const result = await validateLicenseKey(key)

    expect(result.valid).toBe(true)
    expect(result.tier).toBe('individual')
  })

  it('rejects invalid checksums', async () => {
    const result = await validateLicenseKey('SQL-INDIVIDUAL-test-XXXX')

    expect(result.valid).toBe(false)
    expect(result.message).toContain('checksum')
  })
})
```

### Integration Tests

```typescript
import { renderHook, act } from '@testing-library/react'
import { useTierStore } from '@/store/tier-store'

describe('Tier Store', () => {
  it('enforces connection limits', () => {
    const { result } = renderHook(() => useTierStore())

    // Local tier: 5 connections max
    const check = result.current.checkLimit('connections', 6)

    expect(check.allowed).toBe(false)
    expect(check.limit).toBe(5)
  })
})
```

## Configuration

### Tier Limits

Edit `/config/tier-limits.ts` to adjust limits:

```typescript
export const TIER_LIMITS: Record<TierLevel, TierLimits> = {
  local: {
    connections: 5,
    queryHistory: 50,
    aiMemories: 10,
    savedQueries: 20,
    exportFileSize: 10 * 1024 * 1024, // 10MB
  },
  // ...
}
```

### Feature Flags

Add new features to `/config/tier-limits.ts`:

```typescript
export const TIER_FEATURES: Record<TierLevel, TierFeatures> = {
  local: {
    myNewFeature: false,
  },
  individual: {
    myNewFeature: true,
  },
  team: {
    myNewFeature: true,
  },
}
```

Then use it:

```typescript
const { allowed } = useFeatureGate('myNewFeature')
```

## Production Deployment

### Server-Side License Validation

Implement server validation in `/lib/tiers/license-validator.ts`:

```typescript
async function validateWithServer(key, tier, uuid) {
  const response = await fetch('/api/licenses/validate', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ key, tier, uuid }),
  })

  const data = await response.json()

  return {
    valid: data.valid,
    expiresAt: data.expiresAt ? new Date(data.expiresAt) : undefined,
    issuedAt: data.issuedAt ? new Date(data.issuedAt) : undefined,
  }
}
```

### License Generation (Backend)

Generate licenses server-side:

```typescript
import crypto from 'crypto'

function generateLicenseKey(tier: string, uuid?: string): string {
  const tierString = tier.toUpperCase()
  const licenseUuid = uuid || crypto.randomUUID()

  const data = `${tierString}-${licenseUuid}`
  const checksum = calculateCRC16(data)

  return `SQL-${tierString}-${licenseUuid}-${checksum}`
}
```

## Best Practices

1. **Always check limits before operations**
   - Don't wait for errors - proactively check limits
   - Show warnings when approaching limits

2. **Graceful degradation**
   - Provide alternatives when features are unavailable
   - Guide users to upgrade options

3. **Clear communication**
   - Show what tier is required for features
   - Explain benefits of upgrading

4. **Test all tiers**
   - Use dev mode to test unlimited access
   - Test limit enforcement for local tier
   - Verify license validation

5. **Handle edge cases**
   - Expired licenses
   - Network failures during validation
   - Race conditions in limit checking

## Troubleshooting

### "License validation failed"

- Check license key format: `SQL-{TIER}-{UUID}-{CHECKSUM}`
- Verify UUID is valid RFC 4122 format
- Ensure checksum matches (use `generateLicenseKey` for testing)
- Check network connectivity for server validation

### "Connection limit reached"

- Verify tier limits in `/config/tier-limits.ts`
- Check if dev mode is enabled
- Review connection count logic

### "Feature not available"

- Check `TIER_FEATURES` configuration
- Verify license hasn't expired
- Ensure tier store is initialized

## Support

For issues or questions:

- Check console for tier system logs (dev mode shows all checks)
- Review Redux DevTools for tier store state
- Enable dev mode to bypass all restrictions for testing

## License

This tier management system is part of Howlerops.
