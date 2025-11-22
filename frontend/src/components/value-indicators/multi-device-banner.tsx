/**
 * Multi-Device Banner
 *
 * Detects new devices and shows a friendly banner encouraging upgrade for sync.
 * Dismissible for 30 days.
 */

import { AnimatePresence,motion } from 'framer-motion'
import { Cloud, Smartphone, Sparkles,X } from 'lucide-react'
import React, { useEffect,useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { useUpgradeModal } from '@/components/upgrade-modal'
import { cn } from '@/lib/utils'
import { useTierStore } from '@/store/tier-store'
import { DISMISSAL_DURATIONS,useUpgradePromptStore } from '@/store/upgrade-prompt-store'

export interface MultiDeviceBannerProps {
  /**
   * Position of the banner
   */
  position?: 'top' | 'bottom'

  /**
   * Additional CSS classes
   */
  className?: string
}

/**
 * Multi-Device Banner Component
 *
 * Automatically detects when user is on a new device and suggests upgrade for sync.
 *
 * @example
 * ```typescript
 * // In main layout
 * <MultiDeviceBanner position="top" />
 * ```
 */
export function MultiDeviceBanner({ position = 'top', className }: MultiDeviceBannerProps) {
  const { hasFeature } = useTierStore()
  const { isNewDevice, shouldShowPrompt, dismiss } = useUpgradePromptStore()
  const { showUpgradeModal, UpgradeModalComponent } = useUpgradeModal()
  const [isVisible, setIsVisible] = useState(false)
  const [isClosing, setIsClosing] = useState(false)

  // Check if we should show the banner
  useEffect(() => {
    // Don't show if user already has multi-device feature
    if (hasFeature('multiDevice')) {
      return
    }

    // Check if this is a new device and prompt is allowed
    if (isNewDevice() && shouldShowPrompt('multiDevice')) {
      setIsVisible(true)
    }
  }, [hasFeature, isNewDevice, shouldShowPrompt])

  const handleUpgrade = () => {
    showUpgradeModal('multiDevice')
    setIsVisible(false)
  }

  const handleDismiss = () => {
    setIsClosing(true)
    setTimeout(() => {
      dismiss('multiDevice', DISMISSAL_DURATIONS.long) // Dismiss for 30 days
      setIsVisible(false)
      setIsClosing(false)
    }, 300)
  }

  if (!isVisible) {
    return <>{UpgradeModalComponent}</>
  }

  return (
    <>
      <AnimatePresence>
        {isVisible && (
          <motion.div
            initial={{ opacity: 0, y: position === 'top' ? -100 : 100 }}
            animate={{ opacity: isClosing ? 0 : 1, y: isClosing ? (position === 'top' ? -100 : 100) : 0 }}
            exit={{ opacity: 0, y: position === 'top' ? -100 : 100 }}
            transition={{ duration: 0.3 }}
            className={cn(
              'w-full bg-gradient-to-r from-blue-600 to-purple-600 text-white shadow-lg z-50',
              position === 'top' ? 'border-b' : 'border-t',
              className
            )}
          >
            <div className="container mx-auto px-4 py-3">
              <div className="flex items-center justify-between gap-4">
                {/* Left: Message */}
                <div className="flex items-center gap-3 flex-1">
                  <div className="p-2 bg-white/20 rounded-lg">
                    <Smartphone className="w-5 h-5" />
                  </div>
                  <div>
                    <p className="font-semibold text-sm flex items-center gap-2">
                      Welcome back! Working from a new device?
                      <Badge variant="secondary" className="bg-white/30 text-white border-white/40">
                        <Sparkles className="w-3 h-3 mr-1" />
                        New
                      </Badge>
                    </p>
                    <p className="text-xs text-white/90 mt-0.5">
                      Upgrade to sync your entire workspace automatically across all devices
                    </p>
                  </div>
                </div>

                {/* Center: Features */}
                <div className="hidden md:flex items-center gap-4">
                  <div className="flex items-center gap-2 text-sm">
                    <Cloud className="w-4 h-4" />
                    <span>Cloud Sync</span>
                  </div>
                  <div className="flex items-center gap-2 text-sm">
                    <Smartphone className="w-4 h-4" />
                    <span>Multi-Device</span>
                  </div>
                </div>

                {/* Right: Actions */}
                <div className="flex items-center gap-2">
                  <Button
                    size="sm"
                    variant="secondary"
                    className="bg-white text-blue-600 hover:bg-white/90 font-semibold"
                    onClick={handleUpgrade}
                  >
                    <Sparkles className="w-4 h-4 mr-2" />
                    Sync Your Workspace
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    className="text-white hover:bg-white/20"
                    onClick={handleDismiss}
                  >
                    <X className="w-4 h-4" />
                  </Button>
                </div>
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
      {UpgradeModalComponent}
    </>
  )
}
