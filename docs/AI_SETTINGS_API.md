# AI Settings Page - API Reference

## Overview

The AI settings page should use these Wails-exported methods to manage AI provider configuration. All methods are available via the generated TypeScript bindings.

## Available Methods

### 1. Get Current Configuration

```typescript
import { GetAIConfiguration } from '../wailsjs/go/main/App';

// Get current AI provider configuration
const config = await GetAIConfiguration();

// Returns:
// {
//   provider: "openai" | "anthropic" | "ollama" | "codex" | "claudecode" | "huggingface",
//   apiKey: "sk-****1234",  // Masked for security
//   model: "gpt-4o-mini",
//   endpoint: "http://localhost:11434",  // For Ollama/HuggingFace
//   options: {}
// }
```

### 2. Configure AI Provider

```typescript
import { ConfigureAIProvider } from '../wailsjs/go/main/App';

// Configure a new provider
await ConfigureAIProvider({
  provider: "openai",
  apiKey: "sk-...",
  model: "gpt-4o-mini",
  endpoint: "",
  options: {}
});

// No restart required - takes effect immediately!
```

### 3. Test Provider Connection

```typescript
import { TestAIProvider } from '../wailsjs/go/main/App';

// Test a provider without saving configuration
const status = await TestAIProvider({
  provider: "openai",
  apiKey: "sk-...",
  model: "gpt-4o-mini",
  endpoint: "",
  options: {}
});

// Returns:
// {
//   name: "openai",
//   available: true,
//   error: ""  // Empty if successful
// }
```

### 4. Get All Provider Status

```typescript
import { GetAIProviderStatus } from '../wailsjs/go/main/App';

// Get status of all providers
const statuses = await GetAIProviderStatus();

// Returns:
// {
//   openai: { name: "OpenAI", available: true, error: "" },
//   anthropic: { name: "Anthropic", available: false, error: "Not configured" },
//   ollama: { name: "Ollama", available: true, error: "" },
//   // ...
// }
```

## UI Flow Example

### Settings Page Component

```typescript
import { useState, useEffect } from 'react';
import {
  GetAIConfiguration,
  ConfigureAIProvider,
  TestAIProvider,
  GetAIProviderStatus
} from '../wailsjs/go/main/App';

export function AISettingsPage() {
  const [provider, setProvider] = useState('openai');
  const [apiKey, setAPIKey] = useState('');
  const [model, setModel] = useState('gpt-4o-mini');
  const [endpoint, setEndpoint] = useState('http://localhost:11434');
  const [testing, setTesting] = useState(false);
  const [saving, setSaving] = useState(false);
  const [testResult, setTestResult] = useState(null);

  // Load current configuration on mount
  useEffect(() => {
    loadConfig();
  }, []);

  const loadConfig = async () => {
    try {
      const config = await GetAIConfiguration();
      setProvider(config.provider);
      setAPIKey(config.apiKey || '');
      setModel(config.model || 'gpt-4o-mini');
      setEndpoint(config.endpoint || 'http://localhost:11434');
    } catch (error) {
      console.error('Failed to load AI config:', error);
    }
  };

  const handleTestConnection = async () => {
    setTesting(true);
    setTestResult(null);

    try {
      const result = await TestAIProvider({
        provider,
        apiKey,
        model,
        endpoint,
        options: {}
      });

      setTestResult(result);
    } catch (error) {
      setTestResult({
        name: provider,
        available: false,
        error: error.message
      });
    } finally {
      setTesting(false);
    }
  };

  const handleSave = async () => {
    setSaving(true);

    try {
      await ConfigureAIProvider({
        provider,
        apiKey,
        model,
        endpoint,
        options: {}
      });

      // Show success message
      alert('AI provider configured successfully!');
    } catch (error) {
      alert(`Failed to configure: ${error.message}`);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="ai-settings">
      <h2>AI Provider Settings</h2>

      {/* Provider Selector */}
      <div>
        <label>Provider:</label>
        <select value={provider} onChange={(e) => setProvider(e.target.value)}>
          <option value="openai">OpenAI</option>
          <option value="anthropic">Anthropic Claude</option>
          <option value="ollama">Ollama (Local)</option>
          <option value="codex">Codex</option>
          <option value="claudecode">Claude Code</option>
          <option value="huggingface">HuggingFace</option>
        </select>
      </div>

      {/* API Key (for cloud providers) */}
      {['openai', 'anthropic', 'codex'].includes(provider) && (
        <div>
          <label>API Key:</label>
          <input
            type="password"
            value={apiKey}
            onChange={(e) => setAPIKey(e.target.value)}
            placeholder="sk-..."
          />
        </div>
      )}

      {/* Endpoint (for local providers) */}
      {['ollama', 'huggingface'].includes(provider) && (
        <div>
          <label>Endpoint:</label>
          <input
            type="text"
            value={endpoint}
            onChange={(e) => setEndpoint(e.target.value)}
            placeholder="http://localhost:11434"
          />
        </div>
      )}

      {/* Model */}
      <div>
        <label>Model:</label>
        <select value={model} onChange={(e) => setModel(e.target.value)}>
          {provider === 'openai' && (
            <>
              <option value="gpt-4o-mini">GPT-4o Mini (Recommended)</option>
              <option value="gpt-4o">GPT-4o</option>
              <option value="gpt-4-turbo">GPT-4 Turbo</option>
              <option value="gpt-3.5-turbo">GPT-3.5 Turbo</option>
            </>
          )}
          {provider === 'anthropic' && (
            <>
              <option value="claude-3-5-sonnet-20241022">Claude 3.5 Sonnet (Recommended)</option>
              <option value="claude-3-5-haiku-20241022">Claude 3.5 Haiku</option>
              <option value="claude-3-opus-20240229">Claude 3 Opus</option>
            </>
          )}
          {provider === 'ollama' && (
            <>
              <option value="sqlcoder:7b">SQLCoder 7B (Recommended)</option>
              <option value="codellama:7b">CodeLlama 7B</option>
              <option value="llama3.1:8b">Llama 3.1 8B</option>
            </>
          )}
        </select>
      </div>

      {/* Test Result */}
      {testResult && (
        <div className={testResult.available ? 'success' : 'error'}>
          {testResult.available ? (
            <span>✅ Connection successful!</span>
          ) : (
            <span>❌ {testResult.error}</span>
          )}
        </div>
      )}

      {/* Actions */}
      <div className="actions">
        <button
          onClick={handleTestConnection}
          disabled={testing || !apiKey}
        >
          {testing ? 'Testing...' : 'Test Connection'}
        </button>

        <button
          onClick={handleSave}
          disabled={saving || !apiKey}
        >
          {saving ? 'Saving...' : 'Save & Apply'}
        </button>
      </div>

      <p className="help-text">
        Changes take effect immediately - no restart required!
      </p>
    </div>
  );
}
```

