import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Switch } from "@/components/ui/switch"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { PageErrorBoundary } from "@/components/page-error-boundary"
import { useTheme } from "@/hooks/use-theme"
import { ArrowLeft, Brain, Key, Server, AlertTriangle, Download, Play, CheckCircle } from "lucide-react"
import { useNavigate } from "react-router-dom"
import { useEffect, useState, useRef } from "react"
import { useAIConfig } from "@/store/ai-store"
import { useOllamaDetection } from "@/hooks/use-ollama-detection"
import { useToast } from "@/hooks/use-toast"

export function Settings() {
  const { theme, setTheme } = useTheme()
  const navigate = useNavigate()
  const { toast } = useToast()

  // AI Configuration
  const { config: aiConfig, updateConfig, testConnection, connectionStatus } = useAIConfig()
  const ollamaDetection = useOllamaDetection(aiConfig.provider === 'ollama')

  // Track initial config to detect changes
  const initialConfigRef = useRef(aiConfig)
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false)

  const [showApiKeys, setShowApiKeys] = useState({
    openai: false,
    anthropic: false,
    codex: false
  })
  const hasAutoTestedLocalRef = useRef(false)

  const [testMessage, setTestMessage] = useState<{provider: string, message: string, type: 'success' | 'error'} | null>(null)

  const handleAiConfigChange = (key: string, value: string | number | boolean) => {
    updateConfig({ [key]: value })

    if (key === 'provider') {
      const recommendedModelMap: Record<string, string> = {
        openai: 'gpt-4o-mini',
        anthropic: 'claude-3-5-sonnet-20241022',
        ollama: 'sqlcoder:7b',
        huggingface: 'sqlcoder:7b',
        claudecode: 'opus',
        codex: 'code-davinci-002',
      }

      const recommendedModel = recommendedModelMap[value as string]
      if (recommendedModel) {
        updateConfig({ selectedModel: recommendedModel })
      }
    }
  }

  const handleTestConnection = async (provider: string) => {
    setTestMessage(null) // Clear previous message
    try {
      const success = await testConnection(provider)
      if (success) {
        // Success notification is now handled by Wails dialog in the store
        // Optionally show inline success message as well
        const providerName = provider === 'claudecode' ? 'Claude Code' :
                           provider === 'codex' ? 'Codex' :
                           provider === 'huggingface' ? 'Hugging Face' :
                           provider === 'ollama' ? 'Ollama' :
                           provider === 'openai' ? 'OpenAI' :
                           provider === 'anthropic' ? 'Anthropic' : provider
        setTestMessage({
          provider,
          message: `${providerName} connection successful!`,
          type: 'success'
        })
        // Clear message after 3 seconds (shorter since we also have dialog)
        setTimeout(() => setTestMessage(null), 3000)
      }
    } catch (error) {
      // Error notification is now handled by Wails dialog in the store
      // Still show inline error for immediate feedback
      const message = error instanceof Error ? error.message : 'Connection test failed'
      setTestMessage({
        provider,
        message,
        type: 'error'
      })
      // Clear error message after 5 seconds (shorter since we also have dialog)
      setTimeout(() => setTestMessage(null), 5000)
    }
  }

  const handleOpenTerminal = async (commands: string[]) => {
    try {
      await fetch('/api/ai/ollama/open-terminal', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ commands }),
      })
    } catch (error) {
      alert(error instanceof Error ? error.message : 'Failed to open terminal')
    }
  }

  // Automatically attempt to initialise the embedded runtime once when selected.
  useEffect(() => {
    if (aiConfig.provider === 'ollama' && !hasAutoTestedLocalRef.current) {
      hasAutoTestedLocalRef.current = true
      testConnection('ollama').catch(() => {
        // Ignore errors here; the UI already shows status details and the user can retry manually.
      })
    }
    if (aiConfig.provider !== 'ollama' && hasAutoTestedLocalRef.current) {
      hasAutoTestedLocalRef.current = false
    }
  }, [aiConfig.provider, testConnection])

  // Detect changes in config
  useEffect(() => {
    const hasChanges = JSON.stringify(initialConfigRef.current) !== JSON.stringify(aiConfig)
    setHasUnsavedChanges(hasChanges)
  }, [aiConfig])

  // Handle save button click
  const handleSaveSettings = () => {
    try {
      // Settings are already auto-saved via updateConfig, so just update the initial ref
      initialConfigRef.current = aiConfig
      setHasUnsavedChanges(false)

      toast({
        title: "Settings saved",
        description: "Your settings have been saved successfully.",
      })
    } catch (error) {
      toast({
        title: "Error saving settings",
        description: error instanceof Error ? error.message : "Failed to save settings",
        variant: "destructive",
      })
    }
  }

  return (
    <PageErrorBoundary pageName="Settings">
      <div className="flex flex-1 h-full min-h-0 w-full flex-col overflow-y-auto">
      <div className="container mx-auto p-6 max-w-4xl">
        <div className="flex items-center gap-4 mb-6">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => navigate('/dashboard')}
        >
          <ArrowLeft className="h-4 w-4 mr-2" />
          Back
        </Button>
        <h1 className="text-3xl font-bold">Settings</h1>
      </div>

      <div className="space-y-6">
        {/* Appearance */}
        <Card>
          <CardHeader>
            <CardTitle>Appearance</CardTitle>
            <CardDescription>
              Customize the look and feel of HowlerOps
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label htmlFor="theme">Theme</Label>
                <p className="text-sm text-muted-foreground">
                  Select your preferred theme
                </p>
              </div>
              <Select value={theme} onValueChange={setTheme}>
                <SelectTrigger className="w-[180px]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="light">Light</SelectItem>
                  <SelectItem value="dark">Dark</SelectItem>
                  <SelectItem value="system">System</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label htmlFor="font-size">Editor Font Size</Label>
                <p className="text-sm text-muted-foreground">
                  Set the font size for the SQL editor
                </p>
              </div>
              <Input
                id="font-size"
                type="number"
                className="w-[100px]"
                defaultValue={14}
                min={10}
                max={24}
              />
            </div>
          </CardContent>
        </Card>

        {/* Editor Settings */}
        <Card>
          <CardHeader>
            <CardTitle>Editor</CardTitle>
            <CardDescription>
              Configure SQL editor behavior
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label htmlFor="auto-complete">Auto Complete</Label>
                <p className="text-sm text-muted-foreground">
                  Enable SQL auto-completion
                </p>
              </div>
              <Switch id="auto-complete" defaultChecked />
            </div>

            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label htmlFor="word-wrap">Word Wrap</Label>
                <p className="text-sm text-muted-foreground">
                  Wrap long lines in the editor
                </p>
              </div>
              <Switch id="word-wrap" />
            </div>

            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label htmlFor="line-numbers">Line Numbers</Label>
                <p className="text-sm text-muted-foreground">
                  Show line numbers in the editor
                </p>
              </div>
              <Switch id="line-numbers" defaultChecked />
            </div>
          </CardContent>
        </Card>

        {/* Query Settings */}
        <Card>
          <CardHeader>
            <CardTitle>Query Execution</CardTitle>
            <CardDescription>
              Configure query execution behavior
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label htmlFor="query-timeout">Query Timeout (seconds)</Label>
                <p className="text-sm text-muted-foreground">
                  Maximum time for query execution
                </p>
              </div>
              <Input
                id="query-timeout"
                type="number"
                className="w-[100px]"
                defaultValue={30}
                min={5}
                max={300}
              />
            </div>

            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label htmlFor="result-limit">Default Result Limit</Label>
                <p className="text-sm text-muted-foreground">
                  Default number of rows to fetch
                </p>
              </div>
              <Input
                id="result-limit"
                type="number"
                className="w-[100px]"
                defaultValue={1000}
                min={100}
                max={10000}
                step={100}
              />
            </div>

            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label htmlFor="auto-commit">Auto Commit</Label>
                <p className="text-sm text-muted-foreground">
                  Automatically commit transactions
                </p>
              </div>
              <Switch id="auto-commit" defaultChecked />
            </div>
          </CardContent>
        </Card>

        {/* AI Assistant Settings */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Brain className="h-5 w-5" />
              AI Assistant
            </CardTitle>
            <CardDescription>
              Configure AI-powered text-to-SQL generation and query assistance
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            {/* Enable AI Features */}
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label htmlFor="ai-enabled">Enable AI Features</Label>
                <p className="text-sm text-muted-foreground">
                  Enable natural language to SQL conversion and query assistance
                </p>
              </div>
              <Switch
                id="ai-enabled"
                checked={aiConfig.enabled}
                onCheckedChange={(checked) => handleAiConfigChange('enabled', checked)}
              />
            </div>

            {aiConfig.enabled && (
              <>
                {/* AI Provider Selection */}
                <div className="space-y-2">
                  <Label htmlFor="ai-provider">AI Provider</Label>
                  <Select
                    value={aiConfig.provider}
                    onValueChange={(value) => handleAiConfigChange('provider', value)}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="openai">OpenAI (GPT-4)</SelectItem>
                      <SelectItem value="anthropic">Anthropic (Claude)</SelectItem>
                      <SelectItem value="claudecode">Claude Code (CLI)</SelectItem>
                      <SelectItem value="codex">OpenAI Codex</SelectItem>
                      <SelectItem value="ollama">Ollama (Local)</SelectItem>
                      <SelectItem value="huggingface">Hugging Face (Local)</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                {/* Memory Sync Toggle */}
                <div className="flex items-start justify-between">
                  <div className="space-y-0.5 pr-4">
                    <Label htmlFor="ai-sync-memories">Sync AI Memories</Label>
                    <p className="text-sm text-muted-foreground">
                      Keep conversation history in sync with your workspace storage. Memories always stay local-first.
                    </p>
                  </div>
                  <Switch
                    id="ai-sync-memories"
                    checked={aiConfig.syncMemories}
                    onCheckedChange={(checked) => handleAiConfigChange('syncMemories', checked)}
                  />
                </div>

                {/* OpenAI Configuration */}
                {aiConfig.provider === 'openai' && (
                  <div className="space-y-4 p-4 border rounded-lg bg-muted/20">
                    <div className="flex items-center gap-2">
                      <Key className="h-4 w-4" />
                      <Label className="text-sm font-medium">OpenAI Configuration</Label>
                    </div>

                    <div className="space-y-2">
                      <Label htmlFor="openai-api-key">API Key</Label>
                      <div className="flex gap-2">
                        <Input
                          id="openai-api-key"
                          type={showApiKeys.openai ? "text" : "password"}
                          placeholder="sk-..."
                          value={aiConfig.openaiApiKey}
                          onChange={(e) => handleAiConfigChange('openaiApiKey', e.target.value)}
                        />
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => setShowApiKeys({...showApiKeys, openai: !showApiKeys.openai})}
                        >
                          {showApiKeys.openai ? "Hide" : "Show"}
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleTestConnection('openai')}
                          disabled={connectionStatus.openai === 'testing'}
                        >
                          {connectionStatus.openai === 'testing' ? 'Testing...' : 'Test'}
                        </Button>
                      </div>
                    </div>

                    <div className="space-y-2">
                      <Label htmlFor="openai-model">Model</Label>
                      <Select
                        value={aiConfig.selectedModel}
                        onValueChange={(value) => handleAiConfigChange('selectedModel', value)}
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="gpt-4o-mini">GPT-4o Mini (Recommended)</SelectItem>
                          <SelectItem value="gpt-4o">GPT-4o</SelectItem>
                          <SelectItem value="gpt-4-turbo">GPT-4 Turbo</SelectItem>
                          <SelectItem value="gpt-3.5-turbo">GPT-3.5 Turbo</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  </div>
                )}

                {/* Anthropic Configuration */}
                {aiConfig.provider === 'anthropic' && (
                  <div className="space-y-4 p-4 border rounded-lg bg-muted/20">
                    <div className="flex items-center gap-2">
                      <Key className="h-4 w-4" />
                      <Label className="text-sm font-medium">Anthropic Configuration</Label>
                    </div>

                    <div className="space-y-2">
                      <Label htmlFor="anthropic-api-key">API Key</Label>
                      <div className="flex gap-2">
                        <Input
                          id="anthropic-api-key"
                          type={showApiKeys.anthropic ? "text" : "password"}
                          placeholder="sk-ant-..."
                          value={aiConfig.anthropicApiKey}
                          onChange={(e) => handleAiConfigChange('anthropicApiKey', e.target.value)}
                        />
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => setShowApiKeys({...showApiKeys, anthropic: !showApiKeys.anthropic})}
                        >
                          {showApiKeys.anthropic ? "Hide" : "Show"}
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleTestConnection('anthropic')}
                          disabled={connectionStatus.anthropic === 'testing'}
                        >
                          {connectionStatus.anthropic === 'testing' ? 'Testing...' : 'Test'}
                        </Button>
                      </div>
                    </div>

                    <div className="space-y-2">
                      <Label htmlFor="anthropic-model">Model</Label>
                      <Select
                        value={aiConfig.selectedModel}
                        onValueChange={(value) => handleAiConfigChange('selectedModel', value)}
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="claude-3-5-sonnet-20241022">Claude 3.5 Sonnet (Recommended)</SelectItem>
                          <SelectItem value="claude-3-5-haiku-20241022">Claude 3.5 Haiku</SelectItem>
                          <SelectItem value="claude-3-opus-20240229">Claude 3 Opus</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  </div>
                )}

                {/* Local (Ollama) Configuration */}
                {aiConfig.provider === 'ollama' && (
                  <div className="space-y-4 p-4 border rounded-lg bg-muted/20">
                    <div className="flex items-center gap-2">
                      <Server className="h-4 w-4" />
                      <Label className="text-sm font-medium">Ollama Configuration</Label>
                    </div>

                    <div className="space-y-2">
                      <Label>Ollama Status</Label>
                      <div className="flex flex-wrap items-center gap-2 p-2 rounded-md bg-muted/50">
                        {ollamaDetection.installed ? (
                          <CheckCircle className="h-4 w-4 text-green-600" />
                        ) : (
                          <AlertTriangle className="h-4 w-4 text-red-600" />
                        )}
                        <span className="text-sm">
                          {ollamaDetection.installed ? 'Installed' : 'Not Installed'}
                        </span>
                        {ollamaDetection.running && (
                          <>
                            <CheckCircle className="h-4 w-4 text-green-600" />
                            <span className="text-sm text-green-600">Running</span>
                          </>
                        )}
                        {ollamaDetection.version && (
                          <span className="text-xs text-muted-foreground">
                            v{ollamaDetection.version}
                          </span>
                        )}
                        {ollamaDetection.available_models?.length ? (
                          <span className="text-xs text-muted-foreground">
                            Models: {ollamaDetection.available_models.slice(0, 3).join(', ')}
                            {ollamaDetection.available_models.length > 3 ? '…' : ''}
                          </span>
                        ) : null}
                      </div>
                      {(ollamaDetection.error || ollamaDetection.backend_available === false) && (
                        <p className="text-xs text-red-600 dark:text-red-400">
                          {ollamaDetection.error ?? 'Unable to reach the HowlerOps backend. Start the desktop app (`make dev`) to enable automatic model management.'}
                        </p>
                      )}
                    <div className="flex flex-wrap gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => ollamaDetection.detectOllama()}
                        disabled={ollamaDetection.isDetecting}
                        >
                          {ollamaDetection.isDetecting ? 'Checking…' : 'Refresh Status'}
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                        onClick={() => handleTestConnection('ollama')}
                        disabled={connectionStatus.ollama === 'testing'}
                      >
                        {connectionStatus.ollama === 'testing' ? 'Testing…' : 'Test Connection'}
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleOpenTerminal(['ollama serve'])}
                      >
                        Open Terminal (Serve)
                      </Button>
                    </div>
                    </div>

                    {!ollamaDetection.installed && (
                      <div className="space-y-2">
                        <Label>Install Ollama</Label>
                        <p className="text-sm text-muted-foreground">
                          Install Ollama once and we’ll reuse it every time you open HowlerOps.
                        </p>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={async () => {
                            try {
                              const instructions = await ollamaDetection.getInstallInstructions()
                              const newWindow = window.open('', '_blank', 'width=600,height=400')
                              if (newWindow) {
                                newWindow.document.write(`
                                  <html>
                                    <head><title>Ollama Installation Instructions</title></head>
                                    <body style="font-family: monospace; padding: 20px; white-space: pre-wrap;">${instructions}</body>
                                  </html>
                                `)
                              } else {
                                alert(instructions)
                              }
                            } catch (error) {
                              alert(`Failed to get installation instructions: ${error}`)
                            }
                          }}
                        >
                          <Download className="h-4 w-4 mr-2" />
                          Get Installation Instructions
                        </Button>
                      </div>
                    )}

                    {ollamaDetection.installed && !ollamaDetection.running && (
                      <div className="space-y-2">
                        <Label>Start Ollama Service</Label>
                        <div className="text-sm text-muted-foreground mb-2">
                          Ollama is installed but not running. Click below to start the service.
                        </div>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={async () => {
                            try {
                              await ollamaDetection.startOllamaService()
                            } catch (error) {
                              alert(`Failed to start Ollama: ${error}

You can also start it manually by running: ollama serve`)
                            }
                          }}
                          disabled={ollamaDetection.isStarting}
                        >
                          <Play className="h-4 w-4 mr-2" />
                          {ollamaDetection.isStarting ? 'Starting…' : 'Start Service'}
                        </Button>
                        <div className="text-xs text-muted-foreground">
                          Or run manually: <code className="bg-muted px-1 rounded">ollama serve</code>
                        </div>
                      </div>
                    )}

                    <div className="space-y-2">
                      <Label htmlFor="ollama-endpoint">Endpoint URL</Label>
                      <div className="flex gap-2">
                        <Input
                          id="ollama-endpoint"
                          placeholder="http://localhost:11434"
                          value={aiConfig.ollamaEndpoint}
                          onChange={(e) => handleAiConfigChange('ollamaEndpoint', e.target.value)}
                        />
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleTestConnection('ollama')}
                          disabled={connectionStatus.ollama === 'testing'}
                        >
                          {connectionStatus.ollama === 'testing' ? 'Testing…' : 'Test'}
                        </Button>
                      </div>
                    </div>

                    <div className="space-y-2">
                      <Label htmlFor="ollama-model">Model</Label>
                      <Select
                        value={aiConfig.selectedModel}
                        onValueChange={(value) => handleAiConfigChange('selectedModel', value)}
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="sqlcoder:7b">SQLCoder 7B (Recommended)</SelectItem>
                          <SelectItem value="codellama:7b">CodeLlama 7B</SelectItem>
                          <SelectItem value="llama3.1:8b">Llama 3.1 8B</SelectItem>
                          <SelectItem value="mistral:7b">Mistral 7B</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>

                    <div className="space-y-2 p-3 bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-800 rounded-md">
                      <div className="flex items-start gap-2">
                        <AlertTriangle className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5" />
                        <div className="text-sm text-blue-800 dark:text-blue-200 space-y-2">
                          <div>
                            <p className="font-medium">Hugging Face Models via Ollama</p>
                            <p>This provider uses Ollama to run Hugging Face models locally. Make sure Ollama is installed and running.</p>
                          </div>
                          <div className="flex flex-wrap gap-2 items-center">
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => handleTestConnection('huggingface')}
                              disabled={connectionStatus.huggingface === 'testing'}
                            >
                              {connectionStatus.huggingface === 'testing' ? 'Testing…' : 'Test Hugging Face'}
                            </Button>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => handleOpenTerminal([`ollama pull ${aiConfig.selectedModel}`])}
                            >
                              Open Terminal (Pull Model)
                            </Button>
                            <code className="bg-blue-100 dark:bg-blue-900/30 px-1 rounded text-xs">
                              ollama pull sqlcoder:7b
                            </code>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                )}

                {/* Claude Code Configuration */}
                {aiConfig.provider === 'claudecode' && (
                  <div className="space-y-4 p-4 border rounded-lg bg-muted/20">
                    <div className="flex items-center gap-2">
                      <Brain className="h-4 w-4" />
                      <Label className="text-sm font-medium">Claude Code Configuration</Label>
                    </div>

                    <div className="space-y-2">
                      <Label htmlFor="claudecode-path">Claude CLI Path (optional)</Label>
                      <div className="flex gap-2">
                        <Input
                          id="claudecode-path"
                          placeholder="Leave empty to use system PATH"
                          value={aiConfig.claudeCodePath}
                          onChange={(e) => handleAiConfigChange('claudeCodePath', e.target.value)}
                        />
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleTestConnection('claudecode')}
                          disabled={connectionStatus.claudecode === 'testing'}
                        >
                          {connectionStatus.claudecode === 'testing' ? 'Testing...' : 'Test'}
                        </Button>
                        {connectionStatus.claudecode === 'connected' && (
                          <CheckCircle className="h-5 w-5 text-green-500" />
                        )}
                      </div>
                      <p className="text-xs text-muted-foreground">
                        To authenticate, run <code className="bg-muted px-1 rounded">claude login</code> in your terminal. The app will automatically detect your credentials from <code className="bg-muted px-1 rounded">~/.claude/.credentials.json</code> or the <code className="bg-muted px-1 rounded">CLAUDE_CODE_OAUTH_TOKEN</code> environment variable.
                      </p>
                      {testMessage && testMessage.provider === 'claudecode' && (
                        <div className={`mt-2 p-2 rounded text-sm ${
                          testMessage.type === 'success'
                            ? 'bg-green-100 dark:bg-green-900/20 text-green-800 dark:text-green-200 border border-green-200 dark:border-green-800'
                            : 'bg-red-100 dark:bg-red-900/20 text-red-800 dark:text-red-200 border border-red-200 dark:border-red-800'
                        }`}>
                          {testMessage.type === 'success' ? '✅' : '❌'} {testMessage.message}
                        </div>
                      )}
                    </div>

                    <div className="space-y-2">
                      <Label htmlFor="claudecode-model">Model</Label>
                      <Select
                        value={aiConfig.selectedModel}
                        onValueChange={(value) => handleAiConfigChange('selectedModel', value)}
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="opus">Claude Opus (Most Capable)</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>

                    <div className="space-y-2 p-3 bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-800 rounded-md">
                      <div className="flex items-start gap-2">
                        <AlertTriangle className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5" />
                        <div className="text-sm text-blue-800 dark:text-blue-200">
                          <p className="font-medium">About Claude Code</p>
                          <p className="mt-1">Claude Code runs locally using the Claude CLI. To set up:</p>
                          <ol className="list-decimal list-inside mt-1 space-y-0.5">
                            <li>Install Claude Code from <a href="https://claude.ai/code" target="_blank" rel="noreferrer" className="underline">claude.ai/code</a></li>
                            <li>Run <code className="bg-blue-100 dark:bg-blue-900/30 px-1 rounded">claude login</code> in your terminal</li>
                            <li>Click "Test" above to verify the connection</li>
                          </ol>
                        </div>
                      </div>
                    </div>
                  </div>
                )}

                {/* Codex Configuration */}
                {aiConfig.provider === 'codex' && (
                  <div className="space-y-4 p-4 border rounded-lg bg-muted/20">
                    <div className="flex items-center gap-2">
                      <Key className="h-4 w-4" />
                      <Label className="text-sm font-medium">OpenAI Codex Configuration</Label>
                    </div>

                    <div className="space-y-2">
                      <Label htmlFor="codex-api-key">API Key (optional)</Label>
                      <div className="flex gap-2">
                        <Input
                          id="codex-api-key"
                          type={showApiKeys.codex ? "text" : "password"}
                          placeholder="Leave empty to use global credentials"
                          value={aiConfig.codexApiKey}
                          onChange={(e) => handleAiConfigChange('codexApiKey', e.target.value)}
                        />
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => setShowApiKeys({...showApiKeys, codex: !showApiKeys.codex})}
                        >
                          {showApiKeys.codex ? "Hide" : "Show"}
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleTestConnection('codex')}
                          disabled={connectionStatus.codex === 'testing'}
                        >
                          {connectionStatus.codex === 'testing' ? 'Testing...' : 'Test'}
                        </Button>
                        {connectionStatus.codex === 'connected' && (
                          <CheckCircle className="h-5 w-5 text-green-500" />
                        )}
                      </div>
                      <p className="text-xs text-muted-foreground">
                        Leave empty to automatically use credentials from <code className="bg-muted px-1 rounded">~/.codex/auth.json</code>, <code className="bg-muted px-1 rounded">OPENAI_API_KEY</code>, or <code className="bg-muted px-1 rounded">CODEX_API_KEY</code> environment variable. Authenticate via <code className="bg-muted px-1 rounded">openai login</code> CLI command.
                      </p>
                      {testMessage && testMessage.provider === 'codex' && (
                        <div className={`mt-2 p-2 rounded text-sm ${
                          testMessage.type === 'success'
                            ? 'bg-green-100 dark:bg-green-900/20 text-green-800 dark:text-green-200 border border-green-200 dark:border-green-800'
                            : 'bg-red-100 dark:bg-red-900/20 text-red-800 dark:text-red-200 border border-red-200 dark:border-red-800'
                        }`}>
                          {testMessage.type === 'success' ? '✅' : '❌'} {testMessage.message}
                        </div>
                      )}
                    </div>

                    <div className="space-y-2">
                      <Label htmlFor="codex-organization">Organization ID (optional)</Label>
                      <Input
                        id="codex-organization"
                        placeholder="org-..."
                        value={aiConfig.codexOrganization}
                        onChange={(e) => handleAiConfigChange('codexOrganization', e.target.value)}
                      />
                    </div>

                    <div className="space-y-2">
                      <Label htmlFor="codex-model">Model</Label>
                      <Select
                        value={aiConfig.selectedModel}
                        onValueChange={(value) => handleAiConfigChange('selectedModel', value)}
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="code-davinci-002">Codex Davinci (Recommended)</SelectItem>
                          <SelectItem value="code-cushman-001">Codex Cushman (Faster)</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>

                    <div className="space-y-2 p-3 bg-amber-50 dark:bg-amber-950/20 border border-amber-200 dark:border-amber-800 rounded-md">
                      <div className="flex items-start gap-2">
                        <AlertTriangle className="h-4 w-4 text-amber-600 dark:text-amber-400 mt-0.5" />
                        <div className="text-sm text-amber-800 dark:text-amber-200">
                          <p className="font-medium">Codex Access Required</p>
                          <p className="mt-1">OpenAI Codex requires special access. To set up:</p>
                          <ol className="list-decimal list-inside mt-1 space-y-0.5">
                            <li>Apply for Codex access through OpenAI</li>
                            <li>Run <code className="bg-amber-100 dark:bg-amber-900/30 px-1 rounded">openai login</code> in your terminal, or</li>
                            <li>Set your API key in the field above or via environment variables</li>
                            <li>Click "Test" to verify the connection</li>
                          </ol>
                        </div>
                      </div>
                    </div>
                  </div>
                )}

                {/* Hugging Face Configuration */}
                {aiConfig.provider === 'huggingface' && (
                  <div className="space-y-4 p-4 border rounded-lg bg-muted/20">
                    <div className="flex items-center gap-2">
                      <Server className="h-4 w-4" />
                      <Label className="text-sm font-medium">Hugging Face Configuration</Label>
                    </div>

                    {/* Ollama Status */}
                    <div className="space-y-2">
                      <Label>Ollama Status</Label>
                      <div className="flex items-center gap-2 p-2 rounded-md bg-muted/50">
                        {ollamaDetection.installed ? (
                          <CheckCircle className="h-4 w-4 text-green-600" />
                        ) : (
                          <AlertTriangle className="h-4 w-4 text-red-600" />
                        )}
                        <span className="text-sm">
                          {ollamaDetection.installed ? 'Installed' : 'Not Installed'}
                        </span>
                        {ollamaDetection.running && (
                          <>
                            <CheckCircle className="h-4 w-4 text-green-600" />
                            <span className="text-sm text-green-600">Running</span>
                          </>
                        )}
                        {ollamaDetection.version && (
                          <span className="text-xs text-muted-foreground">v{ollamaDetection.version}</span>
                        )}
                      </div>
                    </div>

                    {/* Ollama Actions */}
                    {!ollamaDetection.installed && (
                      <div className="space-y-2">
                        <Label>Install Ollama</Label>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={async () => {
                            try {
                              const instructions = await ollamaDetection.getInstallInstructions()
                              const newWindow = window.open('', '_blank', 'width=600,height=400')
                              if (newWindow) {
                                newWindow.document.write(`
                                  <html>
                                    <head><title>Ollama Installation Instructions</title></head>
                                    <body style="font-family: monospace; padding: 20px; white-space: pre-wrap;">${instructions}</body>
                                  </html>
                                `)
                              } else {
                                alert(instructions)
                              }
                            } catch (error) {
                              alert(`Failed to get installation instructions: ${error}`)
                            }
                          }}
                        >
                          <Download className="h-4 w-4 mr-2" />
                          Get Installation Instructions
                        </Button>
                      </div>
                    )}

                    {ollamaDetection.installed && !ollamaDetection.running && (
                      <div className="space-y-2">
                        <Label>Start Ollama Service</Label>
                        <div className="text-sm text-muted-foreground">
                          Ollama is installed but not running. Click below to start the service.
                        </div>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={async () => {
                            try {
                              await ollamaDetection.startOllamaService()
                            } catch (error) {
                              alert(`Failed to start Ollama: ${error}\n\nYou can also start it manually by running: ollama serve`)
                            }
                          }}
                          disabled={ollamaDetection.isStarting}
                        >
                          <Play className="h-4 w-4 mr-2" />
                          {ollamaDetection.isStarting ? 'Starting…' : 'Start Service'}
                        </Button>
                      </div>
                    )}

                    <div className="space-y-2">
                      <Label htmlFor="huggingface-endpoint">Ollama Endpoint URL</Label>
                      <div className="flex gap-2">
                        <Input
                          id="huggingface-endpoint"
                          placeholder="http://localhost:11434"
                          value={aiConfig.huggingfaceEndpoint}
                          onChange={(e) => handleAiConfigChange('huggingfaceEndpoint', e.target.value)}
                        />
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleTestConnection('huggingface')}
                          disabled={connectionStatus.huggingface === 'testing'}
                        >
                          {connectionStatus.huggingface === 'testing' ? 'Testing…' : 'Test'}
                        </Button>
                      </div>
                    </div>

                    <div className="space-y-2">
                      <Label htmlFor="huggingface-model">Model</Label>
                      <Select
                        value={aiConfig.selectedModel}
                        onValueChange={(value) => handleAiConfigChange('selectedModel', value)}
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="sqlcoder:7b">SQLCoder 7B (Recommended)</SelectItem>
                          <SelectItem value="codellama:7b">CodeLlama 7B</SelectItem>
                          <SelectItem value="llama3.1:8b">Llama 3.1 8B</SelectItem>
                          <SelectItem value="mistral:7b">Mistral 7B</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>

                    {ollamaDetection.running && (
                      <div className="space-y-2">
                        <Label>Install Recommended Model</Label>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={async () => {
                            try {
                              await ollamaDetection.pullModel('sqlcoder:7b')
                              alert('SQLCoder 7B model installed successfully!')
                            } catch (error) {
                              alert(`Failed to install model: ${error}`)
                            }
                          }}
                          disabled={ollamaDetection.isPulling}
                        >
                          <Download className="h-4 w-4 mr-2" />
                          {ollamaDetection.isPulling ? 'Installing…' : 'Install SQLCoder 7B'}
                        </Button>
                      </div>
                    )}
                  </div>
                )}

                {/* Advanced Settings */}
                <div className="space-y-4 p-4 border rounded-lg bg-muted/20">
                  <Label className="text-sm font-medium">Advanced Settings</Label>

                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label htmlFor="max-tokens">Max Tokens</Label>
                      <Input
                        id="max-tokens"
                        type="number"
                        min={256}
                        max={4096}
                        value={aiConfig.maxTokens}
                        onChange={(e) => handleAiConfigChange('maxTokens', parseInt(e.target.value))}
                      />
                    </div>

                    <div className="space-y-2">
                      <Label htmlFor="temperature">Temperature</Label>
                      <Input
                        id="temperature"
                        type="number"
                        min={0}
                        max={1}
                        step={0.1}
                        value={aiConfig.temperature}
                        onChange={(e) => handleAiConfigChange('temperature', parseFloat(e.target.value))}
                      />
                    </div>
                  </div>
                </div>
              </>
            )}
          </CardContent>
        </Card>

        {/* Save Button */}
        <div className="flex justify-end gap-4">
          <Button variant="outline" onClick={() => navigate('/dashboard')}>
            Cancel
          </Button>
          <Button onClick={handleSaveSettings} disabled={!hasUnsavedChanges}>
            Save Settings
          </Button>
        </div>
        </div>
      </div>
    </div>
    </PageErrorBoundary>
  )
}
