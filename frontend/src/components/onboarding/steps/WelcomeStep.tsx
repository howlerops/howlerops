import { Cloud,Database, Users, Zap } from "lucide-react";

import { Button } from "@/components/ui/button";

interface WelcomeStepProps {
  onNext: () => void;
  onSkip: () => void;
}

export function WelcomeStep({ onNext, onSkip }: WelcomeStepProps) {
  return (
    <div className="flex flex-col items-center text-center space-y-6 py-8">
      <div className="w-20 h-20 rounded-full bg-primary/10 flex items-center justify-center">
        <Database className="w-10 h-10 text-primary" />
      </div>

      <div className="space-y-2">
        <h2 className="text-3xl font-bold">Welcome to Howlerops</h2>
        <p className="text-muted-foreground text-lg max-w-md mx-auto">
          Your powerful database management tool that makes working with data a
          breeze.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 w-full max-w-2xl mt-8">
        <div className="flex items-start gap-3 p-4 rounded-lg border bg-card">
          <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center flex-shrink-0">
            <Zap className="w-5 h-5 text-primary" />
          </div>
          <div className="text-left">
            <h3 className="font-semibold mb-1">Lightning Fast</h3>
            <p className="text-sm text-muted-foreground">
              Run queries and get results instantly with our optimized engine
            </p>
          </div>
        </div>

        <div className="flex items-start gap-3 p-4 rounded-lg border bg-card">
          <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center flex-shrink-0">
            <Users className="w-5 h-5 text-primary" />
          </div>
          <div className="text-left">
            <h3 className="font-semibold mb-1">Team Collaboration</h3>
            <p className="text-sm text-muted-foreground">
              Share queries, templates, and connections with your team
            </p>
          </div>
        </div>

        <div className="flex items-start gap-3 p-4 rounded-lg border bg-card">
          <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center flex-shrink-0">
            <Cloud className="w-5 h-5 text-primary" />
          </div>
          <div className="text-left">
            <h3 className="font-semibold mb-1">Cloud Sync</h3>
            <p className="text-sm text-muted-foreground">
              Access your work anywhere with automatic cloud synchronization
            </p>
          </div>
        </div>

        <div className="flex items-start gap-3 p-4 rounded-lg border bg-card">
          <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center flex-shrink-0">
            <Database className="w-5 h-5 text-primary" />
          </div>
          <div className="text-left">
            <h3 className="font-semibold mb-1">Multi-Database</h3>
            <p className="text-sm text-muted-foreground">
              Connect to PostgreSQL, MySQL, SQLite, and more
            </p>
          </div>
        </div>
      </div>

      <div className="flex items-center gap-3 pt-4">
        <Button variant="outline" onClick={onSkip}>
          Skip Setup
        </Button>
        <Button onClick={onNext} size="lg">
          Get Started
          <span className="ml-2 text-xs text-muted-foreground">3 min</span>
        </Button>
      </div>
    </div>
  );
}
