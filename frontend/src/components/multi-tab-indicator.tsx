/**
 * Multi-Tab Indicator Component
 *
 * Visual indicator showing multi-tab synchronization status.
 * Displays active tab count, primary tab marker, connection status,
 * and password share button.
 *
 * Features:
 * - Active tab count badge
 * - Primary tab indicator
 * - Connection status icon
 * - Password share request button
 * - Tooltip with detailed information
 *
 * Usage:
 * ```tsx
 * <MultiTabIndicator
 *   onRequestPasswordShare={() => requestPasswordShare(['conn-1'])}
 * />
 * ```
 */

import {
  Clock,
  Crown,
  Info,
  Key,
  Users,
  Wifi,
  WifiOff} from 'lucide-react'
import React, { useMemo } from 'react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import { useMultiTabSync } from '@/hooks/use-multi-tab-sync'
import { cn } from '@/lib/utils'

export interface MultiTabIndicatorProps {
  /**
   * Callback when password share is requested
   */
  onRequestPasswordShare?: () => void

  /**
   * Connection IDs to request passwords for
   */
  connectionIdsToShare?: string[]

  /**
   * Custom className for styling
   */
  className?: string

  /**
   * Show detailed info in popover
   * @default true
   */
  showDetails?: boolean

  /**
   * Compact mode (smaller, icon-only)
   * @default false
   */
  compact?: boolean
}

/**
 * Multi-Tab Indicator Component
 */
