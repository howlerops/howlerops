import { ReactNode,useEffect, useState } from "react"

import { Tutorial } from "@/types/tutorial"

import { TutorialEngine } from "./TutorialEngine"
import { getTutorialById } from "./tutorials"

interface TutorialTriggerProps {
  tutorialId: string
  trigger: "page_visit" | "feature_use" | "manual"
  maxTriggerCount?: number
  delay?: number
  children: ReactNode
}

const TRIGGER_STORAGE_KEY = "sql-studio-tutorial-triggers"

export function TutorialTrigger({
  tutorialId,
  trigger,
  maxTriggerCount = 1,
  delay = 2000,
  children,
}: TutorialTriggerProps) {
  const [showTutorial, setShowTutorial] = useState(false)
  const [tutorial, setTutorial] = useState<Tutorial | null>(null)

  useEffect(() => {
    const savedTriggers = localStorage.getItem(TRIGGER_STORAGE_KEY)
    const triggers = savedTriggers ? JSON.parse(savedTriggers) : {}

    const currentCount = triggers[tutorialId] || 0

    // Check if we should trigger the tutorial
    if (currentCount < maxTriggerCount && trigger === "page_visit") {
      const timer = setTimeout(() => {
        const tutorialData = getTutorialById(tutorialId)
        if (tutorialData) {
          setTutorial(tutorialData)
          setShowTutorial(true)

          // Update trigger count
          triggers[tutorialId] = currentCount + 1
          localStorage.setItem(TRIGGER_STORAGE_KEY, JSON.stringify(triggers))
        }
      }, delay)

      return () => clearTimeout(timer)
    }
  }, [tutorialId, trigger, maxTriggerCount, delay])

  return (
    <>
      {children}
      {tutorial && (
        <TutorialEngine
          tutorial={tutorial}
          open={showTutorial}
          onOpenChange={setShowTutorial}
        />
      )}
    </>
  )
}
