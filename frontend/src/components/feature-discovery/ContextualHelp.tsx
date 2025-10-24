import { useState, useEffect } from "react"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { X, Lightbulb } from "lucide-react"
import { motion, AnimatePresence } from "framer-motion"

interface ContextualHelpProps {
  id: string
  title: string
  message: string
  actionText?: string
  onAction?: () => void
  trigger?: () => boolean
  delay?: number
}

const DISMISSED_HELP_KEY = "sql-studio-dismissed-help"

export function ContextualHelp({
  id,
  title,
  message,
  actionText,
  onAction,
  trigger,
  delay = 5000,
}: ContextualHelpProps) {
  const [show, setShow] = useState(false)
  const [isDismissed, setIsDismissed] = useState(false)

  useEffect(() => {
    const dismissed = localStorage.getItem(DISMISSED_HELP_KEY)
    const dismissedHelp = dismissed ? JSON.parse(dismissed) : []

    if (dismissedHelp.includes(id)) {
      setIsDismissed(true)
      return
    }

    const timer = setTimeout(() => {
      if (!trigger || trigger()) {
        setShow(true)
      }
    }, delay)

    return () => clearTimeout(timer)
  }, [id, trigger, delay])

  const handleDismiss = () => {
    const dismissed = localStorage.getItem(DISMISSED_HELP_KEY)
    const dismissedHelp = dismissed ? JSON.parse(dismissed) : []

    if (!dismissedHelp.includes(id)) {
      dismissedHelp.push(id)
      localStorage.setItem(DISMISSED_HELP_KEY, JSON.stringify(dismissedHelp))
    }

    setIsDismissed(true)
    setShow(false)
  }

  const handleAction = () => {
    onAction?.()
    handleDismiss()
  }

  if (isDismissed) {
    return null
  }

  return (
    <AnimatePresence>
      {show && (
        <motion.div
          initial={{ opacity: 0, y: 20, scale: 0.95 }}
          animate={{ opacity: 1, y: 0, scale: 1 }}
          exit={{ opacity: 0, y: 20, scale: 0.95 }}
          transition={{ duration: 0.2 }}
          className="fixed bottom-4 right-4 z-50 max-w-sm"
        >
          <Card className="p-4 border-2 border-amber-500/50 bg-amber-50 dark:bg-amber-950 shadow-lg">
            <div className="flex items-start gap-3">
              <div className="w-8 h-8 rounded-full bg-amber-500/20 flex items-center justify-center flex-shrink-0">
                <Lightbulb className="w-4 h-4 text-amber-600 dark:text-amber-400" />
              </div>

              <div className="flex-1 min-w-0">
                <div className="flex items-start justify-between mb-1">
                  <h3 className="font-semibold text-sm text-amber-900 dark:text-amber-100">
                    {title}
                  </h3>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={handleDismiss}
                    className="h-5 w-5 rounded-full -mt-1"
                  >
                    <X className="h-3 w-3" />
                  </Button>
                </div>

                <p className="text-sm text-amber-800 dark:text-amber-200 mb-3">
                  {message}
                </p>

                <div className="flex items-center gap-2">
                  {actionText && onAction && (
                    <Button
                      onClick={handleAction}
                      size="sm"
                      variant="default"
                      className="bg-amber-600 hover:bg-amber-700 text-white"
                    >
                      {actionText}
                    </Button>
                  )}
                  <Button
                    onClick={handleDismiss}
                    size="sm"
                    variant="ghost"
                    className="text-amber-900 dark:text-amber-100"
                  >
                    Dismiss
                  </Button>
                </div>
              </div>
            </div>
          </Card>
        </motion.div>
      )}
    </AnimatePresence>
  )
}

// Predefined contextual help suggestions
export const contextualHelpSuggestions = {
  slowQuery: {
    id: "slow-query-help",
    title: "Query taking a while?",
    message: "Try optimizing your query or adding indexes to improve performance.",
    actionText: "Learn More",
  },
  noSavedQueries: {
    id: "no-saved-queries-help",
    title: "Haven't saved any queries yet?",
    message: "Save your frequently used queries for quick access later.",
    actionText: "Learn How",
  },
  singlePlayer: {
    id: "single-player-help",
    title: "Working solo?",
    message: "Invite your team to collaborate on queries and databases.",
    actionText: "Invite Team",
  },
  noTemplates: {
    id: "no-templates-help",
    title: "Speed up your workflow",
    message: "Query templates can save you time on repetitive tasks.",
    actionText: "Browse Templates",
  },
}
