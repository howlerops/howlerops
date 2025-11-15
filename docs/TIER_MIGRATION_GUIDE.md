# Tier System Migration Guide

Step-by-step guide for integrating the tier management system into Howlerops.

## Phase 1: Core Setup (5 minutes)

### Step 1: Initialize the Store

**File: `frontend/src/App.tsx` or main entry point**

```typescript
import { initializeTierStore } from '@/lib/tiers'

function App() {
  useEffect(() => {
    // Initialize tier system on app startup
    initializeTierStore()
  }, [])

  return (
    // ... your app
  )
}
```

### Step 2: Verify Installation

Open your app and check the browser console:

```
🔑 Development License Keys
Local: SQL-LOCAL-...
Individual: SQL-INDIVIDUAL-...
Team: SQL-TEAM-...
```

You should see the tier store initialized with default (Local) tier.

## Phase 2: Visual Integration (10 minutes)

### Step 3: Add Tier Badge to Header

**Already done!** The tier badge is in the header at:
`/frontend/src/components/layout/header.tsx`

Verify it appears in the top-right area of the header.

### Step 4: Create Settings Page Tab

**File: `frontend/src/pages/settings.tsx`** (or wherever your settings are)

```typescript
import { TierSettingsPanel } from '@/components/tier-settings-panel'

function Settings() {
  const [activeTab, setActiveTab] = useState('general')

  return (
    <Tabs value={activeTab} onValueChange={setActiveTab}>
      <TabsList>
        <TabsTrigger value="general">General</TabsTrigger>
        <TabsTrigger value="tier">Plan & License</TabsTrigger>
        {/* ... other tabs */}
      </TabsList>

      <TabsContent value="general">
        {/* General settings */}
      </TabsContent>

      <TabsContent value="tier">
        <TierSettingsPanel />
      </TabsContent>
    </Tabs>
  )
}
```

## Phase 3: Feature Gates (30 minutes)

### Step 5: Gate Cloud Sync Feature

**Example location: Where sync is triggered**

```typescript
import { useFeatureGate } from '@/hooks/use-feature-gate'

function SyncButton() {
  const { allowed, showUpgrade, requiredTier } = useFeatureGate('sync')

  if (!allowed) {
    return (
      <Button onClick={showUpgrade} variant="outline">
        <Cloud className="mr-2 h-4 w-4" />
        Sync (Requires {requiredTier})
      </Button>
    )
  }

  return (
    <Button onClick={handleSync}>
      <Cloud className="mr-2 h-4 w-4" />
      Sync to Cloud
    </Button>
  )
}
```

### Step 6: Gate Team Features

**Example: Team sharing UI**

```typescript
import { useFeatureGate } from '@/hooks/use-feature-gate'

function ShareQueryButton() {
  const { allowed, showUpgrade } = useFeatureGate('teamSharing')

  return (
    <DropdownMenu>
      <DropdownMenuTrigger>
        <Button>Share</Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent>
        <DropdownMenuItem onClick={copyLink}>
          Copy Link
        </DropdownMenuItem>
        {allowed ? (
          <DropdownMenuItem onClick={shareWithTeam}>
            Share with Team
          </DropdownMenuItem>
        ) : (
          <DropdownMenuItem onClick={showUpgrade}>
            Share with Team (Team Plan)
          </DropdownMenuItem>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
```

### Step 7: Add Feature Badges

Show which features require upgrades:

```typescript
function FeatureCard({ feature, children }) {
  const gate = useFeatureGate(feature)

  return (
    <Card>
      <CardHeader>
        <CardTitle>
          {children}
          {!gate.allowed && (
            <Badge variant="outline" className="ml-2">
              {gate.requiredTier}
            </Badge>
          )}
        </CardTitle>
      </CardHeader>
      <CardContent>
        {gate.allowed ? (
          <FeatureContent />
        ) : (
          <UpgradePrompt onClick={gate.showUpgrade} />
        )}
      </CardContent>
    </Card>
  )
}
```

## Phase 4: Limit Warnings (20 minutes)

### Step 8: Add Connection Limit Warning

**File: Connection Manager or similar**

```typescript
import { useTierLimit } from '@/hooks/use-tier-limit'

function ConnectionManager() {
  const connections = useConnectionStore(s => s.connections)
  const limit = useTierLimit('connections', connections.length)

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h2>Connections</h2>
        <span className="text-sm text-muted-foreground">
          {limit.displayString}
        </span>
      </div>

      {!limit.isUnlimited && (
        <Progress
          value={limit.percentage}
          className={cn(
            'mb-4',
            limit.colorIndicator === 'red' && 'bg-red-100',
            limit.colorIndicator === 'yellow' && 'bg-yellow-100',
          )}
        />
      )}

      {limit.isNearLimit && !limit.isAtLimit && (
        <Alert className="mb-4">
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>
            You're approaching your connection limit. Consider upgrading.
          </AlertDescription>
        </Alert>
      )}

      {limit.isAtLimit && (
        <Alert variant="destructive" className="mb-4">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>
            Connection limit reached. <Button onClick={limit.showUpgrade}>Upgrade</Button>
          </AlertDescription>
        </Alert>
      )}

      {/* Connection list */}
    </div>
  )
}
```

