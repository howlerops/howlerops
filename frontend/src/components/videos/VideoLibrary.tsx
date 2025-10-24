import { useState } from "react"
import { Input } from "@/components/ui/input"
import { Card } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Search, Play, Clock } from "lucide-react"
import { VideoPlayer } from "./VideoPlayer"
import { Dialog, DialogContent } from "@/components/ui/dialog"

interface Video {
  id: string
  title: string
  description: string
  duration: number // in seconds
  difficulty: "beginner" | "intermediate" | "advanced"
  thumbnail: string
  src: string
  category: string
}

const videos: Video[] = [
  {
    id: "getting-started",
    title: "Getting Started with SQL Studio",
    description: "Learn the basics and set up your first database connection",
    duration: 180, // 3 minutes
    difficulty: "beginner",
    thumbnail: "/thumbnails/getting-started.jpg",
    src: "/videos/getting-started.mp4",
    category: "Basics",
  },
  {
    id: "first-query",
    title: "Your First Query",
    description: "Write and execute your first SQL query",
    duration: 120, // 2 minutes
    difficulty: "beginner",
    thumbnail: "/thumbnails/first-query.jpg",
    src: "/videos/first-query.mp4",
    category: "Basics",
  },
  {
    id: "query-templates",
    title: "Working with Query Templates",
    description: "Create reusable query templates to save time",
    duration: 240, // 4 minutes
    difficulty: "intermediate",
    thumbnail: "/thumbnails/query-templates.jpg",
    src: "/videos/query-templates.mp4",
    category: "Advanced",
  },
  {
    id: "team-collaboration",
    title: "Team Collaboration Basics",
    description: "Share queries and databases with your team",
    duration: 300, // 5 minutes
    difficulty: "intermediate",
    thumbnail: "/thumbnails/team-collaboration.jpg",
    src: "/videos/team-collaboration.mp4",
    category: "Collaboration",
  },
  {
    id: "cloud-sync",
    title: "Cloud Sync Deep Dive",
    description: "Understanding and managing cloud synchronization",
    duration: 360, // 6 minutes
    difficulty: "advanced",
    thumbnail: "/thumbnails/cloud-sync.jpg",
    src: "/videos/cloud-sync.mp4",
    category: "Advanced",
  },
  {
    id: "tips-tricks",
    title: "Advanced Tips & Tricks",
    description: "Power user features to supercharge your workflow",
    duration: 420, // 7 minutes
    difficulty: "advanced",
    thumbnail: "/thumbnails/tips-tricks.jpg",
    src: "/videos/tips-tricks.mp4",
    category: "Advanced",
  },
]

const difficultyColors = {
  beginner: "bg-green-100 text-green-800 dark:bg-green-950 dark:text-green-300",
  intermediate: "bg-yellow-100 text-yellow-800 dark:bg-yellow-950 dark:text-yellow-300",
  advanced: "bg-red-100 text-red-800 dark:bg-red-950 dark:text-red-300",
}