export function MultiTabIndicator({
  onRequestPasswordShare,
  connectionIdsToShare = [],
  className,
  showDetails = true,
  compact = false
}: MultiTabIndicatorProps) {
  const {
    isConnected,
    tabCount,
    isPrimaryTab,
    activeTabs,
    currentTabId,
    isPasswordSharePending
  } = useMultiTabSync()

  // Format tab list for display
  const tabList = useMemo(() => {
    return Array.from(activeTabs.values())
      .sort((a, b) => a.lastHeartbeat - b.lastHeartbeat)
      .map((tab, index) => ({
        ...tab,
        number: index + 1,
        isCurrent: tab.tabId === currentTabId,
        age: Date.now() - tab.lastHeartbeat
      }))
  }, [activeTabs, currentTabId])

  // Format age for display
  const formatAge = (ms: number): string => {
    const seconds = Math.floor(ms / 1000)
    if (seconds < 60) return `${seconds}s ago`
    const minutes = Math.floor(seconds / 60)
    if (minutes < 60) return `${minutes}m ago`
    const hours = Math.floor(minutes / 60)
    return `${hours}h ago`
  }

  // Don't show if only one tab
  if (tabCount <= 1 && !showDetails) {
    return null
  }

  const handlePasswordShare = () => {
    if (onRequestPasswordShare) {
      onRequestPasswordShare()
    }
  }

  const indicator = (
    <div
      className={cn(
        'inline-flex items-center gap-2 px-3 py-1.5 rounded-lg border bg-background/50 backdrop-blur-sm transition-all',
        isConnected ? 'border-green-500/20 bg-green-500/5' : 'border-red-500/20 bg-red-500/5',
        compact && 'px-2 py-1',
        className
      )}
    >
      {/* Connection Status Icon */}
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <div className={cn(
              'flex items-center justify-center',
              compact ? 'w-4 h-4' : 'w-5 h-5'
            )}>
              {isConnected ? (
                <Wifi className={cn(
                  'text-green-600',
                  compact ? 'h-3 w-3' : 'h-4 w-4'
                )} />
              ) : (
                <WifiOff className={cn(
                  'text-red-600',
                  compact ? 'h-3 w-3' : 'h-4 w-4'
                )} />
              )}
            </div>
          </TooltipTrigger>
          <TooltipContent>
            <p>{isConnected ? 'Multi-tab sync active' : 'Multi-tab sync disconnected'}</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>

      {/* Tab Count Badge */}
      {!compact && (
        <div className="flex items-center gap-1.5">
          <Users className="h-4 w-4 text-muted-foreground" />
          <Badge variant="secondary" className="text-xs font-medium">
            {tabCount} {tabCount === 1 ? 'tab' : 'tabs'}
          </Badge>
        </div>
      )}

      {/* Primary Tab Indicator */}
      {isPrimaryTab && !compact && (
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <div className="flex items-center">
                <Crown className="h-4 w-4 text-yellow-600" />
              </div>
            </TooltipTrigger>
            <TooltipContent>
              <p>Primary tab</p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      )}

      {/* Password Share Button */}
      {onRequestPasswordShare && connectionIdsToShare.length > 0 && !isPrimaryTab && (
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="sm"
                onClick={handlePasswordShare}
                disabled={isPasswordSharePending}
                className={cn(
                  'h-6 px-2',
                  compact && 'h-5 px-1'
                )}
              >
                <Key className={cn(
                  'text-muted-foreground',
                  compact ? 'h-3 w-3' : 'h-3.5 w-3.5'
                )} />
                {isPasswordSharePending && (
                  <Clock className={cn(
                    'ml-1 animate-spin text-muted-foreground',
                    compact ? 'h-2 w-2' : 'h-3 w-3'
                  )} />
                )}
              </Button>
            </TooltipTrigger>
            <TooltipContent>
              <p>
                {isPasswordSharePending
                  ? 'Requesting passwords...'
                  : 'Request passwords from other tabs'}
              </p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      )}

      {/* Compact badge */}
      {compact && (
        <Badge variant="secondary" className="text-xs font-medium">
          {tabCount}
        </Badge>
      )}
    </div>
  )

  // Wrap in popover if details are enabled
  if (showDetails && tabCount > 1) {
    return (
      <Popover>
        <PopoverTrigger asChild>
          {indicator}
        </PopoverTrigger>
        <PopoverContent className="w-80" align="end">
          <div className="space-y-3">
            {/* Header */}
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Users className="h-4 w-4 text-muted-foreground" />
                <h4 className="font-semibold text-sm">Active Tabs</h4>
              </div>
              <Badge variant="secondary" className="text-xs">
                {tabCount} {tabCount === 1 ? 'tab' : 'tabs'}
              </Badge>
            </div>

            {/* Tab List */}
            <div className="space-y-2">
              {tabList.map((tab) => (
                <div
                  key={tab.tabId}
                  className={cn(
                    'flex items-center justify-between p-2 rounded-md transition-colors',
                    tab.isCurrent && 'bg-primary/5 border border-primary/20'
                  )}
                >
                  <div className="flex items-center gap-2">
                    <div className={cn(
                      'flex items-center justify-center w-6 h-6 rounded-full text-xs font-medium',
                      tab.isCurrent ? 'bg-primary text-primary-foreground' : 'bg-muted text-muted-foreground'
                    )}>
                      {tab.number}
                    </div>
                    <div className="flex flex-col">
                      <div className="flex items-center gap-1">
                        <span className="text-sm font-medium">
                          {tab.isCurrent ? 'This tab' : `Tab ${tab.number}`}
                        </span>
                        {tab.isPrimary && (
                          <Crown className="h-3 w-3 text-yellow-600" />
                        )}
                      </div>
                      <span className="text-xs text-muted-foreground">
                        {formatAge(tab.age)}
                      </span>
                    </div>
                  </div>

                  {/* Status indicator */}
                  <div className={cn(
                    'w-2 h-2 rounded-full',
                    tab.age < 15000 ? 'bg-green-500' : 'bg-yellow-500'
                  )} />
                </div>
              ))}
            </div>

            {/* Info message */}
            <div className="flex items-start gap-2 p-2 rounded-md bg-muted/50">
              <Info className="h-4 w-4 text-muted-foreground mt-0.5 flex-shrink-0" />
              <p className="text-xs text-muted-foreground">
                Changes are automatically synchronized across all tabs. The primary tab
                {isPrimaryTab ? ' (this tab)' : ''} coordinates background tasks.
              </p>
            </div>
          </div>
        </PopoverContent>
      </Popover>
    )
  }

  return indicator
}
