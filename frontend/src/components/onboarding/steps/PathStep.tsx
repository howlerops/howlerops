import {
  Bot,
  CheckCircle2,
  Compass,
  FileText,
  Sparkles,
  Users,
} from "lucide-react"
import { useState } from "react"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

interface PathStepProps {
  onComplete: (path: string) => void
  onBack: () => void
}

const paths = [
  {
    id: "explore",
    icon: Compass,
    title: "Explore on my own",
    description: "I'll discover features as I go",
    color: "text-blue-500",
    bgColor: "bg-blue-50 dark:bg-blue-950",
    borderColor: "border-blue-200 dark:border-blue-800",
  },
  {
    id: "templates",
    icon: FileText,
    title: "Show me query templates",
    description: "Start with ready-to-use templates",
    color: "text-green-500",
    bgColor: "bg-green-50 dark:bg-green-950",
    borderColor: "border-green-200 dark:border-green-800",
  },
  {
    id: "ai",
    icon: Bot,
    title: "Help me write a query",
    description: "Use AI assistance to get started",
    color: "text-purple-500",
    bgColor: "bg-purple-50 dark:bg-purple-950",
    borderColor: "border-purple-200 dark:border-purple-800",
  },
  {
    id: "team",
    icon: Users,
    title: "Invite my team",
    description: "Set up team collaboration",
    color: "text-orange-500",
    bgColor: "bg-orange-50 dark:bg-orange-950",
    borderColor: "border-orange-200 dark:border-orange-800",
  },
]

export function PathStep({ onComplete, onBack }: PathStepProps) {
  const [selectedPath, setSelectedPath] = useState<string | null>(null)

  const handleSelectPath = (pathId: string) => {
    setSelectedPath(pathId)
    // Small delay for visual feedback before completing
    setTimeout(() => {
      onComplete(pathId)
    }, 300)
  }

  return (
    <div className="max-w-4xl mx-auto space-y-6 py-8">
      <div className="text-center space-y-2 mb-8">
        <div className="w-16 h-16 rounded-full bg-gradient-to-br from-primary to-purple-500 flex items-center justify-center mx-auto mb-4">
          <Sparkles className="w-8 h-8 text-white" />
        </div>
        <h2 className="text-2xl font-bold">You're all set!</h2>
        <p className="text-muted-foreground text-lg">
          Choose your next step to continue your journey
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {paths.map((path) => {
          const Icon = path.icon
          const isSelected = selectedPath === path.id

          return (
            <button
              key={path.id}
              onClick={() => handleSelectPath(path.id)}
              disabled={selectedPath !== null}
              className={cn(
                "relative p-6 rounded-lg border-2 text-left transition-all hover:shadow-lg group",
                isSelected
                  ? `${path.borderColor} ${path.bgColor} scale-105`
                  : "border-border hover:border-primary/50"
              )}
            >
              {isSelected && (
                <div className="absolute -top-2 -right-2 w-8 h-8 rounded-full bg-green-500 flex items-center justify-center shadow-lg">
                  <CheckCircle2 className="w-5 h-5 text-white" />
                </div>
              )}

              <div
                className={cn(
                  "w-12 h-12 rounded-lg flex items-center justify-center mb-4 transition-transform group-hover:scale-110",
                  isSelected ? path.bgColor : "bg-muted"
                )}
              >
                <Icon className={cn("w-6 h-6", isSelected ? path.color : "text-muted-foreground")} />
              </div>

              <h3 className="font-semibold text-lg mb-2">{path.title}</h3>
              <p className="text-sm text-muted-foreground">
                {path.description}
              </p>
            </button>
          )
        })}
      </div>

      <div className="text-center pt-6">
        <p className="text-sm text-muted-foreground">
          Don't worry, you can access all features anytime from the main menu
        </p>
      </div>

      <div className="flex items-center justify-center gap-3">
        <Button
          variant="outline"
          onClick={onBack}
          disabled={selectedPath !== null}
        >
          Back
        </Button>
      </div>
    </div>
  )
}
