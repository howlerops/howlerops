/**
 * CodeMirror SQL Editor Component
 * 
 * A React wrapper for CodeMirror 6 with SQL support
 */

import { useEffect, useRef, forwardRef, useImperativeHandle } from 'react'
import { EditorView, lineNumbers, highlightActiveLineGutter, highlightSpecialChars, drawSelection, dropCursor, rectangularSelection, crosshairCursor, highlightActiveLine, keymap } from '@codemirror/view'
import { EditorState, Extension } from '@codemirror/state'
import { defaultHighlightStyle, syntaxHighlighting, indentOnInput, bracketMatching, foldGutter, foldKeymap } from '@codemirror/language'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { searchKeymap, highlightSelectionMatches } from '@codemirror/search'
import { autocompletion, completionKeymap, closeBrackets, closeBracketsKeymap } from '@codemirror/autocomplete'
import { lintKeymap } from '@codemirror/lint'
import { createSQLExtensions, updateEditorSchema, Connection, SchemaNode, ColumnLoader } from '@/lib/codemirror-sql'
import { cn } from '@/lib/utils'
import { createInlineAISuggestionsExtension } from '@/lib/codemirror-ai'
import { aiSuggest } from '@/lib/wails-ai-api'

export interface CodeMirrorEditorProps {
  value?: string
  onChange?: (value: string) => void
  onMount?: (editor: EditorView) => void
  onExecute?: () => void
  theme?: 'light' | 'dark'
  height?: string
  readOnly?: boolean
  placeholder?: string
  connections?: Connection[]
  schemas?: Map<string, SchemaNode[]>
  mode?: 'single' | 'multi'
  columnLoader?: ColumnLoader
  className?: string
  aiEnabled?: boolean
  aiLanguage?: string
}

export interface CodeMirrorEditorRef {
  getView: () => EditorView | null
  getValue: () => string
  getSelectedText: () => string
  getCursorOffset: () => number
  setValue: (value: string) => void
  focus: () => void
}

export const CodeMirrorEditor = forwardRef<CodeMirrorEditorRef, CodeMirrorEditorProps>(
  (
    {
      value = '',
      onChange,
      onMount,
      onExecute,
      theme = 'light',
      height = '400px',
      readOnly = false,
      placeholder,
      connections = [],
      schemas = new Map(),
      mode = 'single',
      columnLoader,
      className,
      aiEnabled,
      aiLanguage
    },
    ref
  ) => {
    const editorRef = useRef<HTMLDivElement>(null)
    const viewRef = useRef<EditorView | null>(null)
    const valueRef = useRef(value)
    const onChangeRef = useRef(onChange)
    const onExecuteRef = useRef(onExecute)
    const onMountRef = useRef(onMount)

    // Update value ref
    useEffect(() => {
      valueRef.current = value
    }, [value])

    // Update callback refs
    useEffect(() => {
      onChangeRef.current = onChange
      onExecuteRef.current = onExecute
      onMountRef.current = onMount
    }, [onChange, onExecute, onMount])

    // Initialize editor
    useEffect(() => {
      if (!editorRef.current) return

      const basicSetup: Extension[] = [
        lineNumbers(),
        highlightActiveLineGutter(),
        highlightSpecialChars(),
        history(),
        foldGutter(),
        drawSelection(),
        dropCursor(),
        EditorState.allowMultipleSelections.of(true),
        indentOnInput(),
        syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
        bracketMatching(),
        closeBrackets(),
        autocompletion(),
        rectangularSelection(),
        crosshairCursor(),
        highlightActiveLine(),
        highlightSelectionMatches(),
        keymap.of([
          // Cmd/Ctrl+Enter to execute query
          ...(onExecuteRef.current ? [{
            key: 'Mod-Enter',
            run: () => {
              onExecuteRef.current?.()
              return true
            }
          }] : []),
          ...closeBracketsKeymap,
          ...defaultKeymap,
          ...searchKeymap,
          ...historyKeymap,
          ...foldKeymap,
          ...completionKeymap,
          ...lintKeymap,
        ])
      ]

      const extensions: Extension[] = [
        EditorView.theme({
          '&': {
            height: '100%',
            maxHeight: '100%',
          },
          '.cm-scroller': {
            overflow: 'auto',
          },
          '.cm-content': {
            minHeight: '100%',
          },
        }),
        ...basicSetup,
        ...createSQLExtensions(theme, columnLoader, (value: string) => {
          onChangeRef.current?.(value)
        }),
        // Inject AI inline suggestions conditionally
        ...((aiEnabled && !readOnly)
          ? createInlineAISuggestionsExtension({
              enabled: true,
              language: aiLanguage ?? 'sql',
              delay: 800,
              maxChars: 4000,
              getSuggestion: (prefix, suffix, language) => aiSuggest(prefix, suffix, language)
            })
          : []),
        EditorView.editable.of(!readOnly),
        EditorState.readOnly.of(readOnly)
      ]

      if (placeholder) {
        extensions.push(
          EditorView.theme({
            '.cm-placeholder': {
              color: 'var(--color-muted-foreground, #999)'
            }
          })
        )
      }

      const state = EditorState.create({
        doc: valueRef.current,
        extensions
      })

      const view = new EditorView({
        state,
        parent: editorRef.current
      })

      viewRef.current = view

      // Call onMount callback
      if (onMountRef.current) {
        onMountRef.current(view)
      }

      return () => {
        view.destroy()
        viewRef.current = null
      }
    }, [aiEnabled, aiLanguage, columnLoader, theme, readOnly, placeholder])

    // Update value when prop changes
    useEffect(() => {
      const view = viewRef.current
      if (!view) return

      const currentValue = view.state.doc.toString()
      if (currentValue !== value) {
        view.dispatch({
          changes: {
            from: 0,
            to: currentValue.length,
            insert: value
          }
        })
      }
    }, [value])

    // Update schema, connections, and mode
    useEffect(() => {
      const view = viewRef.current
      if (!view) return

      updateEditorSchema(view, connections, schemas, mode)
    }, [connections, schemas, mode])

    // Expose methods via ref
    useImperativeHandle(ref, () => ({
      getView: () => viewRef.current,
      getValue: () => viewRef.current?.state.doc.toString() || '',
      getSelectedText: () => {
        const view = viewRef.current
        if (!view) {
          return ''
        }

        const { ranges } = view.state.selection

        const segments = ranges
          .filter(range => !range.empty)
          .map(range => view.state.sliceDoc(range.from, range.to))

        if (segments.length === 0) {
          return ''
        }

        return segments.join('\n')
      },
      getCursorOffset: () => viewRef.current?.state.selection.main.head ?? 0,
      setValue: (newValue: string) => {
        const view = viewRef.current
        if (!view) return

        view.dispatch({
          changes: {
            from: 0,
            to: view.state.doc.length,
            insert: newValue
          }
        })
      },
      focus: () => {
        viewRef.current?.focus()
      }
    }))

    return (
      <div 
        className={cn(
          'codemirror-wrapper border overflow-hidden',
          'bg-background text-foreground',
          className
        )}
        style={{ height }}
      >
        <div ref={editorRef} className="h-full" />
      </div>
    )
  }
)

CodeMirrorEditor.displayName = 'CodeMirrorEditor'
