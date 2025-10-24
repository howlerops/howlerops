import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import { LucideIcon } from "lucide-react"

interface EmptyStateProps {
  icon?: LucideIcon
  illustration?: string
  title: string
  description: string
  primaryAction?: {
    label: string
    onClick: () => void
  }
  secondaryAction?: {
    label: string
    onClick: () => void
  }
  className?: string
}

export function EmptyState({
  icon: Icon,
  illustration,
  title,
  description,
  primaryAction,
  secondaryAction,
  className,
}: EmptyStateProps) {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center text-center py-12 px-4",
        className
      )}
    >
      {/* Illustration or Icon */}
      {illustration ? (
        <img
          src={illustration}
          alt={title}
          className="w-64 h-64 object-contain mb-6 opacity-80"
        />
      ) : Icon ? (
        <div className="w-24 h-24 rounded-full bg-muted flex items-center justify-center mb-6">
          <Icon className="w-12 h-12 text-muted-foreground" />
        </div>
      ) : null}

      {/* Content */}
      <div className="max-w-md space-y-2 mb-6">
        <h3 className="text-xl font-semibold">{title}</h3>
        <p className="text-muted-foreground">{description}</p>
      </div>

      {/* Actions */}
      <div className="flex items-center gap-3">
        {primaryAction && (
          <Button onClick={primaryAction.onClick}>
            {primaryAction.label}
          </Button>
        )}
        {secondaryAction && (
          <Button variant="outline" onClick={secondaryAction.onClick}>
            {secondaryAction.label}
          </Button>
        )}
      </div>
    </div>
  )
}

// Predefined empty states for common scenarios
import {
  Database,
  FileText,
  Users,
  FileCode,
  Search,
  Inbox,
} from "lucide-react"

export const emptyStates = {
  noConnections: {
    icon: Database,
    title: "No database connections",
    description: "Connect your first database to start running queries and exploring your data.",
  },
  noSavedQueries: {
    icon: FileText,
    title: "No saved queries yet",
    description: "Save your queries to access them anytime and share them with your team.",
  },
  noTemplates: {
    icon: FileCode,
    title: "No query templates",
    description: "Create reusable query templates to speed up your workflow.",
  },
  noTeamMembers: {
    icon: Users,
    title: "No team members",
    description: "Invite your team to collaborate on queries, templates, and databases.",
  },
  noResults: {
    icon: Inbox,
    title: "No results found",
    description: "Your query returned no results. Try adjusting your filters or query conditions.",
  },
  noSearchResults: {
    icon: Search,
    title: "No matches found",
    description: "We couldn't find anything matching your search. Try different keywords.",
  },
}