## Configuration Persistence

Configuration is stored in environment variables in memory. For persistent storage across app restarts, you should:

1. **Option A:** Save to app preferences/local storage in the frontend
2. **Option B:** Add a persistent config file in the backend

Example for frontend persistence:

```typescript
// Save configuration
const saveConfig = async (config) => {
  await ConfigureAIProvider(config);
  localStorage.setItem('ai-config', JSON.stringify({
    provider: config.provider,
    model: config.model,
    endpoint: config.endpoint,
    // Don't save apiKey in localStorage for security
  }));
};

// Load on app start
const loadSavedConfig = async () => {
  const saved = localStorage.getItem('ai-config');
  if (saved) {
    const config = JSON.parse(saved);
    // User will need to re-enter API key on first use
    setProvider(config.provider);
    setModel(config.model);
    setEndpoint(config.endpoint);
  }
};
```

## Security Considerations

1. **API Keys:**
   - Never store API keys in localStorage (use secure storage or prompt user each time)
   - `GetAIConfiguration()` returns masked keys (`sk-****1234`) for display
   - Keys are kept in memory only during the app session

2. **Test Before Save:**
   - Always use `TestAIProvider()` before `ConfigureAIProvider()`
   - Prevents saving invalid credentials

3. **Error Handling:**
   - Always wrap API calls in try-catch
   - Display user-friendly error messages
   - Log errors for debugging

## Provider-Specific Notes

### OpenAI
- Requires valid API key from platform.openai.com
- Most reliable and highest quality
- Pay-per-use pricing

### Anthropic
- Requires API key from console.anthropic.com
- Excellent for SQL generation
- Pay-per-use pricing

### Ollama
- No API key needed
- Runs locally (privacy-friendly)
- Requires Ollama installed and running
- Free to use

### Claude Code
- Requires Claude CLI binary installed
- Set `binary_path` in options: `{ binary_path: "/usr/local/bin/claude" }`

### Codex
- Uses OpenAI API (same key as OpenAI)
- Optimized for code generation
- May have lower availability

### HuggingFace
- Currently runs through Ollama endpoint
- No API key needed if using Ollama
- Various open-source models available

## Testing Checklist

When building the AI settings page, test:

- ✅ Load existing configuration on mount
- ✅ Change provider updates UI fields appropriately
- ✅ Test connection button validates configuration
- ✅ Save button applies configuration immediately
- ✅ Error messages display clearly
- ✅ Success states show confirmation
- ✅ API keys are masked in display
- ✅ Configuration persists across page navigation (but not app restart unless you implement persistence)
- ✅ Multiple providers can be tested without affecting current config
- ✅ Status indicators update after configuration change

## TypeScript Bindings Location

After running `wails build`, the TypeScript bindings are generated at:

```
frontend/wailsjs/go/main/App.js
frontend/wailsjs/go/main/App.d.ts
```

Import types from:

```typescript
import { main } from '../wailsjs/go/models';

type ProviderConfig = main.ProviderConfig;
type ProviderStatus = main.ProviderStatus;
```

