import { HelpCircle } from "lucide-react"
import { ReactNode } from "react"

import { Button } from "@/components/ui/button"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"

interface QuickHelpProps {
  topic: string
  title?: string
  children?: ReactNode
}

// Help content for various topics
const helpContent: Record<string, { title: string; content: ReactNode }> = {
  "cron-expressions": {
    title: "Cron Expressions",
    content: (
      <div className="space-y-2 text-sm">
        <p>Schedule queries using cron syntax:</p>
        <div className="space-y-1 font-mono text-xs bg-muted p-2 rounded">
          <div>* * * * * - Every minute</div>
          <div>0 * * * * - Every hour</div>
          <div>0 0 * * * - Daily at midnight</div>
          <div>0 0 * * 1 - Every Monday</div>
          <div>0 9 * * 1-5 - Weekdays at 9am</div>
        </div>
        <p className="text-muted-foreground">
          Format: minute hour day month weekday
        </p>
      </div>
    ),
  },
  "sql-parameters": {
    title: "SQL Parameters",
    content: (
      <div className="space-y-2 text-sm">
        <p>Use parameters to create dynamic queries:</p>
        <div className="space-y-1 font-mono text-xs bg-muted p-2 rounded">
          <div>{"{{user_id}} - Simple parameter"}</div>
          <div>{"{{start_date:date}} - Typed parameter"}</div>
          <div>{"{{status:select}} - Dropdown"}</div>
        </div>
        <p className="text-muted-foreground">
          Parameters make queries reusable with different values
        </p>
      </div>
    ),
  },
  "connection-strings": {
    title: "Connection Strings",
    content: (
      <div className="space-y-2 text-sm">
        <p>Database connection string formats:</p>
        <div className="space-y-2 font-mono text-xs bg-muted p-2 rounded">
          <div>
            <strong>PostgreSQL:</strong>
            <br />
            postgresql://user:pass@host:5432/db
          </div>
          <div>
            <strong>MySQL:</strong>
            <br />
            mysql://user:pass@host:3306/db
          </div>
          <div>
            <strong>SQLite:</strong>
            <br />
            sqlite:///path/to/database.db
          </div>
        </div>
      </div>
    ),
  },
  "query-sharing": {
    title: "Sharing Queries",
    content: (
      <div className="space-y-2 text-sm">
        <p>Share queries with your team:</p>
        <ul className="list-disc list-inside space-y-1 text-muted-foreground">
          <li>
            <strong>View-only:</strong> Team can see and run the query
          </li>
          <li>
            <strong>Can edit:</strong> Team can modify the query
          </li>
          <li>
            <strong>Organization:</strong> Share with entire org
          </li>
        </ul>
        <p className="text-muted-foreground">
          Shared queries sync automatically across all devices
        </p>
      </div>
    ),
  },
}

export function QuickHelp({ topic, title, children }: QuickHelpProps) {
  const content = helpContent[topic]

  if (!content && !children) {
    console.warn(`No help content found for topic: ${topic}`)
    return null
  }

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className="h-6 w-6 rounded-full"
          aria-label="Help"
        >
          <HelpCircle className="h-4 w-4 text-muted-foreground" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-80" side="top">
        <div className="space-y-2">
          <h4 className="font-semibold">{title || content?.title}</h4>
          {children || content?.content}
        </div>
      </PopoverContent>
    </Popover>
  )
}
