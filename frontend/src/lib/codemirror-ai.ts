// Inline AI suggestions for CodeMirror 6
/* eslint-disable @typescript-eslint/no-require-imports */
// Provides a small factory to enable ghost-text suggestions using codemirror-copilot

import type { Extension } from '@codemirror/state'

let inlineCopilotImpl: ((
  fetcher: (prefix: string, suffix: string) => Promise<string>,
  options?: { delay?: number }
) => Extension) | null = null

try {
  // eslint-disable-next-line @typescript-eslint/no-var-requires
  const mod = require('codemirror-copilot')
  // Support multiple export shapes across versions
  if (typeof mod === 'function') {
    inlineCopilotImpl = mod
  } else if (typeof mod?.inlineCopilot === 'function') {
    inlineCopilotImpl = mod.inlineCopilot
  } else if (typeof mod?.default === 'function') {
    inlineCopilotImpl = mod.default
  } else if (typeof mod?.default?.inlineCopilot === 'function') {
    inlineCopilotImpl = mod.default.inlineCopilot
  } else {
    inlineCopilotImpl = null
  }
} catch {
  inlineCopilotImpl = null
}

export interface InlineAIOptions {
  enabled: boolean
  language?: string
  delay?: number
  maxChars?: number
  getSuggestion: (prefix: string, suffix: string, language: string) => Promise<string>
}

function clampContext(text: string, maxChars: number, takeEnd: boolean): string {
  if (text.length <= maxChars) return text
  return takeEnd ? text.slice(text.length - maxChars) : text.slice(0, maxChars)
}

export function createInlineAISuggestionsExtension(options: InlineAIOptions): Extension[] {
  const { enabled, language = 'sql', delay = 800, maxChars = 4000, getSuggestion } = options
  if (!enabled) return []

  const provider = async (prefix: string, suffix: string): Promise<string> => {
    const safePrefix = clampContext(prefix, maxChars, true)
    const safeSuffix = clampContext(suffix, maxChars, false)
    try {
      const result = await getSuggestion(safePrefix, safeSuffix, language)
      return typeof result === 'string' ? result : ''
    } catch {
      return ''
    }
  }

  if (!inlineCopilotImpl) {
    // Dependency not present: fail closed with no-op
    return []
  }
  return [inlineCopilotImpl(provider, { delay })]
}


