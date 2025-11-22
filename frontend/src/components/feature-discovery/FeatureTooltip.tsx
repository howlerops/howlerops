import { Sparkles,X } from "lucide-react"
import { ReactNode,useEffect, useState } from "react"

import { Button } from "@/components/ui/button"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"
import { onboardingTracker } from "@/lib/analytics/onboarding-tracking"

interface FeatureTooltipProps {
  feature: string
  title: string
  description: string
  ctaText?: string
  ctaLink?: string
  dismissible?: boolean
  children: ReactNode
  onCtaClick?: () => void
}

const DISMISSED_FEATURES_KEY = "sql-studio-dismissed-features"

export function FeatureTooltip({
  feature,
  title,
  description,
  ctaText,
  ctaLink,
  dismissible = true,
  children,
  onCtaClick,
}: FeatureTooltipProps) {
  const [open, setOpen] = useState(false)
  const [isDismissed, setIsDismissed] = useState(false)

  useEffect(() => {
    const dismissed = localStorage.getItem(DISMISSED_FEATURES_KEY)
    const dismissedFeatures = dismissed ? JSON.parse(dismissed) : []

    if (dismissedFeatures.includes(feature)) {
      setIsDismissed(true)
    } else {
      // Auto-show after a delay
      const timer = setTimeout(() => {
        setOpen(true)
        onboardingTracker.trackFeatureDiscovered(feature)
      }, 1000)

      return () => clearTimeout(timer)
    }
  }, [feature])

  const handleDismiss = () => {
    const dismissed = localStorage.getItem(DISMISSED_FEATURES_KEY)
    const dismissedFeatures = dismissed ? JSON.parse(dismissed) : []

    if (!dismissedFeatures.includes(feature)) {
      dismissedFeatures.push(feature)
      localStorage.setItem(DISMISSED_FEATURES_KEY, JSON.stringify(dismissedFeatures))
    }

    setIsDismissed(true)
    setOpen(false)
  }

  const handleCtaClick = () => {
    onCtaClick?.()
    if (ctaLink) {
      window.location.href = ctaLink
    }
    handleDismiss()
  }

  if (isDismissed) {
    return <>{children}</>
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>{children}</PopoverTrigger>
      <PopoverContent
        side="top"
        align="center"
        className="w-80 p-0 overflow-hidden border-2 border-primary/50 shadow-lg"
      >
        <div className="bg-gradient-to-br from-primary/10 to-purple-500/10 p-4 space-y-3">
          <div className="flex items-start justify-between">
            <div className="flex items-center gap-2">
              <div className="w-8 h-8 rounded-full bg-primary/20 flex items-center justify-center">
                <Sparkles className="w-4 h-4 text-primary" />
              </div>
              <h3 className="font-semibold">{title}</h3>
            </div>
            {dismissible && (
              <Button
                variant="ghost"
                size="icon"
                onClick={handleDismiss}
                className="h-6 w-6 rounded-full"
              >
                <X className="h-3 w-3" />
              </Button>
            )}
          </div>

          <p className="text-sm text-muted-foreground">{description}</p>

          {ctaText && (
            <Button
              onClick={handleCtaClick}
              size="sm"
              className="w-full"
            >
              {ctaText}
            </Button>
          )}
        </div>
      </PopoverContent>
    </Popover>
  )
}