### Step 9: Add Query History Warning

**File: Query history viewer**

```typescript
import { useTierLimit } from '@/hooks/use-tier-limit'

function QueryHistory() {
  const [historyCount, setHistoryCount] = useState(0)

  useEffect(() => {
    const loadCount = async () => {
      const repo = getQueryHistoryRepository()
      setHistoryCount(await repo.count())
    }
    loadCount()
  }, [])

  const limit = useTierLimit('queryHistory', historyCount)

  return (
    <div>
      <div className="flex items-center justify-between">
        <h3>Query History</h3>
        <Badge variant="outline">{limit.displayString}</Badge>
      </div>

      {limit.isNearLimit && (
        <Alert>
          <AlertDescription>
            Your query history is nearly full. Older queries will be auto-pruned.
            <Button onClick={limit.showUpgrade} size="sm" className="ml-2">
              Upgrade for Unlimited
            </Button>
          </AlertDescription>
        </Alert>
      )}

      {/* History list */}
    </div>
  )
}
```

## Phase 5: Upgrade Flow (45 minutes)

### Step 10: Create Upgrade Modal

**File: `frontend/src/components/upgrade-modal.tsx`**

```typescript
import { Dialog, DialogContent, DialogHeader } from '@/components/ui/dialog'
import { TierBadgeList } from '@/components/tier-badge'
import { useTierStore } from '@/store/tier-store'

interface UpgradeModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  reason?: string
  currentTier?: string
  requiredTier?: string
}

export function UpgradeModal({
  open,
  onOpenChange,
  reason,
  requiredTier,
}: UpgradeModalProps) {
  const navigate = useNavigate()

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl">
        <DialogHeader>
          <DialogTitle>Upgrade Your Plan</DialogTitle>
          <DialogDescription>
            {reason && `To use ${reason}, `}
            you need to upgrade to {requiredTier || 'a higher'} tier.
          </DialogDescription>
        </DialogHeader>

        <TierBadgeList
          variant="card"
          highlightCurrent
          onSelect={(tier) => {
            // Navigate to settings or payment
            navigate(`/settings?tab=tier&target=${tier}`)
            onOpenChange(false)
          }}
        />

        <div className="flex justify-between items-center pt-4">
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            Maybe Later
          </Button>
          <Button onClick={() => navigate('/settings?tab=tier')}>
            View Plans
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
```

### Step 11: Wire Up Upgrade Modal

**File: `frontend/src/App.tsx` or layout wrapper**

```typescript
import { UpgradeModal } from '@/components/upgrade-modal'

function App() {
  const [upgradeModal, setUpgradeModal] = useState<{
    open: boolean
    detail?: any
  }>({ open: false })

  useEffect(() => {
    // Listen for upgrade events from tier system
    const handleUpgrade = (e: CustomEvent) => {
      setUpgradeModal({
        open: true,
        detail: e.detail,
      })
    }

    window.addEventListener('showUpgradeDialog', handleUpgrade)
    return () => window.removeEventListener('showUpgradeDialog', handleUpgrade)
  }, [])

  return (
    <>
      {/* Your app */}

      <UpgradeModal
        open={upgradeModal.open}
        onOpenChange={(open) => setUpgradeModal({ ...upgradeModal, open })}
        reason={upgradeModal.detail?.feature || upgradeModal.detail?.limit}
        requiredTier={upgradeModal.detail?.requiredTier}
      />
    </>
  )
}
```

## Phase 6: Testing (15 minutes)

### Step 12: Test Local Tier

1. Open app (default is Local tier)
2. Try to add 6th connection - should fail
3. Check query history limit (50 max)
4. Verify sync button shows upgrade prompt
5. Verify team features are hidden/gated

### Step 13: Test Individual Tier

```typescript
// In browser console
import { useTierStore } from '@/store/tier-store'
import { devLicenses } from '@/lib/tiers'

// Get the Individual license
console.log(devLicenses.individual)
// Copy it

// Activate it
const result = await useTierStore.getState().activateLicense(devLicenses.individual)
console.log(result)
```

Then verify:
- Can add unlimited connections
- Can see sync features
- Team features still gated
- Badge shows "Individual"

### Step 14: Test Team Tier

```typescript
// Activate team license
await useTierStore.getState().activateLicense(devLicenses.team)
```

Verify:
- All features unlocked
- Team features visible
- Badge shows "Team"

### Step 15: Test Dev Mode

```typescript
// Enable dev mode
useTierStore.getState().enableDevMode()
```

Verify:
- All limits bypassed
- All features unlocked
- "DEV" badge appears

## Phase 7: Production (varies)

### Step 16: Implement Server Validation

**Backend: Create `/api/licenses/validate` endpoint**

