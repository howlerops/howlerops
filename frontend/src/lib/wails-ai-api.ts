// Wails AI API integration
// This module provides AI testing functionality through Wails runtime bindings

import { TestOpenAIConnection, TestAnthropicConnection, TestOllamaConnection, TestClaudeCodeConnection, TestCodexConnection, TestHuggingFaceConnection, ShowNotification, StartClaudeCodeLogin, StartCodexLogin } from '../../wailsjs/go/main/App'
import { main } from '../../wailsjs/go/models'
import { toast } from '@/hooks/use-toast'

export type AITestResponse = main.AITestResponse

export interface AITestParams {
  provider: string
  apiKey?: string
  model?: string
  endpoint?: string
  organization?: string
  binaryPath?: string
}

/**
 * Test AI provider connection using Wails runtime bindings
 */
export async function testAIProviderConnection(params: AITestParams): Promise<AITestResponse> {
  try {
    let response: AITestResponse

    switch (params.provider) {
      case 'openai':
        if (!params.apiKey) {
          throw new Error('OpenAI API key is required')
        }
        response = await TestOpenAIConnection(params.apiKey, params.model || '')
        break

      case 'anthropic':
        if (!params.apiKey) {
          throw new Error('Anthropic API key is required')
        }
        response = await TestAnthropicConnection(params.apiKey, params.model || '')
        break

      case 'ollama':
        response = await TestOllamaConnection(params.endpoint || '', params.model || '')
        break

      case 'claudecode':
        response = await TestClaudeCodeConnection(params.binaryPath || '', params.model || '')
        break

      case 'codex':
        if (!params.apiKey) {
          throw new Error('Codex API key is required')
        }
        response = await TestCodexConnection(params.apiKey, params.model || '', params.organization || '')
        break

      case 'huggingface':
        response = await TestHuggingFaceConnection(params.endpoint || '', params.model || '')
        break

      default:
        throw new Error(`Unknown provider: ${params.provider}`)
    }

    return response
  } catch (error) {
    console.error(`AI provider test failed for ${params.provider}:`, error)
    return {
      success: false,
      message: '',
      error: error instanceof Error ? error.message : 'Unknown error occurred'
    }
  }
}

export async function launchClaudeCodeLogin(binaryPath: string): Promise<AITestResponse> {
  try {
    return await StartClaudeCodeLogin(binaryPath)
  } catch (error) {
    console.error('Claude login failed:', error)
    return {
      success: false,
      message: '',
      error: error instanceof Error ? error.message : 'Unable to launch Claude login',
    }
  }
}

export async function launchCodexLogin(binaryPath: string): Promise<AITestResponse> {
  try {
    return await StartCodexLogin(binaryPath)
  } catch (error) {
    console.error('Codex login failed:', error)
    return {
      success: false,
      message: '',
      error: error instanceof Error ? error.message : 'Unable to launch Codex login',
    }
  }
}

/**
 * Show a notification using Wails MessageDialog
 */
export async function showWailsNotification(title: string, message: string, isError: boolean = false): Promise<void> {
  try {
    await ShowNotification(title, message, isError)
  } catch (error) {
    console.error('Failed to show Wails notification:', error)
  }
}

/**
 * Show a toast notification (non-blocking UI notification)
 */
export function showToastNotification(title: string, message: string, variant: 'default' | 'success' | 'destructive' = 'default'): void {
  toast({
    title,
    description: message,
    variant,
    duration: variant === 'destructive' ? 8000 : 5000,
  })
}

/**
 * Show notification with both toast and optional Wails dialog
 */
export async function showHybridNotification(
  title: string,
  message: string,
  isError: boolean = false,
  useDialog: boolean = false
): Promise<void> {
  // Always show toast for immediate feedback
  showToastNotification(title, message, isError ? 'destructive' : 'success')

  // Optionally show dialog for critical messages
  if (useDialog) {
    await showWailsNotification(title, message, isError)
  }
}

// Inline editor suggestion endpoint wrapper.
// Returns code-only suggestion text or empty string on failure.
export async function aiSuggest(prefix: string, suffix: string, language: string = 'sql'): Promise<string> {
  try {
    const { GenericChat } = await import('../../wailsjs/go/main/App')
    const system = `You are an inline code completion engine. Continue the user's code strictly.
Rules:
- Output ONLY code with no commentary.
- Respect the specified language: ${language}.
- Use context before and after the cursor to complete naturally.
- Keep suggestions short (<= 200 characters) unless essential.`
    const prompt = `Complete the code at the cursor.
Language: ${language}
---
PREFIX:
${prefix}
---
SUFFIX:
${suffix}
---
Return only the code to insert at the cursor.`

    const resp = await GenericChat({
      prompt,
      context: '',
      system,
      provider: '',
      model: '',
      maxTokens: 128,
      temperature: 0.1,
      metadata: { intent: 'inline-completion', language }
    })

    const suggestion = (resp?.content || '').trim()
    if (!suggestion) return ''
    // Strip accidental triple backticks
    const cleaned = suggestion.replace(/^```[\s\S]*?\n|```$/g, '').trim()
    return cleaned
  } catch (error) {
    console.error('Inline AI suggestion failed:', error)
    return ''
  }
}
