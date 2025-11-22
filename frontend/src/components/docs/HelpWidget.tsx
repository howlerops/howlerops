import { HelpCircle } from "lucide-react"
import { useEffect,useState } from "react"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

import { HelpPanel } from "./HelpPanel"

interface HelpWidgetProps {
  className?: string
}

export function HelpWidget({ className }: HelpWidgetProps) {
  const [open, setOpen] = useState(false)

  // Keyboard shortcut: ? or Cmd+?
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (
        (e.key === "?" && !e.metaKey && !e.ctrlKey) ||
        ((e.metaKey || e.ctrlKey) && e.key === "/")
      ) {
        e.preventDefault()
        setOpen((prev) => !prev)
      }
    }

    window.addEventListener("keydown", handleKeyDown)
    return () => window.removeEventListener("keydown", handleKeyDown)
  }, [])

  return (
    <>
      <Button
        onClick={() => setOpen(true)}
        size="icon"
        variant="outline"
        className={cn(
          "fixed bottom-6 right-6 z-40 h-12 w-12 rounded-full shadow-lg hover:shadow-xl transition-all",
          className
        )}
        aria-label="Open help"
      >
        <HelpCircle className="h-5 w-5" />
      </Button>

      <HelpPanel open={open} onOpenChange={setOpen} />
    </>
  )
}
