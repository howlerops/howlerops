/**
 * Soft Limit Toast
 *
 * Non-blocking toast notification when soft limits are exceeded.
 * Shows upgrade value without preventing actions.
 */

import React from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Sparkles, Database, History, Brain, FileDown } from 'lucide-react'
import type { UpgradeTrigger } from '@/store/upgrade-prompt-store'

/**
 * Limit type to icon mapping
 */
const LIMIT_ICONS: Record<string, React.ReactNode> = {
  connections: <Database className="w-5 h-5" />,
  queryHistory: <History className="w-5 h-5" />,
  aiMemories: <Brain className="w-5 h-5" />,
  export: <FileDown className="w-5 h-5" />,
}

/**
 * Toast configuration for each limit type
 */
interface ToastConfig {
  title: string
  description: string
  action: string
}

const TOAST_CONFIGS: Record<string, ToastConfig> = {
  connections: {
    title: 'Connection limit reached',
    description: "You've reached your 5 connection limit. Upgrade for unlimited connections.",
    action: 'Upgrade for Unlimited',
  },
  queryHistory: {
    title: 'Query history limit reached',
    description: 'Older queries will be auto-deleted. Upgrade for unlimited history.',
    action: 'Keep All Queries',
  },
  aiMemories: {
    title: 'AI memory limit reached',
    description: 'Older memories will be forgotten. Upgrade for unlimited AI context.',
    action: 'Upgrade for More Memory',
  },
  export: {
    title: 'Export file too large',
    description: 'Your export exceeds the 10MB limit. Upgrade for larger exports.',
    action: 'Upgrade for Larger Exports',
  },
}

export interface SoftLimitToastProps {
  /**
   * Limit type that was exceeded
   */
  limitType: keyof typeof TOAST_CONFIGS

  /**
   * Current usage value
   */
  usage: number

  /**
   * Soft limit value
   */
  softLimit: number

  /**
   * Callback when upgrade is clicked
   */
  onUpgrade?: () => void

  /**
   * Custom title (overrides default)
   */
  title?: string

  /**
   * Custom description (overrides default)
   */
  description?: string

  /**
   * Custom action text (overrides default)
   */
  actionText?: string

  /**
   * Auto-dismiss duration in milliseconds
   * @default 10000
   */
  duration?: number
}

/**
 * Show a soft limit toast notification
 *
 * @example
 * ```typescript
 * showSoftLimitToast({
 *   limitType: 'connections',
 *   usage: 6,
 *   softLimit: 5,
 *   onUpgrade: () => showUpgradeModal('connections')
 * })
 * ```
 */
export function showSoftLimitToast({
  limitType,
  usage,
  softLimit,
  onUpgrade,
  title: customTitle,
  description: customDescription,
  actionText: customActionText,
  duration = 10000,
}: SoftLimitToastProps) {
  const config = TOAST_CONFIGS[limitType]
  if (!config) {
    console.warn(`Unknown limit type: ${limitType}`)
    return
  }

  const icon = LIMIT_ICONS[limitType]
  const title = customTitle || config.title
  const description = customDescription || config.description
  const actionText = customActionText || config.action

  toast(
    <div className="flex items-start gap-3 w-full">
      <div className="p-2 rounded-lg bg-orange-100 text-orange-600 dark:bg-orange-950">
        {icon}
      </div>
      <div className="flex-1 space-y-1">
        <p className="font-semibold text-sm">{title}</p>
        <p className="text-xs text-muted-foreground">{description}</p>
        <div className="flex items-center gap-2 mt-2">
          <Button
            size="sm"
            className="h-7 bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white"
            onClick={() => {
              onUpgrade?.()
              toast.dismiss()
            }}
          >
            <Sparkles className="w-3 h-3 mr-1" />
            {actionText}
          </Button>
          <Button
            size="sm"
            variant="ghost"
            className="h-7 text-xs"
            onClick={() => toast.dismiss()}
          >
            Dismiss
          </Button>
        </div>
      </div>
    </div>,
    {
      duration,
      className: 'p-4',
    }
  )
}

/**
 * Hook for easy soft limit toast usage
 */
export function useSoftLimitToast() {
  const show = (props: SoftLimitToastProps) => {
    showSoftLimitToast(props)
  }

  return { showSoftLimitToast: show }
}
