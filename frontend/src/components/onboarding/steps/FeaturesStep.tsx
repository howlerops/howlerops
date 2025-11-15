import { Button } from "@/components/ui/button";
import {
  FileText,
  Users,
  Cloud,
  Bot,
  Zap,
  Shield,
  ArrowRight,
} from "lucide-react";

interface FeaturesStepProps {
  onNext: () => void;
  onBack: () => void;
}

const features = [
  {
    icon: FileText,
    title: "Query Templates",
    description:
      "Save time with reusable query templates. Create your own or use from our library.",
    color: "text-blue-500",
    bgColor: "bg-blue-50 dark:bg-blue-950",
  },
  {
    icon: Cloud,
    title: "Cloud Sync",
    description:
      "Access your queries and connections from any device with automatic cloud synchronization.",
    color: "text-purple-500",
    bgColor: "bg-purple-50 dark:bg-purple-950",
  },
  {
    icon: Users,
    title: "Team Collaboration",
    description:
      "Share queries, templates, and connections with your team. Work together seamlessly.",
    color: "text-green-500",
    bgColor: "bg-green-50 dark:bg-green-950",
  },
  {
    icon: Bot,
    title: "AI Query Assistant",
    description:
      "Get intelligent suggestions and optimizations. Write queries faster with AI help.",
    color: "text-orange-500",
    bgColor: "bg-orange-50 dark:bg-orange-950",
  },
  {
    icon: Zap,
    title: "Performance Insights",
    description:
      "Monitor query performance, identify bottlenecks, and optimize your database operations.",
    color: "text-yellow-500",
    bgColor: "bg-yellow-50 dark:bg-yellow-950",
  },
  {
    icon: Shield,
    title: "Secure by Default",
    description:
      "Enterprise-grade security with encrypted connections and granular access controls.",
    color: "text-red-500",
    bgColor: "bg-red-50 dark:bg-red-950",
  },
];

export function FeaturesStep({ onNext, onBack }: FeaturesStepProps) {
  return (
    <div className="max-w-5xl mx-auto space-y-6 py-8">
      <div className="text-center space-y-2 mb-8">
        <h2 className="text-2xl font-bold">Discover Powerful Features</h2>
        <p className="text-muted-foreground">
          Howlerops is packed with features to supercharge your workflow
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {features.map((feature, index) => {
          const Icon = feature.icon;
          return (
            <div
              key={index}
              className="group p-6 rounded-lg border-2 border-border hover:border-primary/50 transition-all hover:shadow-lg"
            >
              <div
                className={`w-12 h-12 rounded-lg ${feature.bgColor} flex items-center justify-center mb-4 group-hover:scale-110 transition-transform`}
              >
                <Icon className={`w-6 h-6 ${feature.color}`} />
              </div>
              <h3 className="font-semibold text-lg mb-2">{feature.title}</h3>
              <p className="text-sm text-muted-foreground">
                {feature.description}
              </p>
            </div>
          );
        })}
      </div>

      <div className="text-center pt-4">
        <p className="text-sm text-muted-foreground mb-6">
          And there's much more to explore as you use Howlerops!
        </p>
      </div>

      <div className="flex items-center gap-3">
        <Button variant="outline" onClick={onBack}>
          Back
        </Button>
        <Button onClick={onNext} className="flex-1">
          Continue
          <ArrowRight className="ml-2 h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}
