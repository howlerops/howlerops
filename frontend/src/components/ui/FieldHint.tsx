import { AlertCircle, Info } from "lucide-react"
import * as React from "react"

import { cn } from "@/lib/utils"

export interface FieldHintProps extends React.HTMLAttributes<HTMLDivElement> {
  children: React.ReactNode
  variant?: "default" | "error" | "warning"
  icon?: boolean
}

export function FieldHint({
  children,
  variant = "default",
  icon = true,
  className,
  ...props
}: FieldHintProps) {
  const Icon = variant === "error" || variant === "warning" ? AlertCircle : Info

  return (
    <div
      className={cn(
        "flex items-start gap-2 text-sm mt-1.5",
        variant === "default" && "text-muted-foreground",
        variant === "error" && "text-destructive",
        variant === "warning" && "text-amber-600 dark:text-amber-500",
        className
      )}
      {...props}
    >
      {icon && <Icon className="h-4 w-4 mt-0.5 flex-shrink-0" />}
      <div className="flex-1">{children}</div>
    </div>
  )
}
