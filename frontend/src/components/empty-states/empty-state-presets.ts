import {
  Database,
  FileCode,
  FileText,
  Inbox,
  Search,
  Users,
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
