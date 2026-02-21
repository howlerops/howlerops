import {
  ArrowRight,
  Bot,
  Braces,
  ChevronRight,
  Cloud,
  Code2,
  ExternalLink,
  GitBranch,
  Github,
  Globe,
  Layers,
  Sparkles,
  Terminal,
  Zap,
} from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import { cn } from '@/lib/utils'

// ---------------------------------------------------------------------------
// Data
// ---------------------------------------------------------------------------

const FEATURES = [
  {
    icon: GitBranch,
    title: 'Evolutionary Agent Groups',
    description:
      'Agents evolve together in groups, sharing experience across lineages. Each generation benefits from the collective knowledge of all prior runs.',
  },
  {
    icon: Globe,
    title: 'Multi-LLM Support',
    description:
      'Works with any LLM provider via LiteLLM — Claude, GPT-4o, Llama, Gemini, and more. Swap models without changing your workflow.',
  },
  {
    icon: Braces,
    title: 'MCP Integration',
    description:
      'Use Howler Agents directly from Claude Code, Cursor, and OpenCode via the Model Context Protocol. One command to connect.',
  },
  {
    icon: Cloud,
    title: 'Local or Cloud',
    description:
      'Run the full evolution loop locally with in-memory storage, or deploy to Kubernetes with PostgreSQL/pgvector and Redis for scale.',
  },
  {
    icon: Zap,
    title: 'Capability Vectors',
    description:
      'Binary probe vectors measure agent abilities across tasks, enabling novelty-based selection that prevents local optima collapse.',
  },
  {
    icon: Layers,
    title: 'Shared Experience Pool',
    description:
      'Cross-lineage knowledge transfer through structured execution traces. Every agent generation stands on the shoulders of its ancestors.',
  },
] as const

const HOW_IT_WORKS_STEPS = [
  { label: 'Population', description: 'Seed agents with diverse starting strategies' },
  { label: 'Groups', description: 'Form cohorts that share task context' },
  { label: 'Evaluate', description: 'Score each agent on real benchmark tasks' },
  { label: 'Select', description: 'Rank by performance and novelty vectors' },
  { label: 'Reproduce', description: 'Meta-LLM synthesises next-generation agents' },
  { label: 'Next Gen', description: 'Repeat with accumulated shared experience' },
] as const

const INTEGRATIONS = [
  {
    label: 'Claude Code / MCP',
    language: 'bash',
    code: `# Add Howler Agents as an MCP server in Claude Code
claude mcp add howler-agents -- howler-agents serve`,
  },
  {
    label: 'Python SDK',
    language: 'bash',
    code: `# Install the core library with MCP extras
pip install howler-agents-core[mcp]

# Run your first evolution
from howler_agents import Swarm
swarm = Swarm.from_preset("coding")
result = await swarm.evolve(generations=5)`,
  },
  {
    label: 'REST API',
    language: 'bash',
    code: `# Start an evolution run via the HTTP API
curl -X POST https://api.howlerops.com/v1/swarms \\
  -H "Authorization: Bearer $HOWLER_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"domain":"coding","generations":10}'`,
  },
  {
    label: 'CLI',
    language: 'bash',
    code: `# Evolve a coding agent swarm from the terminal
howler-agents evolve --domain coding --generations 10 --output ./results`,
  },
] as const

const ARCHITECTURE_LAYERS = [
  {
    title: 'Core Library (Python)',
    description: 'Evolution loop, tournament selection, experience pool, and novelty scoring.',
    tags: ['Python', 'asyncio'],
  },
  {
    title: 'Service Layer',
    description: 'FastAPI + gRPC with PostgreSQL/pgvector for experience storage and Redis for job queues.',
    tags: ['FastAPI', 'gRPC', 'pgvector', 'Redis'],
  },
  {
    title: 'Dashboard UI',
    description: 'Real-time evolution monitoring built with TanStack Start and Shadcn UI components.',
    tags: ['TanStack Start', 'Shadcn'],
  },
  {
    title: 'TypeScript SDK',
    description: 'Type-safe client for the service layer using ConnectRPC and TanStack Query.',
    tags: ['ConnectRPC', 'TanStack Query'],
  },
  {
    title: 'MCP Server',
    description: 'stdio/SSE transport adapter so any MCP-compatible editor can drive evolution runs.',
    tags: ['MCP', 'stdio', 'SSE'],
  },
] as const