```typescript
// Example Express endpoint
app.post('/api/licenses/validate', async (req, res) => {
  const { key, tier, uuid } = req.body

  // Check database
  const license = await db.licenses.findOne({ key })

  if (!license) {
    return res.json({ valid: false, message: 'License not found' })
  }

  if (license.revoked) {
    return res.json({ valid: false, message: 'License revoked' })
  }

  if (license.expiresAt && new Date() > new Date(license.expiresAt)) {
    return res.json({ valid: false, message: 'License expired' })
  }

  res.json({
    valid: true,
    expiresAt: license.expiresAt,
    issuedAt: license.issuedAt,
  })
})
```

**Frontend: Update `validateWithServer` in `license-validator.ts`**

Uncomment and update the server validation code.

### Step 17: Implement License Generation

**Backend: License generation endpoint**

```typescript
app.post('/api/licenses/generate', requireAdmin, async (req, res) => {
  const { tier, userId, durationDays } = req.body

  const uuid = crypto.randomUUID()
  const tierString = tier.toUpperCase()
  const data = `${tierString}-${uuid}`
  const checksum = calculateCRC16(data)

  const key = `SQL-${tierString}-${uuid}-${checksum}`

  const license = await db.licenses.create({
    key,
    tier,
    userId,
    issuedAt: new Date(),
    expiresAt: durationDays
      ? new Date(Date.now() + durationDays * 24 * 60 * 60 * 1000)
      : null,
  })

  res.json({ license })
})
```

### Step 18: Payment Integration

**Option A: Stripe**

```typescript
// After successful payment
const license = await fetch('/api/licenses/generate', {
  method: 'POST',
  body: JSON.stringify({
    tier: 'individual',
    userId: user.id,
    durationDays: 365,
  }),
})

// Email license to user
await sendLicenseEmail(user.email, license.key)

// Auto-activate in-app
await useTierStore.getState().activateLicense(license.key)
```

**Option B: Gumroad/LemonSqueezy**

Configure webhook to call your license generation endpoint.

### Step 19: Remove Dev Mode in Production

**File: `.env.production`**

```bash
VITE_TIER_DEV_MODE=false  # or omit entirely
```

### Step 20: Analytics

Track tier-related events:

```typescript
// In useFeatureGate hook
useEffect(() => {
  if (!allowed) {
    analytics.track('Feature Gate Blocked', {
      feature,
      currentTier,
      requiredTier,
    })
  }
}, [allowed, feature])

// In upgrade modal
function handleUpgradeClick(targetTier) {
  analytics.track('Upgrade Clicked', {
    from: currentTier,
    to: targetTier,
    reason,
  })
}
```

## Verification Checklist

After migration, verify:

### UI Elements
- [ ] Tier badge appears in header
- [ ] Badge shows correct tier
- [ ] Badge is clickable and navigates to settings
- [ ] Settings page has tier/license tab
- [ ] License activation form works
- [ ] Upgrade modal appears when needed

### Limits
- [ ] Connection limit enforced
- [ ] Query history limit enforced
- [ ] Limits auto-prune when reached (Local)
- [ ] Warnings appear at 80% usage
- [ ] Errors appear at 100% usage
- [ ] Progress bars show usage correctly

### Features
- [ ] Sync feature gated for Local
- [ ] Team features gated for Local/Individual
- [ ] Feature gates show upgrade prompts
- [ ] Upgrade prompts navigate correctly
- [ ] All features work in Individual
- [ ] All features work in Team

### License Management
- [ ] License activates successfully
- [ ] License persists across refreshes
- [ ] License deactivation works
- [ ] Expiration warnings appear
- [ ] Expired licenses revert to Local
- [ ] Invalid licenses rejected

### Development Tools
- [ ] Dev licenses generated in console
- [ ] Dev mode can be enabled
- [ ] Dev mode bypasses all restrictions
- [ ] Dev mode badge appears
- [ ] Type checking passes

### Production
- [ ] Server validation working
- [ ] License generation working
- [ ] Payment integration working
- [ ] Email delivery working
- [ ] Analytics tracking
- [ ] No dev mode in production

## Rollback Plan

If issues arise, you can temporarily disable tier enforcement:

```typescript
// In .env
VITE_TIER_DEV_MODE=true
```

This will bypass all limits and feature gates while you investigate.

## Support

For help during migration:

1. Check console for tier system logs
2. Enable dev mode to test without restrictions
3. Review documentation in `/lib/tiers/README.md`
4. Test with dev licenses before using real ones

## Timeline Estimate

- Phase 1 (Setup): 5 minutes
- Phase 2 (Visual): 10 minutes
- Phase 3 (Features): 30 minutes
- Phase 4 (Limits): 20 minutes
- Phase 5 (Upgrade Flow): 45 minutes
- Phase 6 (Testing): 15 minutes
- Phase 7 (Production): Varies (hours to days)

**Total Development Time: ~2 hours**
**Production Integration: Additional time for backend**

---

**Next Steps After Migration:**

1. Monitor tier adoption rates
2. Track upgrade conversions
3. Analyze feature usage patterns
4. Gather user feedback
5. Iterate on pricing/features
6. Add usage analytics dashboard
7. Implement team management UI
8. Build admin license management panel
