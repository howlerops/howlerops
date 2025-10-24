import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Progress } from "@/components/ui/progress"
import {
  Play,
  CheckCircle2,
  Clock,
  Search,
  BookOpen,
  Users,
  Zap,
  Bot,
  TrendingUp,
} from "lucide-react"
import { allTutorials } from "./tutorials"
import { Tutorial, TutorialCategory } from "@/types/tutorial"
import { cn } from "@/lib/utils"

interface TutorialLibraryProps {
  onStartTutorial: (tutorial: Tutorial) => void
  completedTutorials?: string[]
}

const categoryIcons: Record<TutorialCategory, React.ElementType> = {
  basics: BookOpen,
  queries: Zap,
  collaboration: Users,
  advanced: TrendingUp,
  ai: Bot,
  optimization: TrendingUp,
}

const categoryLabels: Record<TutorialCategory, string> = {
  basics: "Basics",
  queries: "Queries",
  collaboration: "Collaboration",
  advanced: "Advanced",
  ai: "AI Assistant",
  optimization: "Optimization",
}

const difficultyColors = {
  beginner: "bg-green-100 text-green-800 dark:bg-green-950 dark:text-green-300",
  intermediate: "bg-yellow-100 text-yellow-800 dark:bg-yellow-950 dark:text-yellow-300",
  advanced: "bg-red-100 text-red-800 dark:bg-red-950 dark:text-red-300",
}

export function TutorialLibrary({
  onStartTutorial,
  completedTutorials = [],
}: TutorialLibraryProps) {
  const [searchQuery, setSearchQuery] = useState("")
  const [selectedCategory, setSelectedCategory] = useState<string>("all")

  const categories = Array.from(
    new Set(allTutorials.map((t) => t.category))
  ) as TutorialCategory[]

  const filteredTutorials = allTutorials.filter((tutorial) => {
    const matchesSearch =
      tutorial.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      tutorial.description.toLowerCase().includes(searchQuery.toLowerCase())

    const matchesCategory =
      selectedCategory === "all" || tutorial.category === selectedCategory

    return matchesSearch && matchesCategory
  })

  const tutorialsByCategory = categories.reduce((acc, category) => {
    acc[category] = filteredTutorials.filter((t) => t.category === category)
    return acc
  }, {} as Record<TutorialCategory, Tutorial[]>)

  const completionPercentage =
    (completedTutorials.length / allTutorials.length) * 100

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="space-y-4">
        <div>
          <h1 className="text-3xl font-bold">Tutorial Library</h1>
          <p className="text-muted-foreground mt-2">
            Master SQL Studio with our guided tutorials
          </p>
        </div>

        {/* Progress */}
        <Card className="p-4">
          <div className="flex items-center justify-between mb-3">
            <div className="flex items-center gap-2">
              <CheckCircle2 className="h-5 w-5 text-green-500" />
              <span className="font-medium">Your Progress</span>
            </div>
            <span className="text-sm text-muted-foreground">
              {completedTutorials.length} of {allTutorials.length} completed
            </span>
          </div>
          <Progress value={completionPercentage} className="h-2" />
        </Card>

        {/* Search */}
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search tutorials..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>
      </div>

      {/* Category Tabs */}
      <Tabs value={selectedCategory} onValueChange={setSelectedCategory}>
        <TabsList className="w-full justify-start overflow-x-auto">
          <TabsTrigger value="all">All Tutorials</TabsTrigger>
          {categories.map((category) => {
            const Icon = categoryIcons[category]
            return (
              <TabsTrigger key={category} value={category} className="gap-2">
                <Icon className="h-4 w-4" />
                {categoryLabels[category]}
              </TabsTrigger>
            )
          })}
        </TabsList>

        <TabsContent value="all" className="mt-6">
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {filteredTutorials.map((tutorial) => (
              <TutorialCard
                key={tutorial.id}
                tutorial={tutorial}
                isCompleted={completedTutorials.includes(tutorial.id)}
                onStart={() => onStartTutorial(tutorial)}
              />
            ))}
          </div>
        </TabsContent>

        {categories.map((category) => (
          <TabsContent key={category} value={category} className="mt-6">
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {tutorialsByCategory[category]?.map((tutorial) => (
                <TutorialCard
                  key={tutorial.id}
                  tutorial={tutorial}
                  isCompleted={completedTutorials.includes(tutorial.id)}
                  onStart={() => onStartTutorial(tutorial)}
                />
              ))}
            </div>
          </TabsContent>
        ))}
      </Tabs>

      {filteredTutorials.length === 0 && (
        <div className="text-center py-12">
          <BookOpen className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
          <h3 className="text-lg font-semibold mb-2">No tutorials found</h3>
          <p className="text-muted-foreground">
            Try adjusting your search or category filter
          </p>
        </div>
      )}
    </div>
  )
}

interface TutorialCardProps {
  tutorial: Tutorial
  isCompleted: boolean
  onStart: () => void
}

function TutorialCard({ tutorial, isCompleted, onStart }: TutorialCardProps) {
  const Icon = categoryIcons[tutorial.category]

  return (
    <Card
      className={cn(
        "group relative overflow-hidden transition-all hover:shadow-lg",
        isCompleted && "border-green-500/50 bg-green-50/50 dark:bg-green-950/20"
      )}
    >
      {isCompleted && (
        <div className="absolute top-2 right-2 z-10">
          <div className="rounded-full bg-green-500 p-1">
            <CheckCircle2 className="h-4 w-4 text-white" />
          </div>
        </div>
      )}

      <div className="p-6 space-y-4">
        <div className="flex items-start gap-3">
          <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center flex-shrink-0">
            <Icon className="w-5 h-5 text-primary" />
          </div>
          <div className="flex-1 min-w-0">
            <h3 className="font-semibold truncate">{tutorial.name}</h3>
            <p className="text-sm text-muted-foreground line-clamp-2 mt-1">
              {tutorial.description}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-2 flex-wrap">
          <Badge
            variant="secondary"
            className={difficultyColors[tutorial.difficulty]}
          >
            {tutorial.difficulty}
          </Badge>
          <Badge variant="outline" className="gap-1">
            <Clock className="h-3 w-3" />
            {tutorial.estimatedMinutes} min
          </Badge>
          <Badge variant="outline">{tutorial.steps.length} steps</Badge>
        </div>

        <Button
          onClick={onStart}
          className="w-full"
          variant={isCompleted ? "outline" : "default"}
        >
          {isCompleted ? (
            <>
              <Play className="h-4 w-4 mr-2" />
              Restart Tutorial
            </>
          ) : (
            <>
              <Play className="h-4 w-4 mr-2" />
              Start Tutorial
            </>
          )}
        </Button>
      </div>
    </Card>
  )
}
