import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Sidebar,
  FileText,
  Database,
  Settings,
  Play,
  ArrowRight,
} from "lucide-react";
import { cn } from "@/lib/utils";

interface TourStepProps {
  onNext: () => void;
  onBack: () => void;
}

const tourPoints = [
  {
    id: "sidebar",
    icon: Sidebar,
    title: "Sidebar Navigation",
    description: "Access all your connections, saved queries, and templates",
    position: "left",
  },
  {
    id: "editor",
    icon: Play,
    title: "Query Editor",
    description: "Write and execute SQL queries with syntax highlighting",
    position: "center",
  },
  {
    id: "schema",
    icon: Database,
    title: "Schema Explorer",
    description: "Browse tables, columns, and relationships visually",
    position: "right",
  },
  {
    id: "saved",
    icon: FileText,
    title: "Saved Queries",
    description: "Organize and access your frequently used queries",
    position: "left",
  },
  {
    id: "settings",
    icon: Settings,
    title: "Settings",
    description: "Customize your experience and manage your account",
    position: "right",
  },
];

export function TourStep({ onNext, onBack }: TourStepProps) {
  const [currentPoint, setCurrentPoint] = useState(0);
  const point = tourPoints[currentPoint];
  const Icon = point.icon;

  const handleNext = () => {
    if (currentPoint < tourPoints.length - 1) {
      setCurrentPoint(currentPoint + 1);
    } else {
      onNext();
    }
  };

  const handlePrevious = () => {
    if (currentPoint > 0) {
      setCurrentPoint(currentPoint - 1);
    }
  };

  return (
    <div className="max-w-4xl mx-auto space-y-8 py-8">
      <div className="text-center space-y-2">
        <h2 className="text-2xl font-bold">Quick Tour</h2>
        <p className="text-muted-foreground">
          Let's explore the main features of Howlerops
        </p>
      </div>

      {/* Simulated UI Preview */}
      <div className="relative border-2 border-border rounded-lg overflow-hidden bg-muted/30 h-96">
        {/* Sidebar highlight */}
        {point.position === "left" && (
          <div className="absolute left-0 top-0 bottom-0 w-64 bg-primary/10 border-r-4 border-primary animate-pulse" />
        )}

        {/* Center highlight */}
        {point.position === "center" && (
          <div className="absolute left-64 right-64 top-0 bottom-0 bg-primary/10 border-4 border-primary animate-pulse" />
        )}

        {/* Right panel highlight */}
        {point.position === "right" && (
          <div className="absolute right-0 top-0 bottom-0 w-64 bg-primary/10 border-l-4 border-primary animate-pulse" />
        )}

        {/* Simulated UI Elements */}
        <div className="flex h-full">
          <div className="w-64 border-r border-border bg-card p-4">
            <div className="space-y-2">
              <div className="h-8 bg-muted rounded" />
              <div className="h-6 bg-muted rounded w-3/4" />
              <div className="h-6 bg-muted rounded w-2/3" />
            </div>
          </div>
          <div className="flex-1 p-4">
            <div className="h-full bg-card rounded border border-border" />
          </div>
          <div className="w-64 border-l border-border bg-card p-4">
            <div className="space-y-2">
              <div className="h-6 bg-muted rounded" />
              <div className="h-6 bg-muted rounded w-3/4" />
              <div className="h-6 bg-muted rounded w-5/6" />
            </div>
          </div>
        </div>

        {/* Feature Callout */}
        <div className="absolute bottom-4 left-1/2 -translate-x-1/2 w-full max-w-md">
          <div className="bg-background border-2 border-primary rounded-lg p-6 shadow-xl">
            <div className="flex items-start gap-4">
              <div className="w-12 h-12 rounded-lg bg-primary/10 flex items-center justify-center flex-shrink-0">
                <Icon className="w-6 h-6 text-primary" />
              </div>
              <div className="flex-1">
                <h3 className="font-semibold text-lg mb-1">{point.title}</h3>
                <p className="text-sm text-muted-foreground">
                  {point.description}
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Progress dots */}
      <div className="flex justify-center gap-2">
        {tourPoints.map((_, index) => (
          <button
            key={index}
            onClick={() => setCurrentPoint(index)}
            className={cn(
              "w-2 h-2 rounded-full transition-all",
              index === currentPoint
                ? "bg-primary w-8"
                : "bg-muted hover:bg-muted-foreground/30"
            )}
            aria-label={`Go to tour point ${index + 1}`}
          />
        ))}
      </div>

      <div className="flex items-center justify-between gap-3">
        <Button
          variant="outline"
          onClick={currentPoint === 0 ? onBack : handlePrevious}
        >
          {currentPoint === 0 ? "Back" : "Previous"}
        </Button>

        <div className="text-sm text-muted-foreground">
          {currentPoint + 1} of {tourPoints.length}
        </div>

        <Button onClick={handleNext}>
          {currentPoint === tourPoints.length - 1 ? (
            "Continue"
          ) : (
            <>
              Next <ArrowRight className="ml-2 h-4 w-4" />
            </>
          )}
        </Button>
      </div>
    </div>
  );
}
