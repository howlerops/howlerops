import { Tutorial } from "@/types/tutorial"
import { queryEditorBasicsTutorial } from "./query-editor-basics"
import { savedQueriesTutorial } from "./saved-queries"

// Additional tutorial stubs (implement these similarly)
const queryTemplatesTutorial: Tutorial = {
  id: "query-templates",
  name: "Query Templates Guide",
  description: "Create and use reusable query templates with parameters",
  category: "queries",
  difficulty: "intermediate",
  estimatedMinutes: 6,
  steps: [],
}

const teamCollaborationTutorial: Tutorial = {
  id: "team-collaboration",
  name: "Team Collaboration",
  description: "Learn how to collaborate with your team on queries and databases",
  category: "collaboration",
  difficulty: "intermediate",
  estimatedMinutes: 5,
  steps: [],
}

const cloudSyncTutorial: Tutorial = {
  id: "cloud-sync",
  name: "Cloud Sync Setup",
  description: "Enable and manage cloud synchronization across devices",
  category: "advanced",
  difficulty: "beginner",
  estimatedMinutes: 3,
  steps: [],
}

const aiAssistantTutorial: Tutorial = {
  id: "ai-assistant",
  name: "AI Query Assistant",
  description: "Use AI to write better queries faster",
  category: "ai",
  difficulty: "beginner",
  estimatedMinutes: 4,
  steps: [],
}

export const allTutorials: Tutorial[] = [
  queryEditorBasicsTutorial,
  savedQueriesTutorial,
  queryTemplatesTutorial,
  teamCollaborationTutorial,
  cloudSyncTutorial,
  aiAssistantTutorial,
]

export function getTutorialById(id: string): Tutorial | undefined {
  return allTutorials.find((t) => t.id === id)
}

export function getTutorialsByCategory(category: string): Tutorial[] {
  return allTutorials.filter((t) => t.category === category)
}

export {
  queryEditorBasicsTutorial,
  savedQueriesTutorial,
  queryTemplatesTutorial,
  teamCollaborationTutorial,
  cloudSyncTutorial,
  aiAssistantTutorial,
}
