import { Link, useLocation } from "react-router-dom"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Switch } from "@/components/ui/switch"
import { useTheme } from "@/hooks/use-theme"
import { useQueryStore } from "@/store/query-store"
import { useConnectionStore } from "@/store/connection-store"
import { useAIConfig } from "@/store/ai-store"
import { useAIQueryAgentStore } from "@/store/ai-query-agent-store"
import { Moon, Sun, Plus, Settings, Sparkles, Database } from "lucide-react"
import { cn } from "@/lib/utils"
import { useNavigate } from "react-router-dom"
import { HowlerOpsIcon } from "@/components/ui/howlerops-icon"

export function Header() {
  const location = useLocation()
  const navigate = useNavigate()
  const { theme, setTheme } = useTheme()
  const { createTab } = useQueryStore()
  const setActiveTab = useQueryStore(state => state.setActiveTab)
  const { activeConnection } = useConnectionStore()
  const { config: aiConfig } = useAIConfig()
  const createAgentSession = useAIQueryAgentStore(state => state.createSession)
  const setActiveAgentSession = useAIQueryAgentStore(state => state.setActiveSession)

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
    <header className="sticky top-0 z-50 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="flex h-14 items-center px-4">
        <div className="flex items-center space-x-6">
          <Link to="/dashboard" className="flex items-center space-x-2">
            <HowlerOpsIcon size={24} variant={theme === "dark" ? "dark" : "light"} />
            <h1 className="text-lg font-semibold">HowlerOps</h1>
          </Link>

          <nav className="flex items-center space-x-4">
            <Link
              to="/dashboard"
              className={cn(
                "text-sm font-medium transition-colors hover:text-primary",
                location.pathname === "/dashboard"
                  ? "text-foreground"
                  : "text-muted-foreground"
              )}
            >
              Dashboard
            </Link>
            <Link
              to="/connections"
              className={cn(
                "text-sm font-medium transition-colors hover:text-primary",
                location.pathname === "/connections"
                  ? "text-foreground"
                  : "text-muted-foreground"
              )}
            >
              Connections
            </Link>
          </nav>
        </div>

        <div className="ml-auto flex items-center space-x-4">

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="sm">
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

          <Button
            variant="ghost"
            size="sm"
            onClick={() => {
              navigate('/settings');
            }}
            title="Settings"
          >
            <Settings className="h-4 w-4" />
          </Button>

          <div className="flex items-center space-x-2">
            <Sun className="h-4 w-4" />
            <Switch
              checked={theme === "dark"}
              onCheckedChange={toggleTheme}
              aria-label="Toggle theme"
            />
            <Moon className="h-4 w-4" />
          </div>
        </div>
      </div>
    </header>
  )
}
