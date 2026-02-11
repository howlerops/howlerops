import { Database, Moon, Plus, Sparkles, Sun } from "lucide-react"
import { Link } from "react-router-dom"
import { useShallow } from "zustand/react/shallow"

import { AuthButton } from "@/components/auth/auth-button"
import { TierBadge } from "@/components/tier-badge"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { HowlerOpsIcon } from "@/components/ui/howlerops-icon"
import { Switch } from "@/components/ui/switch"
import { useTheme } from "@/hooks/use-theme"
import { useAIQueryAgentStore } from "@/store/ai-query-agent-store"
import { useAIConfig } from "@/store/ai-store"
import { useConnectionStore } from "@/store/connection-store"
import { useQueryStore } from "@/store/query-store"

export function Header() {
  const { theme, setTheme } = useTheme()
  const { createTab, setActiveTab } = useQueryStore(useShallow((state) => ({
    createTab: state.createTab,
    setActiveTab: state.setActiveTab,
  })))
  const { activeConnection } = useConnectionStore(useShallow((state) => ({
    activeConnection: state.activeConnection,
  })))
  const { config: aiConfig } = useAIConfig()
  const { createAgentSession, setActiveAgentSession } = useAIQueryAgentStore(useShallow((state) => ({
    createAgentSession: state.createSession,
    setActiveAgentSession: state.setActiveSession,
  })))

  const toggleTheme = () => {
    setTheme(theme === "light" ? "dark" : "light")
  }

  const handleNewSqlTab = () => {
    const tabId = createTab('New Query', {
      type: 'sql',
      connectionId: activeConnection?.id,
    })
    setActiveTab(tabId)
  }

  const handleNewAiTab = () => {
    const sessionId = createAgentSession({
      title: `AI Query ${new Date().toLocaleTimeString()}`,
      provider: aiConfig.provider,
      model: aiConfig.selectedModel,
    })
    const tabId = createTab('AI Query Agent', {
      type: 'ai',
      connectionId: activeConnection?.id,
      aiSessionId: sessionId,
    })
    setActiveAgentSession(sessionId)
    setActiveTab(tabId)
  }

  return (
    <header className="sticky top-0 z-50 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      {/* Native title bar drag zone — sits behind macOS traffic lights */}
      <div className="h-11" />
      {/* Header content — below the native title bar */}
      <div className="flex h-10 items-center border-b px-4 pb-2">
        {/* Logo */}
        <Link to="/dashboard" className="flex items-center space-x-2">
          <HowlerOpsIcon size={24} variant={theme === "dark" ? "dark" : "light"} />
          <h1 className="text-lg font-semibold">HowlerOps</h1>
        </Link>

        {/* Right side actions */}
        <div className="ml-auto flex items-center space-x-3">
          <AuthButton />

          <TierBadge variant="header" />

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="default" size="sm">
                <Plus className="h-4 w-4 mr-2" />
                New Query
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={handleNewSqlTab}>
                <Database className="h-4 w-4 mr-2" />
                SQL Editor Tab
              </DropdownMenuItem>
              <DropdownMenuItem onClick={handleNewAiTab}>
                <Sparkles className="h-4 w-4 mr-2" />
                AI Query Agent
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>

          <div className="flex items-center space-x-2 border-l pl-3 ml-1">
            <Sun className="h-4 w-4 text-muted-foreground" />
            <Switch
              checked={theme === "dark"}
              onCheckedChange={toggleTheme}
              aria-label="Toggle theme"
            />
            <Moon className="h-4 w-4 text-muted-foreground" />
          </div>
        </div>
      </div>
    </header>
  )
}
