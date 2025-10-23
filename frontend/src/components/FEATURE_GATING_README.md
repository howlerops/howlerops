# Feature Gating UI Components

A comprehensive system for showing value instead of blocking users. These components create desire by previewing locked features with compelling upgrade prompts.

## Philosophy

**Preview, Don't Block**: Instead of hiding features completely, show them with upgrade messaging. This approach:
- Creates desire by letting users see what they're missing
- Demonstrates value before asking for payment
- Improves conversion by showing, not telling
- Reduces friction in the upgrade funnel

## Components Overview

### 1. Feature Badge (`feature-badge.tsx`)
Tier indicators that show which plan a feature requires.

**Variants:**
- `inline` - Small badge next to feature name
- `tooltip` - Badge with hover tooltip
- `banner` - Larger banner style
- `pill` - Rounded pill shape

**Usage:**
```tsx
<FeatureBadge tier="individual" variant="inline" />
```

### 2. Upgrade Button (`upgrade-button.tsx`)
Contextual buttons that trigger upgrade modal with feature context.

**Variants:**
- `default` - Primary button style
- `outline` - Outlined button
- `ghost` - Transparent button
- `gradient` - Eye-catching gradient button

**Features:**
- Automatic event tracking
- Cooldown to prevent spam
- Multiple size options
- Icon support

**Usage:**
```tsx
<UpgradeButton
  trigger="queryHistorySearch"
  requiredTier="individual"
  size="sm"
>
  Unlock Search
</UpgradeButton>
```

### 3. Locked Feature Overlay (`locked-feature-overlay.tsx`)
Semi-transparent overlay for locked features.

**Features:**
- Shows disabled UI underneath
- Beautiful gradient overlays
- Benefit lists
- Dismissible option
- Smooth animations

**Usage:**
```tsx
<LockedFeatureOverlay
  feature="queryHistorySearch"
  requiredTier="individual"
  title="Search Query History"
  benefits={["Full-text search", "Advanced filters", "Save favorites"]}
>
  <Input placeholder="Search..." disabled />
</LockedFeatureOverlay>
```

### 4. Feature Preview (`feature-preview.tsx`)
Interactive previews of locked features.

**Variants:**
- `card` - Feature card with preview
- `overlay` - Full overlay with preview
- `inline` - Inline preview trigger

**Features:**
- Screenshot support
- Hover interactions
- Benefit highlights
- Toggle preview visibility

**Usage:**
```tsx
<FeaturePreview
  feature="sync"
  tier="individual"
  title="Cloud Sync"
  description="Keep workspace synced across devices"
  benefits={["Auto-sync", "Offline support", "99.9% uptime"]}
  variant="card"
>
  <SyncStatusIndicator disabled />
</FeaturePreview>
```

### 5. Soft Limit Warning (`soft-limit-warning.tsx`)
Warnings when approaching usage limits.

**Variants:**
- `banner` - Full width banner
- `toast` - Toast notification
- `inline` - Inline warning
- `badge` - Small badge indicator
- `card` - Detailed card with progress

**Features:**
- Progress bars
- Severity levels (info, warning, critical)
- Auto-calculated urgency
- Dismissible with cooldown

**Usage:**
```tsx
<SoftLimitWarning
  limit="connections"
  current={5}
  soft={5}
  variant="banner"
  requiredTier="individual"
/>
```

### 6. Value Comparison Table (`value-comparison-table.tsx`)
Feature comparison across tiers.

**Features:**
- Responsive design
- Current tier highlighting
- Mobile-friendly stacked view
- Upgrade CTAs
- Category grouping

**Usage:**
```tsx
<ValueComparisonTable
  currentTier="local"
  highlightTier="individual"
  features={[
    {
      name: 'Connections',
      local: '5',
      individual: 'Unlimited',
      team: 'Unlimited'
    },
    {
      name: 'Cloud Sync',
      local: false,
      individual: true,
      team: true
    }
  ]}
/>
```

### 7. Trial Banner (`trial-banner.tsx`)
Promotes free trial with countdown.

**Variants:**
- `banner` - Top/bottom banner
- `modal` - Full modal overlay
- `card` - Feature card
- `floating` - Floating notification

**Features:**
- Countdown display
- Auto-dismiss with cooldown
- Feature highlights
- Multiple positions

**Usage:**
```tsx
<TrialBanner
  daysRemaining={14}
  features={[
    "Full access to all Pro features",
    "No credit card required",
    "Cancel anytime"
  ]}
  variant="banner"
/>
```

### 8. Upgrade Success Animation (`upgrade-success-animation.tsx`)
Celebration after successful upgrade.

**Features:**
- Confetti animation
- Tier badge reveal
- Auto-close after duration
- Success confirmation
- Orbiting particles
- Star burst effects

**Usage:**
```tsx
<UpgradeSuccessAnimation
  tier="individual"
  duration={3000}
  onComplete={() => navigate('/dashboard')}
/>
```

### 9. Enhanced Feature Gate Hook (`use-feature-gate.ts`)
React hook with soft gating support.

