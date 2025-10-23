# Feature Gating UI Components - Usage Examples

This guide shows how to use the feature gating components to create value-focused upgrade experiences.

## Philosophy: Preview, Don't Block

Instead of hiding features completely, show them with upgrade messaging. This creates desire and demonstrates value.

## Quick Start

### 1. Soft Gate with Overlay

Show the feature UI with a beautiful overlay:

```tsx
import { useFeatureGate } from '@/hooks/use-feature-gate'

function CloudSyncButton() {
  const { allowed, renderLocked } = useFeatureGate('sync', 'soft')

  if (!allowed) {
    return renderLocked(
      <Button disabled>
        <Cloud className="h-4 w-4 mr-2" />
        Enable Sync
      </Button>,
      "Cloud Sync",
      [
        "Sync workspace across all devices",
        "Never lose your work again",
        "Automatic backups every 5 minutes"
      ]
    )
  }

  return <Button onClick={handleEnableSync}>Enable Sync</Button>
}
```

### 2. Feature Preview Card

Show a preview card for locked features:

```tsx
import { FeaturePreview } from '@/components/feature-preview'

function ProFeaturesSection() {
  return (
    <div className="grid gap-6 md:grid-cols-2">
      <FeaturePreview
        feature="queryHistorySearch"
        tier="individual"
        title="Search Query History"
        description="Find any query instantly with full-text search"
        benefits={[
          "Full-text search across 1000+ queries",
          "Filter by connection, date, status",
          "Save favorite queries"
        ]}
        variant="card"
      >
        <Input placeholder="Search queries..." disabled />
      </FeaturePreview>
    </div>
  )
}
```

### 3. Inline Upgrade Badge

Add a badge next to locked features:

```tsx
import { FeatureBadge } from '@/components/feature-badge'
import { UpgradeBadgeButton } from '@/components/upgrade-button'

function FeatureListItem({ feature, locked }) {
  return (
    <li className="flex items-center justify-between">
      <span>Query History Search</span>
      {locked && (
        <div className="flex items-center gap-2">
          <FeatureBadge tier="individual" variant="inline" />
          <UpgradeBadgeButton
            trigger="query_history_search"
            requiredTier="individual"
          />
        </div>
      )}
    </li>
  )
}
```

### 4. Soft Limit Warnings

Show warnings when approaching limits:

```tsx
import { SoftLimitWarning } from '@/components/soft-limit-warning'
import { useTierStore } from '@/store/tier-store'

function ConnectionsList({ connections }) {
  const { checkLimit } = useTierStore()
  const { allowed, isNearLimit } = checkLimit('connections', connections.length)

  return (
    <div className="space-y-4">
      {isNearLimit && (
        <SoftLimitWarning
          limit="connections"
          current={connections.length}
          soft={5}
          variant="banner"
          requiredTier="individual"
        />
      )}

      <ConnectionsList items={connections} />
    </div>
  )
}
```

### 5. Trial Banner

Promote free trial at the top of the app:

```tsx
import { TrialBanner } from '@/components/trial-banner'

function AppLayout() {
  const [showTrial, setShowTrial] = useState(true)

  return (
    <div>
      {showTrial && (
        <TrialBanner
          daysRemaining={14}
          features={[
            "Full access to all Pro features",
            "No credit card required",
            "Cancel anytime"
          ]}
          variant="banner"
          onDismiss={() => setShowTrial(false)}
        />
      )}

      <main>...</main>
    </div>
  )
}
```

### 6. Comparison Table

Show feature comparison on pricing page:

```tsx
import { ValueComparisonTable } from '@/components/value-comparison-table'
import { useTierStore } from '@/store/tier-store'

function PricingPage() {
  const { currentTier } = useTierStore()

  return (
    <ValueComparisonTable
      currentTier={currentTier}
      highlightTier="individual"
      features={[
        {
          name: 'Connections',
          local: '5',
          individual: 'Unlimited',
          team: 'Unlimited'
        },
        {
          name: 'Query History',
          local: '50 queries',
          individual: 'Unlimited',
          team: 'Unlimited'
        },
        {
          name: 'Cloud Sync',
          local: false,
          individual: true,
          team: true
        },
        {
          name: 'Team Sharing',
          local: false,
          individual: false,
          team: true
        }
      ]}
    />
  )
}
```

### 7. Success Animation

Celebrate successful upgrades:

```tsx
import { UpgradeSuccessAnimation } from '@/components/upgrade-success-animation'

function UpgradeFlow() {
  const [showSuccess, setShowSuccess] = useState(false)

  const handleUpgradeComplete = async (tier) => {
    // Process payment...
    setShowSuccess(true)
  }

  return (
    <>
      {/* Your upgrade flow */}

      {showSuccess && (
        <UpgradeSuccessAnimation
          tier="individual"
          onComplete={() => {
            setShowSuccess(false)
            navigate('/dashboard')
          }}
        />
      )}
    </>
  )
}
```

## Advanced Patterns

### Pattern 1: Progressive Disclosure

Start with inline badge, escalate to overlay on interaction:

```tsx
function QueryHistorySearch() {
  const { allowed } = useFeatureGate('queryHistorySearch', 'soft')
  const [showOverlay, setShowOverlay] = useState(!allowed)

  if (!allowed && showOverlay) {
    return (
      <LockedFeatureOverlay
        feature="queryHistorySearch"
        requiredTier="individual"
        title="Search Query History"
        benefits={["Full-text search", "Advanced filters", "Save favorites"]}
        dismissible={true}
        onDismiss={() => setShowOverlay(false)}
      >
        <SearchUI disabled />
      </LockedFeatureOverlay>
    )
  }

  return (
    <div className="relative">
      <SearchUI disabled={!allowed} />
      {!allowed && (
        <UpgradeBadgeButton
          trigger="query_search"
          requiredTier="individual"
          className="absolute top-2 right-2"
        />
      )}
    </div>
  )
}
```

### Pattern 2: Contextual Upgrade Prompts

Show upgrade prompts at relevant moments:

```tsx
function QueryEditor() {
  const { checkLimit } = useTierStore()
  const [queries, setQueries] = useState([])

  const handleSaveQuery = () => {
    const { allowed, isAtLimit } = checkLimit('savedQueries', queries.length)

    if (isAtLimit) {
      return (
        <UpgradePromptCard
          title="Query Limit Reached"
          description="You've saved 50 queries. Upgrade for unlimited."
          requiredTier="individual"
          benefits={[
            "Unlimited saved queries",
            "Organize with folders",
            "Share with team"
          ]}
        />
      )
    }

    // Save query...
  }
}
```

### Pattern 3: Feature Discovery

Let users discover locked features:

```tsx
function FeatureShowcase() {
  const proFeatures = [
    {
      feature: 'sync',
      tier: 'individual',
      title: 'Cloud Sync',
      description: 'Keep workspace synced across devices',
      benefits: ['Auto-sync every 5 min', 'Offline support', '99.9% uptime']
    },
    {
      feature: 'teamSharing',
      tier: 'team',
      title: 'Team Collaboration',
      description: 'Share connections and queries with team',
      benefits: ['Real-time collaboration', 'Role-based access', 'Activity log']
    }
  ]

  return (
    <GridFeaturePreview features={proFeatures} />
  )
}
```

## Mobile Considerations

All components are mobile-responsive:

```tsx
// Desktop: Full comparison table
// Mobile: Stacked tier selector
function ResponsivePricing() {
  const isMobile = useMediaQuery('(max-width: 768px)')

  if (isMobile) {
    return <MobileComparison currentTier="local" features={features} />
  }

  return <ValueComparisonTable currentTier="local" features={features} />
}
```

## Accessibility

All components support:
- Keyboard navigation
- Screen readers (ARIA labels)
- Focus management
- High contrast mode

```tsx
// Components handle accessibility automatically
<UpgradeButton
  trigger="feature_x"
  aria-label="Upgrade to unlock this feature"
>
  Unlock
</UpgradeButton>
```

## Analytics Integration

Track upgrade funnel with built-in events:

```tsx
// Components automatically track:
// - upgrade_button_click
// - upgrade_prompt_shown
// - feature_preview_viewed

// Listen to events for custom tracking:
window.addEventListener('showUpgradeDialog', (e) => {
  analytics.track('upgrade_dialog_shown', {
    feature: e.detail.feature,
    trigger: e.detail.trigger,
    tier: e.detail.requiredTier
  })
})
```

## Best Practices

### DO:
- Show value before asking for upgrade
- Use soft gating for secondary features
- Provide clear benefit lists
- Celebrate successful upgrades
- Track which features drive upgrades

### DON'T:
- Block core functionality completely
- Show upgrade prompts too frequently (respect cooldown)
- Hide upgrade path
- Use aggressive/dark patterns
- Forget mobile users

## Component Reference

| Component | Use Case | Variant |
|-----------|----------|---------|
| `FeatureBadge` | Inline tier indicators | inline, tooltip, banner, pill |
| `UpgradeButton` | Call-to-action buttons | default, outline, ghost, gradient |
| `LockedFeatureOverlay` | Soft-gate features | default, minimal, banner, inline |
| `FeaturePreview` | Showcase locked features | card, overlay, inline |
| `SoftLimitWarning` | Usage limit warnings | banner, toast, inline, badge, card |
| `ValueComparisonTable` | Feature comparison | full, compact, mobile |
| `TrialBanner` | Trial promotion | banner, modal, card, floating |
| `UpgradeSuccessAnimation` | Post-upgrade celebration | fullscreen, toast |

## Testing

Test upgrade flows with dev mode:

```tsx
// Enable all features for testing
import { useTierStore } from '@/store/tier-store'

function DevTools() {
  const { enableDevMode, disableDevMode, devMode } = useTierStore()

  return (
    <button onClick={devMode ? disableDevMode : enableDevMode}>
      {devMode ? 'Disable' : 'Enable'} Dev Mode
    </button>
  )
}
```

## Questions?

See the component source files for detailed prop documentation:
- `/components/feature-badge.tsx`
- `/components/upgrade-button.tsx`
- `/components/locked-feature-overlay.tsx`
- `/components/feature-preview.tsx`
- `/components/soft-limit-warning.tsx`
- `/components/value-comparison-table.tsx`
- `/components/trial-banner.tsx`
- `/components/upgrade-success-animation.tsx`