const PRICING_TIERS = [
  {
    name: 'Open Source',
    price: 'Free',
    note: 'Self-hosted',
    features: [
      'Full evolution loop',
      'MCP server included',
      'In-memory or SQLite storage',
      'All LLM providers',
      'Community support',
    ],
    cta: 'Get Started',
    ctaHref: 'https://github.com/howler-agents/howler-agents',
    highlight: false,
  },
  {
    name: 'Cloud',
    price: 'Coming Soon',
    note: 'Managed service',
    features: [
      'Managed infrastructure',
      'API key access',
      'Scale-to-zero compute',
      'Hosted pgvector experience pool',
      'Priority support',
    ],
    cta: 'Join Waitlist',
    ctaHref: 'mailto:hello@howlerops.com',
    highlight: true,
  },
] as const

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

interface CodeBlockProps {
  code: string
}

function CodeBlock({ code }: CodeBlockProps) {
  return (
    <pre className="rounded-md border bg-muted px-4 py-3 text-xs leading-relaxed overflow-x-auto font-mono whitespace-pre">
      <code>{code}</code>
    </pre>
  )
}

interface SectionHeadingProps {
  title: string
  subtitle?: string
  className?: string
}

function SectionHeading({ title, subtitle, className }: SectionHeadingProps) {
  return (
    <div className={cn('space-y-2', className)}>
      <h2 className="text-2xl font-semibold tracking-tight">{title}</h2>
      {subtitle && (
        <p className="text-sm text-muted-foreground max-w-2xl">{subtitle}</p>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export function AgentsPage() {
  return (
    <div className="flex-1 overflow-y-auto">
      <div className="mx-auto max-w-5xl px-6 py-10 space-y-16">

        {/* ---------------------------------------------------------------- */}
        {/* Hero                                                              */}
        {/* ---------------------------------------------------------------- */}
        <section className="space-y-6">
          <div className="flex items-center gap-2">
            <Badge variant="secondary" className="text-xs">
              arXiv:2602.04837
            </Badge>
            <Badge variant="outline" className="text-xs">
              Group-Evolving Agents
            </Badge>
          </div>

          <div className="space-y-3">
            <div className="flex items-center gap-3">
              <Bot className="h-8 w-8 text-primary flex-shrink-0" />
              <h1 className="text-4xl font-semibold tracking-tight">Howler Agents</h1>
            </div>
            <p className="text-lg text-muted-foreground">
              Group-Evolving AI Agents — agents that evolve together
            </p>
          </div>

          <p className="text-sm text-muted-foreground max-w-2xl leading-relaxed">
            Based on the GEA architecture (arXiv:2602.04837). Groups of AI agents share experience
            across lineages via a structured pool, enabling rapid capability growth without
            catastrophic forgetting. Achieves{' '}
            <span className="font-medium text-foreground">71% on SWE-bench</span> and{' '}
            <span className="font-medium text-foreground">88.3% on Polyglot</span> — state-of-the-art
            results for fully automated agent evolution.
          </p>

          <div className="flex items-center gap-3 flex-wrap">
            <Button asChild size="sm">
              <a
                href="http://209.38.173.33/docs-site"
                target="_blank"
                rel="noopener noreferrer"
              >
                Get Started
                <ArrowRight className="ml-2 h-4 w-4" />
              </a>
            </Button>
            <Button asChild variant="outline" size="sm">
              <a
                href="https://github.com/howler-agents/howler-agents"
                target="_blank"
                rel="noopener noreferrer"
              >
                <Github className="mr-2 h-4 w-4" />
                View on GitHub
                <ExternalLink className="ml-2 h-3 w-3 text-muted-foreground" />
              </a>
            </Button>
          </div>

          {/* Stats row */}
          <div className="grid grid-cols-3 gap-4 pt-2">
            {[
              { value: '71%', label: 'SWE-bench' },
              { value: '88.3%', label: 'Polyglot' },
              { value: 'Any LLM', label: 'via LiteLLM' },
            ].map((stat) => (
              <div
                key={stat.label}
                className="rounded-lg border bg-card px-4 py-3 text-center"
              >
                <p className="text-xl font-semibold text-primary">{stat.value}</p>
                <p className="text-xs text-muted-foreground mt-0.5">{stat.label}</p>
              </div>
            ))}
          </div>
        </section>

        <Separator />

        {/* ---------------------------------------------------------------- */}
        {/* Key Features                                                      */}
        {/* ---------------------------------------------------------------- */}
        <section className="space-y-6">
          <SectionHeading
            title="Key Features"
            subtitle="Everything you need to run and deploy evolving agent populations at any scale."
          />

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {FEATURES.map((feature) => {
              const Icon = feature.icon
              return (
                <Card key={feature.title} className="flex flex-col">
                  <CardHeader className="pb-2">
                    <div className="mb-2 flex h-8 w-8 items-center justify-center rounded-md border bg-muted">
                      <Icon className="h-4 w-4 text-primary" />
                    </div>
                    <CardTitle className="text-sm font-semibold">{feature.title}</CardTitle>
                  </CardHeader>
                  <CardContent className="flex-1">
                    <CardDescription className="text-xs leading-relaxed">
                      {feature.description}
                    </CardDescription>
                  </CardContent>
                </Card>
              )
            })}
          </div>
        </section>

        <Separator />

        {/* ---------------------------------------------------------------- */}
        {/* How It Works                                                      */}
        {/* ---------------------------------------------------------------- */}
        <section className="space-y-6">
          <SectionHeading
            title="How It Works"
            subtitle="The GEA evolution loop runs continuously, selecting and reproducing agents based on task performance and capability novelty."
          />

          {/* Flow diagram */}
          <div className="rounded-lg border bg-card p-6">
            <div className="flex flex-wrap items-start gap-2">
              {HOW_IT_WORKS_STEPS.map((step, index) => (
                <div key={step.label} className="flex items-start gap-2">
                  <div className="flex flex-col items-center gap-1">
                    <div className="flex h-8 w-8 items-center justify-center rounded-full border bg-muted text-xs font-semibold text-muted-foreground flex-shrink-0">
                      {index + 1}
                    </div>
                    <div className="text-center max-w-[90px]">
                      <p className="text-xs font-medium">{step.label}</p>
                      <p className="text-[10px] text-muted-foreground leading-tight mt-0.5">
                        {step.description}
                      </p>
                    </div>
                  </div>
                  {index < HOW_IT_WORKS_STEPS.length - 1 && (
                    <ChevronRight className="h-4 w-4 text-muted-foreground mt-2 flex-shrink-0" />
                  )}
                </div>
              ))}
            </div>
          </div>

          {/* Selection detail */}
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm">Performance Selection</CardTitle>
              </CardHeader>
              <CardContent>
                <CardDescription className="text-xs leading-relaxed">
                  Agents are scored on benchmark tasks drawn from the configured domain. The top
                  performers by raw score advance to the reproduction stage.
                </CardDescription>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm">Novelty Selection</CardTitle>
              </CardHeader>
              <CardContent>
                <CardDescription className="text-xs leading-relaxed">
                  Binary capability probe vectors are compared across the population. Agents with
                  strategies that differ significantly from the current archive are rewarded,
                  maintaining diversity across generations.
                </CardDescription>
              </CardContent>
            </Card>
          </div>
        </section>

        <Separator />

        {/* ---------------------------------------------------------------- */}
        {/* Integration                                                        */}
        {/* ---------------------------------------------------------------- */}
        <section className="space-y-6">
          <SectionHeading
            title="Integrations"
            subtitle="Connect Howler Agents to your existing workflow in minutes."
          />

          <div className="space-y-4">
            {INTEGRATIONS.map((integration) => (
              <div key={integration.label} className="space-y-2">
                <div className="flex items-center gap-2">
                  {integration.label === 'Claude Code / MCP' && (
                    <Sparkles className="h-4 w-4 text-primary" />
                  )}
                  {integration.label === 'Python SDK' && (
                    <Code2 className="h-4 w-4 text-primary" />
                  )}
                  {integration.label === 'REST API' && (
                    <Globe className="h-4 w-4 text-primary" />
                  )}
                  {integration.label === 'CLI' && (
                    <Terminal className="h-4 w-4 text-primary" />
                  )}
                  <span className="text-sm font-medium">{integration.label}</span>
                </div>
                <CodeBlock code={integration.code} />
              </div>
            ))}
          </div>
        </section>

        <Separator />

        {/* ---------------------------------------------------------------- */}
        {/* Architecture                                                       */}
        {/* ---------------------------------------------------------------- */}
        <section className="space-y-6">
          <SectionHeading
            title="Architecture"
            subtitle="A layered design that scales from a single laptop to a distributed Kubernetes cluster."
          />

          <div className="space-y-3">
            {ARCHITECTURE_LAYERS.map((layer, index) => (
              <div
                key={layer.title}
                className="flex items-start gap-4 rounded-lg border bg-card px-5 py-4"
              >
                <div className="flex h-6 w-6 flex-shrink-0 items-center justify-center rounded border bg-muted text-xs font-semibold text-muted-foreground">
                  {index + 1}
                </div>
                <div className="flex-1 min-w-0 space-y-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="text-sm font-medium">{layer.title}</span>
                    {layer.tags.map((tag) => (
                      <Badge key={tag} variant="secondary" className="text-[10px] px-1.5 py-0">
                        {tag}
                      </Badge>
                    ))}
                  </div>
                  <p className="text-xs text-muted-foreground">{layer.description}</p>
                </div>
              </div>
            ))}
          </div>
        </section>

        <Separator />

        {/* ---------------------------------------------------------------- */}
        {/* Pricing                                                            */}
        {/* ---------------------------------------------------------------- */}
        <section className="space-y-6">
          <SectionHeading
            title="Pricing"
            subtitle="Start for free with the self-hosted open-source edition. Managed cloud coming soon."
          />

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            {PRICING_TIERS.map((tier) => (
              <Card
                key={tier.name}
                className={cn(
                  'flex flex-col',
                  tier.highlight && 'border-primary/50 bg-accent/20'
                )}
              >
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-base">{tier.name}</CardTitle>
                    {tier.highlight && (
                      <Badge variant="secondary" className="text-xs">
                        Coming Soon
                      </Badge>
                    )}
                  </div>
                  <div className="flex items-baseline gap-1 pt-1">
                    <span className="text-2xl font-semibold">{tier.price}</span>
                    <span className="text-xs text-muted-foreground">/ {tier.note}</span>
                  </div>
                </CardHeader>
                <CardContent className="flex-1 space-y-4">
                  <ul className="space-y-2">
                    {tier.features.map((feature) => (
                      <li key={feature} className="flex items-center gap-2 text-xs">
                        <span className="h-1 w-1 rounded-full bg-primary flex-shrink-0" />
                        {feature}
                      </li>
                    ))}
                  </ul>
                  <Button
                    asChild
                    variant={tier.highlight ? 'default' : 'outline'}
                    size="sm"
                    className="w-full"
                    disabled={tier.highlight}
                  >
                    <a
                      href={tier.ctaHref}
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      {tier.cta}
                    </a>
                  </Button>
                </CardContent>
              </Card>
            ))}
          </div>
        </section>

        {/* ---------------------------------------------------------------- */}
        {/* Footer CTA                                                         */}
        {/* ---------------------------------------------------------------- */}
        <section>
          <div className="rounded-lg border bg-card px-6 py-8 text-center space-y-4">
            <Bot className="mx-auto h-10 w-10 text-primary" />
            <div className="space-y-1">
              <h3 className="text-lg font-semibold">Ready to evolve?</h3>
              <p className="text-sm text-muted-foreground">
                Read the docs or explore the source code to get started in minutes.
              </p>
            </div>
            <div className="flex items-center justify-center gap-3 flex-wrap">
              <Button asChild size="sm">
                <a
                  href="http://209.38.173.33/docs-site"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  Read the Docs
                  <ArrowRight className="ml-2 h-4 w-4" />
                </a>
              </Button>
              <Button asChild variant="outline" size="sm">
                <a
                  href="https://github.com/howler-agents/howler-agents"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  <Github className="mr-2 h-4 w-4" />
                  GitHub
                </a>
              </Button>
            </div>
          </div>
        </section>

      </div>
    </div>
  )
}

export default AgentsPage
