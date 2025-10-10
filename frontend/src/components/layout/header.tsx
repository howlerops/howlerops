import { Link, useLocation } from "react-router-dom"
import { Button } from "@/components/ui/button"
import { Switch } from "@/components/ui/switch"
import { useTheme } from "@/hooks/use-theme"
import { useConnectionStore } from "@/store/connection-store"
import { useQueryStore } from "@/store/query-store"
import { Moon, Sun, Plus, Settings } from "lucide-react"
import { cn } from "@/lib/utils"
import { useNavigate } from "react-router-dom"
import { HowlerOpsIcon } from "@/components/ui/HowlerOpsIcon"

export function Header() {
  const location = useLocation()
  const navigate = useNavigate()
  const { theme, setTheme } = useTheme()
  const { activeConnection } = useConnectionStore()
  const { createTab } = useQueryStore()

  const toggleTheme = () => {
    setTheme(theme === "light" ? "dark" : "light")
  }

  const handleNewQuery = () => {
    createTab()
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
          <div className="text-sm text-muted-foreground">
            {activeConnection ? (
              <span className="flex items-center space-x-2">
                <div className="h-2 w-2 rounded-full bg-green-500" />
                <span>{activeConnection.name}</span>
              </span>
            ) : (
              <span>No connection</span>
            )}
          </div>

          <Button variant="outline" size="sm" onClick={handleNewQuery}>
            <Plus className="h-4 w-4 mr-2" />
            New Query
          </Button>

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
