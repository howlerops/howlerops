/**
 * Global Monaco Editor Configuration
 * Applies settings to prevent clipboard permission errors in WAILS WebView
 */

// Store original clipboard functions
let originalClipboard: {
  writeText?: typeof navigator.clipboard.writeText
  readText?: typeof navigator.clipboard.readText
  patched?: boolean
} = {}

/**
 * Patch clipboard API to fail silently instead of throwing permission errors
 * This is necessary for WAILS WebView environments where clipboard access may be restricted
 */
export function patchClipboardAPI() {
  if (originalClipboard.patched) {
    return // Already patched
  }

  if (typeof window === 'undefined' || !window.navigator?.clipboard) {
    return // Not in browser environment or no clipboard API
  }

  // Store originals
  originalClipboard.writeText = window.navigator.clipboard.writeText
  originalClipboard.readText = window.navigator.clipboard.readText
  originalClipboard.patched = true

  // Create safe wrappers that fail silently
  window.navigator.clipboard.writeText = async function(text: string) {
    try {
      if (originalClipboard.writeText) {
        return await originalClipboard.writeText.call(window.navigator.clipboard, text)
      }
    } catch (err) {
      // Silently fail - don't log to avoid console spam
      return Promise.resolve()
    }
  }

  window.navigator.clipboard.readText = async function() {
    try {
      if (originalClipboard.readText) {
        return await originalClipboard.readText.call(window.navigator.clipboard)
      }
    } catch (err) {
      // Return empty string on failure
      return Promise.resolve('')
    }
    return Promise.resolve('')
  }

  // Also patch the older document.execCommand approach if it exists
  if (document.execCommand) {
    const originalExecCommand = document.execCommand
    document.execCommand = function(command: string, showUI?: boolean, value?: string): boolean {
      try {
        // Block clipboard commands that might trigger permission errors
        if (command === 'copy' || command === 'cut' || command === 'paste') {
          return false
        }
        return originalExecCommand.call(document, command, showUI, value)
      } catch (err) {
        return false
      }
    }
  }
}

/**
 * Configure Monaco Editor loader to use safe defaults
 */
export function configureMonacoDefaults() {
  // Patch clipboard before any Monaco editor is created
  patchClipboardAPI()

  // Set global Monaco defaults if available
  if (typeof window !== 'undefined' && (window as any).monaco) {
    const monaco = (window as any).monaco

    // Configure defaults for all editors
    if (monaco.editor) {
      monaco.editor.EditorOptions = {
        ...monaco.editor.EditorOptions,
        copyWithSyntaxHighlighting: false,
        emptySelectionClipboard: false,
      }
    }
  }
}

/**
 * Get safe editor options that prevent clipboard errors
 */
export function getSafeEditorOptions(options: any = {}) {
  return {
    ...options,
    // Disable clipboard-related features
    copyWithSyntaxHighlighting: false,
    emptySelectionClipboard: false,
    // Disable features that might trigger clipboard access
    dragAndDrop: false,
    // Keep useful features
    fontSize: options.fontSize || 14,
    lineNumbers: options.lineNumbers || 'on',
    minimap: options.minimap || { enabled: false },
    scrollBeyondLastLine: false,
    automaticLayout: true,
    folding: true,
    wordWrap: 'on',
    renderLineHighlight: 'all',
    selectionHighlight: false,
  }
}