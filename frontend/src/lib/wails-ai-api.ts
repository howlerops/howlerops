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
