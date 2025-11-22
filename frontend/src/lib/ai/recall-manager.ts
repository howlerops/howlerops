/**
 * AI Recall Manager
 *
 * Manages AI recall operations - fetching related sessions
 * from backend storage based on semantic similarity.
 */

import type { AIRecallItem } from '@/types/ai'
import { RecallAIMemorySessions } from '../../../wailsjs/go/main/App'

/**
 * Fetches related sessions from backend based on prompt similarity
 *
 * @param prompt - The user prompt to find related sessions for
 * @param limit - Maximum number of sessions to recall (default: 5)
 * @returns Array of recalled session items, or empty array on error
 */
export async function recallRelatedSessions(
  prompt: string,
  limit: number = 5
): Promise<AIRecallItem[]> {
  try {
    const recalled = await RecallAIMemorySessions(prompt, limit)

    if (!Array.isArray(recalled) || recalled.length === 0) {
      return []
    }

    return recalled.map((item): AIRecallItem => ({
      title: String(item.title ?? ''),
      summary: item.summary ? String(item.summary) : undefined,
      content: String(item.content ?? ''),
    }))
  } catch (error) {
    console.error('Failed to recall AI memories:', error)
    return []
  }
}

/**
 * Builds recall context string from recalled items
 *
 * @param items - Recalled session items
 * @returns Formatted recall context string
 */
export function buildRecallContext(items: AIRecallItem[]): string {
  if (items.length === 0) {
    return ''
  }

  return items
    .map((item) => {
      const summary = item.summary ? `Summary: ${item.summary}\n` : ''
      return `Session: ${item.title}\n${summary}Memory:\n${item.content}`
    })
    .join('\n---\n')
}

/**
 * Fetches and builds recall context in one operation
 *
 * @param prompt - The user prompt to find related sessions for
 * @param limit - Maximum number of sessions to recall (default: 5)
 * @returns Formatted recall context string, or undefined if no recalls
 */
export async function getRecallContext(
  prompt: string,
  limit: number = 5
): Promise<string | undefined> {
  const items = await recallRelatedSessions(prompt, limit)

  if (items.length === 0) {
    return undefined
  }

  return buildRecallContext(items)
}
