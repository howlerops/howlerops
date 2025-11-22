import { CheckCircle2,Database } from "lucide-react"
import { useState } from "react"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

interface ConnectionStepProps {
  onNext: () => void
  onBack: () => void
  onSkip: () => void
}

type DatabaseType = "sqlite" | "postgresql" | "mysql" | "skip"

export function ConnectionStep({ onNext, onBack, onSkip }: ConnectionStepProps) {
  const [selectedType, setSelectedType] = useState<DatabaseType | null>(null)
  const [isConnecting, setIsConnecting] = useState(false)
  const [isConnected, setIsConnected] = useState(false)

  const databaseOptions = [
    {
      type: "sqlite" as DatabaseType,
      name: "SQLite",
      description: "Perfect for getting started - no setup required",
      difficulty: "Easiest",
      recommended: true,
    },
    {
      type: "postgresql" as DatabaseType,
      name: "PostgreSQL",
      description: "Connect to your PostgreSQL database",
      difficulty: "Easy",
    },
    {
      type: "mysql" as DatabaseType,
      name: "MySQL",
      description: "Connect to your MySQL database",
      difficulty: "Easy",
    },
  ]

  const handleConnect = async (type: DatabaseType) => {
    setSelectedType(type)
    setIsConnecting(true)

    // Simulate connection attempt
    await new Promise((resolve) => setTimeout(resolve, 1500))

    setIsConnecting(false)
    setIsConnected(true)

    // Auto-advance after showing success
    setTimeout(() => {
      onNext()
    }, 1000)
  }

  const handleSkipConnection = () => {
    setSelectedType("skip")
    onSkip()
  }

  return (
    <div className="max-w-2xl mx-auto space-y-6 py-8">
      <div className="text-center space-y-2 mb-8">
        <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center mx-auto mb-4">
          <Database className="w-8 h-8 text-primary" />
        </div>
        <h2 className="text-2xl font-bold">Connect your first database</h2>
        <p className="text-muted-foreground">
          Choose a database type to get started
        </p>
      </div>

      <div className="grid gap-4">
        {databaseOptions.map((option) => (
          <button
            key={option.type}
            onClick={() => handleConnect(option.type)}
            disabled={isConnecting || isConnected}
            className={cn(
              "relative p-6 rounded-lg border-2 text-left transition-all hover:border-primary/50 hover:shadow-md",
              selectedType === option.type && isConnecting
                ? "border-primary bg-primary/5"
                : "border-border",
              selectedType === option.type && isConnected
                ? "border-green-500 bg-green-50 dark:bg-green-950"
                : "",
              option.recommended ? "ring-2 ring-primary/20" : ""
            )}
          >
            {option.recommended && (
              <div className="absolute -top-3 left-4 px-2 py-0.5 bg-primary text-primary-foreground text-xs font-medium rounded">
                Recommended
              </div>
            )}

            <div className="flex items-start justify-between">
              <div className="flex-1">
                <div className="flex items-center gap-3 mb-2">
                  <Database className="w-6 h-6 text-primary" />
                  <h3 className="font-semibold text-lg">{option.name}</h3>
                  <span className="px-2 py-0.5 bg-muted text-muted-foreground text-xs rounded">
                    {option.difficulty}
                  </span>
                </div>
                <p className="text-sm text-muted-foreground">
                  {option.description}
                </p>
              </div>

              {selectedType === option.type && (
                <div className="ml-4">
                  {isConnecting && (
                    <div className="w-6 h-6 border-2 border-primary border-t-transparent rounded-full animate-spin" />
                  )}
                  {isConnected && (
                    <CheckCircle2 className="w-6 h-6 text-green-500" />
                  )}
                </div>
              )}
            </div>
          </button>
        ))}
      </div>

      {isConnected && (
        <div className="p-4 rounded-lg bg-green-50 dark:bg-green-950 border border-green-200 dark:border-green-800 text-center">
          <CheckCircle2 className="w-8 h-8 text-green-500 mx-auto mb-2" />
          <p className="text-sm font-medium text-green-900 dark:text-green-100">
            Successfully connected! Moving to the next step...
          </p>
        </div>
      )}

      <div className="flex items-center gap-3 pt-4">
        <Button variant="outline" onClick={onBack} disabled={isConnecting}>
          Back
        </Button>
        <Button
          variant="ghost"
          onClick={handleSkipConnection}
          disabled={isConnecting || isConnected}
          className="flex-1"
        >
          Skip for now
        </Button>
      </div>

      <p className="text-xs text-center text-muted-foreground">
        Don't worry, you can add more connections later from Settings
      </p>
    </div>
  )
}
