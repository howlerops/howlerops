import { useState } from "react";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from "@/components/ui/sheet";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { Badge } from "@/components/ui/badge";
import {
  Search,
  BookOpen,
  Video,
  MessageCircle,
  ExternalLink,
  FileText,
  Zap,
  Database,
  Users,
  Settings,
} from "lucide-react";
import { onboardingTracker } from "@/lib/analytics/onboarding-tracking";
import { cn } from "@/lib/utils";

interface HelpPanelProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

interface HelpArticle {
  id: string;
  title: string;
  category: string;
  icon: React.ElementType;
  description: string;
  url?: string;
}

const helpArticles: HelpArticle[] = [
  {
    id: "getting-started",
    title: "Getting Started Guide",
    category: "Basics",
    icon: BookOpen,
    description: "Learn the fundamentals of Howlerops",
  },
  {
    id: "query-editor",
    title: "Using the Query Editor",
    category: "Basics",
    icon: FileText,
    description: "Write and execute SQL queries efficiently",
  },
  {
    id: "connections",
    title: "Managing Database Connections",
    category: "Databases",
    icon: Database,
    description: "Connect to PostgreSQL, MySQL, SQLite, and more",
  },
  {
    id: "query-templates",
    title: "Working with Query Templates",
    category: "Queries",
    icon: Zap,
    description: "Create and use reusable query templates",
  },
  {
    id: "collaboration",
    title: "Team Collaboration",
    category: "Teams",
    icon: Users,
    description: "Share queries and collaborate with your team",
  },
  {
    id: "settings",
    title: "Customizing Your Settings",
    category: "Settings",
    icon: Settings,
    description: "Personalize your Howlerops experience",
  },
];

const popularArticles = helpArticles.slice(0, 4);

export function HelpPanel({ open, onOpenChange }: HelpPanelProps) {
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedCategory, setSelectedCategory] = useState<string | null>(null);

  const categories = Array.from(new Set(helpArticles.map((a) => a.category)));

  const filteredArticles = helpArticles.filter((article) => {
    const matchesSearch =
      searchQuery === "" ||
      article.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
      article.description.toLowerCase().includes(searchQuery.toLowerCase());

    const matchesCategory =
      !selectedCategory || article.category === selectedCategory;

    return matchesSearch && matchesCategory;
  });

  const handleSearch = (query: string) => {
    setSearchQuery(query);
    if (query) {
      onboardingTracker.trackHelpSearched(query);
    }
  };

  const handleArticleClick = (article: HelpArticle) => {
    // In a real implementation, this would open the article
    console.log("Opening article:", article.id);
  };

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-full sm:max-w-lg p-0">
        <SheetHeader className="p-6 pb-4">
          <SheetTitle>Help & Documentation</SheetTitle>
          <SheetDescription>
            Find answers and learn how to use Howlerops
          </SheetDescription>
        </SheetHeader>

        <div className="px-6 pb-4">
          {/* Search */}
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Search documentation..."
              value={searchQuery}
              onChange={(e) => handleSearch(e.target.value)}
              className="pl-9"
            />
          </div>
        </div>

        <ScrollArea className="h-[calc(100vh-12rem)] px-6">
          <div className="space-y-6 pb-6">
            {/* Category filters */}
            {searchQuery === "" && (
              <div className="flex flex-wrap gap-2">
                <Badge
                  variant={selectedCategory === null ? "default" : "outline"}
                  className="cursor-pointer"
                  onClick={() => setSelectedCategory(null)}
                >
                  All
                </Badge>
                {categories.map((category) => (
                  <Badge
                    key={category}
                    variant={
                      selectedCategory === category ? "default" : "outline"
                    }
                    className="cursor-pointer"
                    onClick={() => setSelectedCategory(category)}
                  >
                    {category}
                  </Badge>
                ))}
              </div>
            )}

            {/* Popular Articles */}
            {searchQuery === "" && selectedCategory === null && (
              <div>
                <h3 className="text-sm font-semibold mb-3">Popular Articles</h3>
                <div className="space-y-2">
                  {popularArticles.map((article) => (
                    <ArticleItem
                      key={article.id}
                      article={article}
                      onClick={() => handleArticleClick(article)}
                    />
                  ))}
                </div>
              </div>
            )}

            <Separator />

            {/* All Articles */}
            <div>
              <h3 className="text-sm font-semibold mb-3">
                {searchQuery ? "Search Results" : "All Articles"}
              </h3>
              <div className="space-y-2">
                {filteredArticles.map((article) => (
                  <ArticleItem
                    key={article.id}
                    article={article}
                    onClick={() => handleArticleClick(article)}
                  />
                ))}
              </div>

              {filteredArticles.length === 0 && (
                <div className="text-center py-8 text-muted-foreground">
                  <Search className="h-8 w-8 mx-auto mb-2 opacity-50" />
                  <p className="text-sm">No articles found</p>
                </div>
              )}
            </div>

            <Separator />

            {/* Additional Resources */}
            <div>
              <h3 className="text-sm font-semibold mb-3">More Resources</h3>
              <div className="space-y-2">
                <Button
                  variant="outline"
                  className="w-full justify-start gap-2"
                  onClick={() => console.log("Open videos")}
                >
                  <Video className="h-4 w-4" />
                  Video Tutorials
                </Button>
                <Button
                  variant="outline"
                  className="w-full justify-start gap-2"
                  onClick={() =>
                    window.open("https://community.sqlstudio.com", "_blank")
                  }
                >
                  <MessageCircle className="h-4 w-4" />
                  Community Forum
                  <ExternalLink className="h-3 w-3 ml-auto" />
                </Button>
                <Button
                  variant="outline"
                  className="w-full justify-start gap-2"
                  onClick={() => console.log("Contact support")}
                >
                  <MessageCircle className="h-4 w-4" />
                  Contact Support
                </Button>
              </div>
            </div>
          </div>
        </ScrollArea>

        {/* Footer */}
        <div className="border-t p-4 text-center">
          <p className="text-xs text-muted-foreground">
            Press <kbd className="px-1.5 py-0.5 bg-muted rounded">?</kbd> or{" "}
            <kbd className="px-1.5 py-0.5 bg-muted rounded">Cmd+/</kbd> to open
            help anytime
          </p>
        </div>
      </SheetContent>
    </Sheet>
  );
}

interface ArticleItemProps {
  article: HelpArticle;
  onClick: () => void;
}

function ArticleItem({ article, onClick }: ArticleItemProps) {
  const Icon = article.icon;

  return (
    <button
      onClick={onClick}
      className={cn(
        "w-full flex items-start gap-3 p-3 rounded-lg text-left transition-colors",
        "hover:bg-muted cursor-pointer"
      )}
    >
      <div className="w-8 h-8 rounded-lg bg-primary/10 flex items-center justify-center flex-shrink-0">
        <Icon className="w-4 h-4 text-primary" />
      </div>
      <div className="flex-1 min-w-0">
        <h4 className="font-medium text-sm mb-1">{article.title}</h4>
        <p className="text-xs text-muted-foreground line-clamp-2">
          {article.description}
        </p>
      </div>
    </button>
  );
}
