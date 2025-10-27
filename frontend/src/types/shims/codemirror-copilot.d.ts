declare module 'codemirror-copilot' {
  import type { Extension } from '@codemirror/state'
  export function inlineCopilot(
    fetcher: (prefix: string, suffix: string) => Promise<string>,
    options?: { delay?: number }
  ): Extension
}


