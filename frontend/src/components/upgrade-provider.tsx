/**
 * Upgrade Provider
 *
 * Centralized provider for upgrade prompt management.
 * Handles global upgrade events and state.
 *
 * Usage:
 * Wrap your app with this provider to enable upgrade prompts globally.
 */

import React, { createContext, useContext, useState, useEffect, useCallback } from 'react'
import { UpgradeModal } from './upgrade-modal'
import { MultiDeviceBanner } from './value-indicators/multi-device-banner'
import { useUpgradePromptStore, type UpgradeTrigger } from '@/store/upgrade-prompt-store'
import {
  initializeReminderSystem,
  registerReminderListeners,
} from '@/lib/upgrade-reminders'

interface UpgradeContextValue {
  /**
   * Show upgrade modal with specific trigger
   */
  showUpgrade: (trigger: UpgradeTrigger) => void

  /**
   * Check if upgrade modal is currently shown
   */
  isUpgradeModalOpen: boolean
}

const UpgradeContext = createContext<UpgradeContextValue | null>(null)

/**
 * Hook to access upgrade context - standard React Context pattern
 */
// eslint-disable-next-line react-refresh/only-export-components
export function useUpgrade() {
  const context = useContext(UpgradeContext)
  if (!context) {
    throw new Error('useUpgrade must be used within UpgradeProvider')
  }
  return context
}

export interface UpgradeProviderProps {
  children: React.ReactNode

  /**
   * Show multi-device banner
   * @default true
   */
  showMultiDeviceBanner?: boolean

  /**
   * Enable automatic reminders
   * @default true
   */
  enableReminders?: boolean
}

/**
 * Upgrade Provider Component
 *
 * @example
 * ```typescript
 * // In your main App component
 * function App() {
 *   return (
 *     <UpgradeProvider>
 *       <YourApp />
 *     </UpgradeProvider>
 *   )
 * }
 *
 * // In any component
 * function MyComponent() {
 *   const { showUpgrade } = useUpgrade()
 *
 *   const handleLimitReached = () => {
 *     showUpgrade('connections')
 *   }
 * }
 * ```
 */
export function UpgradeProvider({
  children,
  showMultiDeviceBanner = true,
  enableReminders = true,
}: UpgradeProviderProps) {
  const [isOpen, setIsOpen] = useState(false)
  const [currentTrigger, setCurrentTrigger] = useState<UpgradeTrigger>('manual')
  const { shouldShowPrompt } = useUpgradePromptStore()

  // Initialize upgrade systems
  useEffect(() => {
    if (enableReminders) {
      initializeReminderSystem()

      // Register listeners for reminder events
      registerReminderListeners((trigger) => {
        if (shouldShowPrompt(trigger)) {
          setCurrentTrigger(trigger)
          setIsOpen(true)
        }
      })
    }
  }, [enableReminders, shouldShowPrompt])

  const showUpgrade = useCallback(
    (trigger: UpgradeTrigger) => {
      if (shouldShowPrompt(trigger)) {
        setCurrentTrigger(trigger)
        setIsOpen(true)
      }
    },
    [shouldShowPrompt]
  )

  const contextValue: UpgradeContextValue = {
    showUpgrade,
    isUpgradeModalOpen: isOpen,
  }

  return (
    <UpgradeContext.Provider value={contextValue}>
      {/* Multi-device banner (top of app) */}
      {showMultiDeviceBanner && <MultiDeviceBanner position="top" />}

      {/* Main content */}
      {children}

      {/* Global upgrade modal */}
      <UpgradeModal
        open={isOpen}
        onOpenChange={setIsOpen}
        trigger={currentTrigger}
      />
    </UpgradeContext.Provider>
  )
}
