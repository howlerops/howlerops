import { useState, useEffect } from "react"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Sparkles, ArrowRight, ExternalLink } from "lucide-react"
import { Badge } from "@/components/ui/badge"

interface Feature {
  title: string
  description: string
  icon?: React.ReactNode
  badge?: string
}

interface FeatureAnnouncementProps {
  version: string
  title?: string
  features: Feature[]
  changelogUrl?: string
  onTakeTour?: () => void
}

const LAST_SEEN_VERSION_KEY = "sql-studio-last-seen-version"

export function FeatureAnnouncement({
  version,
  title = "What's New",
  features,
  changelogUrl,
  onTakeTour,
}: FeatureAnnouncementProps) {
  const [open, setOpen] = useState(false)

  useEffect(() => {
    const lastSeenVersion = localStorage.getItem(LAST_SEEN_VERSION_KEY)

    if (lastSeenVersion !== version) {
      setOpen(true)
    }
  }, [version])

  const handleClose = () => {
    localStorage.setItem(LAST_SEEN_VERSION_KEY, version)
    setOpen(false)
  }

  const handleTakeTour = () => {
    handleClose()
    onTakeTour?.()
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <div className="flex items-center gap-3 mb-2">
            <div className="w-12 h-12 rounded-full bg-gradient-to-br from-primary to-purple-500 flex items-center justify-center">
              <Sparkles className="w-6 h-6 text-white" />
            </div>
            <div>
              <DialogTitle className="text-2xl">{title}</DialogTitle>
              <DialogDescription>Version {version}</DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <div className="space-y-6 py-4">
          {features.map((feature, index) => (
            <div key={index} className="flex items-start gap-4">
              {feature.icon && (
                <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center flex-shrink-0">
                  {feature.icon}
                </div>
              )}
              <div className="flex-1">
                <div className="flex items-center gap-2 mb-1">
                  <h3 className="font-semibold">{feature.title}</h3>
                  {feature.badge && (
                    <Badge variant="secondary">{feature.badge}</Badge>
                  )}
                </div>
                <p className="text-sm text-muted-foreground">
                  {feature.description}
                </p>
              </div>
            </div>
          ))}
        </div>

        <div className="flex items-center gap-3 pt-4 border-t">
          {changelogUrl && (
            <Button
              variant="outline"
              onClick={() => window.open(changelogUrl, "_blank")}
              className="gap-2"
            >
              Full Changelog
              <ExternalLink className="h-4 w-4" />
            </Button>
          )}
          {onTakeTour && (
            <Button onClick={handleTakeTour} className="flex-1 gap-2">
              Take a Tour
              <ArrowRight className="h-4 w-4" />
            </Button>
          )}
          {!onTakeTour && (
            <Button onClick={handleClose} className="flex-1">
              Got it
            </Button>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