**Modes:**
- `hard` - Completely blocks access (traditional)
- `soft` - Shows preview with upgrade prompt

**Features:**
- Helper methods for rendering
- Cooldown management
- Analytics integration
- Lazy-loaded components

**Usage:**
```tsx
const { allowed, renderLocked } = useFeatureGate('sync', 'soft')

if (!allowed) {
  return renderLocked(
    <SyncButton disabled />,
    "Cloud Sync",
    ["Sync across devices", "Auto-backup", "Never lose work"]
  )
}
```

## File Structure

```
frontend/src/
├── components/
│   ├── feature-badge.tsx                  # Tier badges
│   ├── upgrade-button.tsx                 # Upgrade CTAs
│   ├── locked-feature-overlay.tsx         # Feature overlays
│   ├── feature-preview.tsx                # Feature previews
│   ├── soft-limit-warning.tsx             # Usage warnings
│   ├── value-comparison-table.tsx         # Comparison tables
│   ├── trial-banner.tsx                   # Trial promotions
│   ├── upgrade-success-animation.tsx      # Success celebrations
│   ├── feature-gating-helpers.tsx         # Lazy-load helpers
│   ├── feature-gating/
│   │   └── index.ts                       # Barrel export
│   ├── FEATURE_GATING_EXAMPLES.md         # Usage examples
│   └── FEATURE_GATING_README.md           # This file
├── hooks/
│   └── use-feature-gate.ts                # Enhanced hook
└── types/
    └── global.d.ts                        # Global types
```

## Quick Start

1. **Import components:**
```tsx
import {
  FeatureBadge,
  UpgradeButton,
  LockedFeatureOverlay,
  useFeatureGate
} from '@/components/feature-gating'
```

2. **Use soft gating:**
```tsx
function MyFeature() {
  const { allowed, renderLocked } = useFeatureGate('sync', 'soft')

  if (!allowed) {
    return renderLocked(
      <FeatureUI disabled />,
      "Feature Name",
      ["Benefit 1", "Benefit 2", "Benefit 3"]
    )
  }

  return <FeatureUI />
}
```

3. **Show usage warnings:**
```tsx
const { checkLimit } = useTierStore()
const { isNearLimit } = checkLimit('connections', connections.length)

if (isNearLimit) {
  return <SoftLimitWarning limit="connections" current={5} soft={5} />
}
```

## Design Principles

### Visual Excellence
- Pixel-perfect spacing and alignment
- Beautiful gradient overlays
- Smooth animations with Framer Motion
- Consistent color schemes per tier
- Dark mode support

### User Experience
- Non-blocking approach (soft gating)
- Clear benefit communication
- Multiple interaction points
- Mobile-responsive
- Accessible (WCAG compliant)

### Technical Quality
- TypeScript strict mode
- Comprehensive prop types
- Reusable components
- Lazy loading support
- Zero circular dependencies

## Accessibility

All components support:
- Keyboard navigation (Tab, Enter, Escape)
- Screen readers (ARIA labels)
- Focus management
- High contrast mode
- Touch-friendly buttons (min 44px)

## Analytics

Built-in event tracking:
- `upgrade_button_click` - When upgrade button is clicked
- `upgrade_prompt_shown` - When upgrade prompt is displayed
- `feature_preview_viewed` - When feature preview is shown

Listen for custom events:
```tsx
window.addEventListener('showUpgradeDialog', (e) => {
  console.log('Upgrade dialog requested:', e.detail)
  // { trigger, feature, requiredTier, timestamp }
})
```

## Best Practices

### DO:
- Show value before asking for upgrade
- Use soft gating for secondary features
- Provide clear benefit lists
- Respect cooldown periods
- Track conversion metrics
- Test mobile responsiveness
- Celebrate successful upgrades

### DON'T:
- Block core functionality completely
- Show upgrade prompts too frequently
- Hide the upgrade path
- Use aggressive dark patterns
- Forget about accessibility
- Ignore mobile users
- Over-animate (keep it subtle)

## Testing

Enable dev mode to test with all features unlocked:
```tsx
const { enableDevMode } = useTierStore()
enableDevMode() // All features unlocked for testing
```

## Examples

See `FEATURE_GATING_EXAMPLES.md` for detailed usage examples including:
- Soft gate patterns
- Progressive disclosure
- Contextual prompts
- Feature discovery
- Mobile considerations
- Analytics integration

## Performance

Components are optimized for performance:
- Lazy loading with React.Suspense
- Minimal re-renders with useCallback
- Efficient animations with Framer Motion
- No circular dependencies
- Tree-shakeable exports

## Browser Support

Tested and working on:
- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+
- Mobile Safari iOS 14+
- Chrome Mobile Android 90+

## Contributing

When adding new feature gating components:
1. Follow existing patterns and naming conventions
2. Add TypeScript types for all props
3. Support dark mode
4. Make it responsive
5. Add accessibility features
6. Include usage examples
7. Test on mobile devices
8. Update the barrel export in `feature-gating/index.ts`

## Questions?

See the component source files for detailed prop documentation and implementation details.
