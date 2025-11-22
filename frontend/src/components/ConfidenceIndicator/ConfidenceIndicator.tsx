import { AlertCircle, AlertTriangle, CheckCircle2 } from "lucide-react"
import { memo } from "react"

import { Badge } from "@/components/ui/badge"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { cn } from "@/lib/utils"

export type ConfidenceLevel = "high" | "medium" | "low"

interface ConfidenceIndicatorProps {
  confidence: number
  className?: string
  showLabel?: boolean
  size?: "sm" | "md" | "lg"
}

function getConfidenceLevel(confidence: number): ConfidenceLevel {
  if (confidence > 0.8) {
    return "high"
  }
  if (confidence >= 0.5) {
    return "medium"
  }
  return "low"
}

function getConfidenceConfig(level: ConfidenceLevel) {
  switch (level) {
    case "high":
      return {
        icon: CheckCircle2,
        color: "text-green-600 dark:text-green-500",
        bgColor: "bg-green-50 dark:bg-green-950",
        borderColor: "border-green-200 dark:border-green-800",
        label: "High confidence",
        description: "Safe to execute - AI is confident in this result",
      }
    case "medium":
      return {
        icon: AlertTriangle,
        color: "text-yellow-600 dark:text-yellow-500",
        bgColor: "bg-yellow-50 dark:bg-yellow-950",
        borderColor: "border-yellow-200 dark:border-yellow-800",
        label: "Medium confidence",
        description: "Review before executing - verify this result",
      }
    case "low":
      return {
        icon: AlertCircle,
        color: "text-red-600 dark:text-red-500",
        bgColor: "bg-red-50 dark:bg-red-950",
        borderColor: "border-red-200 dark:border-red-800",
        label: "Low confidence",
        description: "Verify carefully - AI is uncertain about this result",
      }
  }
}

function getSizeClasses(size: "sm" | "md" | "lg") {
  switch (size) {
    case "sm":
      return {
        icon: "h-3 w-3",
        text: "text-xs",
        badge: "text-[10px] px-1.5 py-0.5",
      }
    case "md":
      return {
        icon: "h-4 w-4",
        text: "text-sm",
        badge: "text-xs px-2 py-1",
      }
    case "lg":
      return {
        icon: "h-5 w-5",
        text: "text-base",
        badge: "text-sm px-2.5 py-1.5",
      }
  }
}

export const ConfidenceIndicator = memo(function ConfidenceIndicator({
  confidence,
  className,
  showLabel = true,
  size = "md",
}: ConfidenceIndicatorProps) {
  const level = getConfidenceLevel(confidence)
  const config = getConfidenceConfig(level)
  const sizeClasses = getSizeClasses(size)
  const Icon = config.icon

  const percentage = Math.round(confidence * 100)

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <div className={cn("inline-flex items-center gap-1.5", className)}>
            <Icon className={cn(sizeClasses.icon, config.color)} />
            {showLabel && (
              <Badge
                variant="outline"
                className={cn(
                  sizeClasses.badge,
                  config.bgColor,
                  config.borderColor,
                  config.color,
                  "font-medium"
                )}
              >
                {percentage}% confidence
              </Badge>
            )}
          </div>
        </TooltipTrigger>
        <TooltipContent>
          <div className="space-y-1">
            <p className="font-medium">{config.label}</p>
            <p className="text-xs text-muted-foreground">{config.description}</p>
          </div>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
})
