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

export interface CodeMirrorEditorProps {
  value?: string
  onChange?: (value: string) => void
  onMount?: (editor: EditorView) => void
  theme?: 'light' | 'dark'
  height?: string
  readOnly?: boolean
  placeholder?: string
  connections?: Connection[]
  schemas?: Map<string, SchemaNode[]>
  mode?: 'single' | 'multi'
  columnLoader?: ColumnLoader
  className?: string
}

export interface CodeMirrorEditorRef {
  getView: () => EditorView | null
  getValue: () => string
  setValue: (value: string) => void
  focus: () => void
}

export const CodeMirrorEditor = forwardRef<CodeMirrorEditorRef, CodeMirrorEditorProps>(
  (
    {
      value = '',
      onChange,
      onMount,
      theme = 'light',
      height = '400px',
      readOnly = false,
      placeholder,
      connections = [],
      schemas = new Map(),
      mode = 'single',
      columnLoader,
      className
    },
    ref
  ) => {
    const editorRef = useRef<HTMLDivElement>(null)
    const viewRef = useRef<EditorView | null>(null)
    const valueRef = useRef(value)

    // Update value ref
    useEffect(() => {
      valueRef.current = value
    }, [value])

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
          ...closeBracketsKeymap,
          ...defaultKeymap,
          ...searchKeymap,
          ...historyKeymap,
          ...foldKeymap,
          ...completionKeymap,
          ...lintKeymap,
        ])
      ]

      const extensions = [
        ...basicSetup,
        ...createSQLExtensions(theme, columnLoader, onChange),
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
        doc: value,
        extensions
      })

      const view = new EditorView({
        state,
        parent: editorRef.current
      })

      viewRef.current = view

      // Update schema state
      if (connections.length > 0 || schemas.size > 0) {
        updateEditorSchema(view, connections, schemas, mode)
      }

      // Call onMount callback
      if (onMount) {
        onMount(view)
      }

      return () => {
        view.destroy()
        viewRef.current = null
      }
    }, []) // eslint-disable-line react-hooks/exhaustive-deps

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

    // Update theme
    useEffect(() => {
      const view = viewRef.current
      if (!view) return

      // Recreate editor with new theme
      // Note: This is a simplified approach. In production, you might want to
      // use compartments for dynamic theme switching
    }, [theme])

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
          'codemirror-wrapper border rounded-md overflow-hidden',
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

