/**
 * Tier Settings Panel
 *
 * Example settings panel component demonstrating tier management features.
 * Use this as a reference for building your own tier management UI.
 *
 * Features:
 * - Current tier display with badge
 * - License activation form
 * - Usage statistics and limits
 * - Tier comparison cards
 * - Upgrade options
 */

import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTierStore } from '@/store/tier-store'
import { useTierLimit } from '@/hooks/use-tier-limit'
import { useFeatureGate } from '@/hooks/use-feature-gate'
import { TierBadge, TierBadgeList } from '@/components/tier-badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { Alert } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { CheckCircle2, XCircle, AlertTriangle, Loader2 } from 'lucide-react'
import { useConnectionStore } from '@/store/connection-store'
import { getQueryHistoryRepository } from '@/lib/storage/repositories/query-history-repository'

/**
 * Tier Settings Panel Component
 *
 * @example
 * ```typescript
 * import { TierSettingsPanel } from '@/components/tier-settings-panel'
 *
 * function Settings() {
 *   return (
 *     <div>
 *       <h1>Settings</h1>
 *       <TierSettingsPanel />
 *     </div>
 *   )
 * }
 * ```
 */
export function TierSettingsPanel() {
  const navigate = useNavigate()
  const {
    currentTier,
    licenseKey,
    expiresAt,
    activateLicense,
    deactivateLicense,
    getFeatures,
    getLimits,
  } = useTierStore()

  const { connections } = useConnectionStore()
  const [queryHistoryCount, setQueryHistoryCount] = useState(0)
  const [licenseInput, setLicenseInput] = useState('')
  const [isActivating, setIsActivating] = useState(false)
  const [activationError, setActivationError] = useState<string | null>(null)

  const features = getFeatures()
  const limits = getLimits()

  // Load query history count
  React.useEffect(() => {
    const loadCount = async () => {
      const repo = getQueryHistoryRepository()
      const count = await repo.count()
      setQueryHistoryCount(count)
    }
    loadCount()
  }, [])

  // Usage limits
  const connectionLimit = useTierLimit('connections', connections.length)
  const queryHistoryLimit = useTierLimit('queryHistory', queryHistoryCount)

  // Feature gates
  const syncGate = useFeatureGate('sync')
  const teamGate = useFeatureGate('teamSharing')
  const rbacGate = useFeatureGate('rbac')

  const handleActivateLicense = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsActivating(true)
    setActivationError(null)

    try {
      const result = await activateLicense(licenseInput)

      if (result.success) {
        setLicenseInput('')
        // Success feedback could be a toast or similar
        alert(`License activated successfully! Welcome to ${result.tier} tier.`)
      } else {
        setActivationError(result.error || 'License activation failed')
      }
    } catch (error) {
      setActivationError(error instanceof Error ? error.message : 'Unknown error')
    } finally {
      setIsActivating(false)
    }
  }

  const handleDeactivate = () => {
    if (confirm('Are you sure you want to deactivate your license? You will be reverted to the Local tier.')) {
      deactivateLicense()
    }
  }

  return (
    <div className="space-y-6 max-w-4xl">
      {/* Current Tier Section */}
      <Card>
        <CardHeader>
          <CardTitle>Current Plan</CardTitle>
          <CardDescription>
            Your active SQL Studio tier and license information
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <TierBadge variant="card" showExpiration />

          {licenseKey && (
            <div className="space-y-2">
              <Label>License Key</Label>
              <div className="flex items-center gap-2">
                <Input
                  value={licenseKey}
                  readOnly
                  className="font-mono text-sm"
                />
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => navigator.clipboard.writeText(licenseKey)}
                >
                  Copy
                </Button>
              </div>
              {expiresAt && (
                <p className="text-sm text-muted-foreground">
                  Expires: {new Date(expiresAt).toLocaleDateString()}
                </p>
              )}
              <Button
                variant="destructive"
                size="sm"
                onClick={handleDeactivate}
              >
                Deactivate License
              </Button>
            </div>
          )}

          {!licenseKey && currentTier === 'local' && (
            <Alert>
              <AlertTriangle className="h-4 w-4" />
              <p className="text-sm">
                You're on the free Local tier. Upgrade to unlock unlimited connections,
                cloud sync, and more!
              </p>
            </Alert>
          )}
        </CardContent>
      </Card>

      {/* License Activation */}
      {!licenseKey && (
        <Card>
          <CardHeader>
            <CardTitle>Activate License</CardTitle>
            <CardDescription>
              Enter your license key to upgrade your SQL Studio tier
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleActivateLicense} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="license-key">License Key</Label>
                <Input
                  id="license-key"
                  placeholder="SQL-INDIVIDUAL-..."
                  value={licenseInput}
                  onChange={(e) => setLicenseInput(e.target.value)}
                  className="font-mono"
                  disabled={isActivating}
                />
                {activationError && (
                  <p className="text-sm text-destructive">{activationError}</p>
                )}
              </div>
              <Button type="submit" disabled={isActivating || !licenseInput.trim()}>
                {isActivating && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                Activate License
              </Button>
            </form>
          </CardContent>
        </Card>
      )}

      {/* Usage Statistics */}
      <Card>
        <CardHeader>
          <CardTitle>Usage & Limits</CardTitle>
          <CardDescription>
            Your current usage against tier limits
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          {/* Connections */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label>Database Connections</Label>
              <span className="text-sm font-medium">{connectionLimit.displayString}</span>
            </div>
            {!connectionLimit.isUnlimited && (
              <>
                <Progress
                  value={connectionLimit.percentage}
                  className={
                    connectionLimit.colorIndicator === 'red'
                      ? 'bg-red-100'
                      : connectionLimit.colorIndicator === 'yellow'
                        ? 'bg-yellow-100'
                        : 'bg-green-100'
                  }
                />
                {connectionLimit.isNearLimit && (
                  <p className="text-sm text-orange-600">
                    ⚠ You're approaching your connection limit
                  </p>
                )}
                {connectionLimit.isAtLimit && (
                  <p className="text-sm text-destructive">
                    ⚠ Connection limit reached. Upgrade to add more connections.
                  </p>
                )}
              </>
            )}
            {connectionLimit.isUnlimited && (
              <p className="text-sm text-green-600">✓ Unlimited connections</p>
            )}
          </div>

          {/* Query History */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label>Query History</Label>
              <span className="text-sm font-medium">{queryHistoryLimit.displayString}</span>
            </div>
            {!queryHistoryLimit.isUnlimited && (
              <>
                <Progress
                  value={queryHistoryLimit.percentage}
                  className={
                    queryHistoryLimit.colorIndicator === 'red'
                      ? 'bg-red-100'
                      : queryHistoryLimit.colorIndicator === 'yellow'
                        ? 'bg-yellow-100'
                        : 'bg-green-100'
                  }
                />
                {queryHistoryLimit.isNearLimit && (
                  <p className="text-sm text-orange-600">
                    ⚠ You're approaching your query history limit. Older queries will be auto-pruned.
                  </p>
                )}
              </>
            )}
            {queryHistoryLimit.isUnlimited && (
              <p className="text-sm text-green-600">✓ Unlimited query history</p>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Features */}
      <Card>
        <CardHeader>
          <CardTitle>Features</CardTitle>
          <CardDescription>
            Available features in your current tier
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            <FeatureRow
              name="Cloud Sync"
              enabled={features.sync}
              requiredTier={syncGate.requiredTier}
            />
            <FeatureRow
              name="Multi-Device Support"
              enabled={features.multiDevice}
            />
            <FeatureRow
              name="AI Memory Sync"
              enabled={features.aiMemorySync}
            />
            <FeatureRow
              name="Team Sharing"
              enabled={features.teamSharing}
              requiredTier={teamGate.requiredTier}
            />
            <FeatureRow
              name="Role-Based Access Control"
              enabled={features.rbac}
              requiredTier={rbacGate.requiredTier}
            />
            <FeatureRow
              name="Audit Logging"
              enabled={features.auditLog === true}
            />
            <FeatureRow
              name="Priority Support"
              enabled={features.prioritySupport}
            />
            <FeatureRow
              name="Custom Themes"
              enabled={features.customThemes}
            />
          </div>
        </CardContent>
      </Card>

      {/* Tier Comparison */}
      <Card>
        <CardHeader>
          <CardTitle>Available Plans</CardTitle>
          <CardDescription>
            Compare all SQL Studio tiers
          </CardDescription>
        </CardHeader>
        <CardContent>
          <TierBadgeList
            variant="card"
            highlightCurrent
            onSelect={(tier) => {
              if (tier !== currentTier) {
                // Handle upgrade/downgrade navigation
                console.log('Selected tier:', tier)
              }
            }}
          />
        </CardContent>
      </Card>
    </div>
  )
}

/**
 * Feature Row Component
 */
interface FeatureRowProps {
  name: string
  enabled: boolean
  requiredTier?: string | null
}

function FeatureRow({ name, enabled, requiredTier }: FeatureRowProps) {
  return (
    <div className="flex items-center justify-between py-2 border-b last:border-0">
      <span className="text-sm">{name}</span>
      <div className="flex items-center gap-2">
        {enabled ? (
          <>
            <CheckCircle2 className="h-4 w-4 text-green-600" />
            <Badge variant="outline" className="bg-green-50 text-green-700 border-green-200">
              Enabled
            </Badge>
          </>
        ) : (
          <>
            <XCircle className="h-4 w-4 text-gray-400" />
            {requiredTier && (
              <Badge variant="outline" className="text-xs">
                {requiredTier} required
              </Badge>
            )}
          </>
        )}
      </div>
    </div>
  )
}
