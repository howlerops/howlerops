import { CheckCircle2, Play, Sparkles } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface FirstQueryStepProps {
  onNext: () => void;
  onBack: () => void;
}

const sampleQuery = `-- Your first query
SELECT
  name,
  email,
  created_at
FROM users
WHERE active = true
ORDER BY created_at DESC
LIMIT 10;`;

const mockResults = [
  {
    name: "Alice Johnson",
    email: "alice@example.com",
    created_at: "2024-01-15",
  },
  { name: "Bob Smith", email: "bob@example.com", created_at: "2024-01-14" },
  { name: "Carol White", email: "carol@example.com", created_at: "2024-01-13" },
];

export function FirstQueryStep({ onNext, onBack }: FirstQueryStepProps) {
  const [hasRun, setHasRun] = useState(false);
  const [isRunning, setIsRunning] = useState(false);
  const [showCelebration, setShowCelebration] = useState(false);

  const handleRunQuery = async () => {
    setIsRunning(true);

    // Simulate query execution
    await new Promise((resolve) => setTimeout(resolve, 1000));

    setIsRunning(false);
    setHasRun(true);
    setShowCelebration(true);

    // Hide celebration after animation
    setTimeout(() => setShowCelebration(false), 3000);
  };

  return (
    <div className="max-w-4xl mx-auto space-y-6 py-8">
      <div className="text-center space-y-2 mb-8">
        <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center mx-auto mb-4">
          <Play className="w-8 h-8 text-primary" />
        </div>
        <h2 className="text-2xl font-bold">Run your first query</h2>
        <p className="text-muted-foreground">
          Click the Run button to execute the sample query below
        </p>
      </div>

      {/* Query Editor Simulation */}
      <div className="border-2 border-border rounded-lg overflow-hidden">
        <div className="bg-muted/50 border-b border-border px-4 py-2 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <div className="text-sm font-medium">Query Editor</div>
            <span className="text-xs text-muted-foreground">sample.sql</span>
          </div>
          <Button
            onClick={handleRunQuery}
            disabled={isRunning || hasRun}
            size="sm"
            className="gap-2"
          >
            {isRunning ? (
              <>
                <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                Running...
              </>
            ) : hasRun ? (
              <>
                <CheckCircle2 className="w-4 h-4" />
                Executed
              </>
            ) : (
              <>
                <Play className="w-4 h-4" />
                Run Query
              </>
            )}
          </Button>
        </div>

        <div className="bg-slate-950 text-slate-50 p-4 font-mono text-sm">
          <pre className="whitespace-pre-wrap">{sampleQuery}</pre>
        </div>

        {hasRun && (
          <div className="border-t border-border">
            <div className="bg-muted/50 px-4 py-2 text-sm font-medium border-b border-border">
              Results (3 rows)
            </div>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-border bg-muted/30">
                    <th className="px-4 py-2 text-left text-sm font-medium">
                      name
                    </th>
                    <th className="px-4 py-2 text-left text-sm font-medium">
                      email
                    </th>
                    <th className="px-4 py-2 text-left text-sm font-medium">
                      created_at
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {mockResults.map((row, i) => (
                    <tr
                      key={i}
                      className={cn(
                        "border-b border-border",
                        i % 2 === 0 ? "bg-muted/10" : ""
                      )}
                    >
                      <td className="px-4 py-2 text-sm">{row.name}</td>
                      <td className="px-4 py-2 text-sm">{row.email}</td>
                      <td className="px-4 py-2 text-sm">{row.created_at}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>

      {/* Celebration Message */}
      {showCelebration && (
        <div className="p-6 rounded-lg bg-gradient-to-r from-green-50 to-emerald-50 dark:from-green-950 dark:to-emerald-950 border-2 border-green-200 dark:border-green-800 text-center animate-in slide-in-from-bottom">
          <div className="flex items-center justify-center gap-2 mb-2">
            <Sparkles className="w-6 h-6 text-green-600 dark:text-green-400" />
            <h3 className="text-xl font-bold text-green-900 dark:text-green-100">
              Great job!
            </h3>
            <Sparkles className="w-6 h-6 text-green-600 dark:text-green-400" />
          </div>
          <p className="text-green-800 dark:text-green-200">
            You've successfully run your first query. You're on your way to
            becoming a Howlerops pro!
          </p>
        </div>
      )}

      <div className="flex items-center gap-3">
        <Button variant="outline" onClick={onBack} disabled={isRunning}>
          Back
        </Button>
        <Button onClick={onNext} disabled={!hasRun} className="flex-1">
          Continue
        </Button>
      </div>
    </div>
  );
}