export function VideoLibrary() {
  const [searchQuery, setSearchQuery] = useState("")
  const [selectedDifficulty, setSelectedDifficulty] = useState<string | null>(null)
  const [selectedVideo, setSelectedVideo] = useState<Video | null>(null)

  const difficulties = ["beginner", "intermediate", "advanced"]
  const categories = Array.from(new Set(videos.map((v) => v.category)))

  const filteredVideos = videos.filter((video) => {
    const matchesSearch =
      searchQuery === "" ||
      video.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
      video.description.toLowerCase().includes(searchQuery.toLowerCase())

    const matchesDifficulty =
      !selectedDifficulty || video.difficulty === selectedDifficulty

    return matchesSearch && matchesDifficulty
  })

  const formatDuration = (seconds: number) => {
    const mins = Math.floor(seconds / 60)
    const secs = seconds % 60
    return secs > 0 ? `${mins}m ${secs}s` : `${mins}m`
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold">Video Tutorials</h1>
        <p className="text-muted-foreground mt-2">
          Watch step-by-step video guides to master SQL Studio
        </p>
      </div>

      {/* Search and Filters */}
      <div className="space-y-4">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search videos..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>

        <div className="flex flex-wrap gap-2">
          <Badge
            variant={selectedDifficulty === null ? "default" : "outline"}
            className="cursor-pointer"
            onClick={() => setSelectedDifficulty(null)}
          >
            All Levels
          </Badge>
          {difficulties.map((difficulty) => (
            <Badge
              key={difficulty}
              variant={selectedDifficulty === difficulty ? "default" : "outline"}
              className="cursor-pointer capitalize"
              onClick={() => setSelectedDifficulty(difficulty)}
            >
              {difficulty}
            </Badge>
          ))}
        </div>
      </div>

      {/* Video Grid */}
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
        {filteredVideos.map((video) => (
          <Card
            key={video.id}
            className="group overflow-hidden hover:shadow-lg transition-shadow cursor-pointer"
            onClick={() => setSelectedVideo(video)}
          >
            {/* Thumbnail */}
            <div className="relative aspect-video bg-muted overflow-hidden">
              {/* Placeholder for thumbnail */}
              <div className="absolute inset-0 flex items-center justify-center bg-gradient-to-br from-primary/20 to-purple-500/20">
                <Play className="h-12 w-12 text-primary" />
              </div>

              {/* Play Overlay */}
              <div className="absolute inset-0 bg-black/0 group-hover:bg-black/40 transition-colors flex items-center justify-center">
                <div className="scale-0 group-hover:scale-100 transition-transform">
                  <div className="w-16 h-16 rounded-full bg-white/90 flex items-center justify-center">
                    <Play className="w-8 h-8 text-black ml-1" />
                  </div>
                </div>
              </div>

              {/* Duration Badge */}
              <div className="absolute bottom-2 right-2 px-2 py-1 bg-black/80 text-white text-xs rounded flex items-center gap-1">
                <Clock className="h-3 w-3" />
                {formatDuration(video.duration)}
              </div>
            </div>

            {/* Content */}
            <div className="p-4 space-y-3">
              <div>
                <h3 className="font-semibold line-clamp-1">{video.title}</h3>
                <p className="text-sm text-muted-foreground line-clamp-2 mt-1">
                  {video.description}
                </p>
              </div>

              <div className="flex items-center gap-2 flex-wrap">
                <Badge
                  variant="secondary"
                  className={difficultyColors[video.difficulty]}
                >
                  {video.difficulty}
                </Badge>
                <Badge variant="outline">{video.category}</Badge>
              </div>
            </div>
          </Card>
        ))}
      </div>

      {filteredVideos.length === 0 && (
        <div className="text-center py-12">
          <Search className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
          <h3 className="text-lg font-semibold mb-2">No videos found</h3>
          <p className="text-muted-foreground">
            Try adjusting your search or filter
          </p>
        </div>
      )}

      {/* Video Player Dialog */}
      <Dialog
        open={selectedVideo !== null}
        onOpenChange={(open) => !open && setSelectedVideo(null)}
      >
        <DialogContent className="max-w-4xl">
          {selectedVideo && (
            <div className="space-y-4">
              <div>
                <h2 className="text-2xl font-bold">{selectedVideo.title}</h2>
                <p className="text-muted-foreground mt-1">
                  {selectedVideo.description}
                </p>
              </div>

              <VideoPlayer
                videoId={selectedVideo.id}
                src={selectedVideo.src}
                title={selectedVideo.title}
              />

              <div className="flex items-center gap-2">
                <Badge
                  variant="secondary"
                  className={difficultyColors[selectedVideo.difficulty]}
                >
                  {selectedVideo.difficulty}
                </Badge>
                <Badge variant="outline">{selectedVideo.category}</Badge>
                <Badge variant="outline" className="gap-1">
                  <Clock className="h-3 w-3" />
                  {formatDuration(selectedVideo.duration)}
                </Badge>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
