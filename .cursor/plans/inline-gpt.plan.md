<!-- 55eae6b0-5305-40b5-a59b-abcda82245fd 67f5dad0-dc45-462e-8f72-c264c598c823 -->
# Add GPT inline suggestions to CodeMirror (only when AI is configured)

### Research conclusion

- We will use `codemirror-copilot`, a lightweight CM6 extension providing Copilot-like ghost text and Tab-accept. It exposes a single `inlineCopilot` extension with caching and debounce, works with any React wrapper or vanilla CM6, and is MIT-licensed. See repo and usage docs: [codemirror-copilot](https://github.com/asadm/codemirror-copilot).
- Alternatives either require writing our own ghost-text decorations with CM6 facets or are unmaintained forks of `codemirror-extension-inline-suggestion` that this library builds upon. No official CM6 copilot plugin exists today. Given simplicity, TS types, and active releases, this is a good fit.

### Implementation outline

1) Frontend dependency

- Add `codemirror-copilot` to `frontend/package.json`.

2) AI-gated extension factory

- Create `frontend/src/lib/codemirror-ai.ts` exporting `createInlineAISuggestionsExtension(opts)`.
- It should:
  - Return `[]` if `opts.enabled` is false.
  - Otherwise return `[inlineCopilot(async (prefix, suffix) => await getSuggestion(prefix, suffix, language), opts.delay ?? 800)]`.
  - Clamp `prefix/suffix` to a configurable `maxChars` (e.g., 4000) to control token costs.
  - Default `language` to `"sql"`.
  - Call our existing AI API (via Wails or generated `ai_service.ts`) so keys never live in the browser.

3) Wire into editor only when AI configured

- In `frontend/src/components/codemirror-editor.tsx`, accept new optional props: `aiEnabled?: boolean`, `aiLanguage?: string`.
- When building `extensions`, append the AI extension after `createSQLExtensions(...)` only if `aiEnabled` is true.
```150:158:frontend/src/components/codemirror-editor.tsx
const extensions = [
  EditorView.theme({ /* ... */ }),
  ...basicSetup,
  ...createSQLExtensions(theme, columnLoader, (value: string) => {
    onChangeRef.current?.(value)
  }),
  // Inject AI suggestions conditionally
  ...(aiEnabled ? createInlineAISuggestionsExtension({
    enabled: true,
    language: aiLanguage ?? 'sql',
    delay: 800,
    maxChars: 4000,
    getSuggestion: (prefix, suffix, language) => aiSuggest(prefix, suffix, language)
  }) : []),
  EditorView.editable.of(!readOnly),
  EditorState.readOnly.of(readOnly)
]
```


4) Gate with existing AI store

- Use `frontend/src/store/ai-store.ts` (existing) to compute `isConfigured` (provider set and token/connection validated). Pass `aiEnabled={isConfigured}` to `QueryEditor` → `CodeMirrorEditor`.
- Default language for the SQL editor is `sql`.

5) Suggestion API

- Implement `aiSuggest(prefix, suffix, language)` in `frontend/src/lib/wails-ai-api.ts` or a small wrapper. Use the backend AI service to avoid exposing secrets. The prompt mirrors the library README: replace `<FILL_ME>` with just the code.
- If an explicit “inline code” endpoint doesn’t exist, reuse the existing chat/completion endpoint with a strict system prompt to return code-only.

6) UX details (minimal)

- Respect existing theme; ghost text uses CM styles. No new UI needed.
- Debounce to 800ms; allow Tab to accept.
- Fail closed: if backend returns empty or error, show nothing.

7) Validation

- Run frontend typecheck/lint/tests; backend tidy/fmt/tests; and `make validate` per repo guidelines. Build the app to ensure Wails binds are intact.

### Notes

- No provider-specific logic in the editor. The backend chooses the model using current user settings and can support OpenAI/Ollama/Anthropic without frontend change.
- Privacy: Only prefix/suffix are sent, truncated to `maxChars`. This keeps token costs bounded.

### To-dos

- [ ] Add codemirror-copilot dependency to frontend/package.json
- [ ] Create createInlineAISuggestionsExtension in frontend/src/lib/codemirror-ai.ts
- [ ] Add optional AI props and inject extension in codemirror-editor.tsx
- [ ] Derive aiEnabled from ai-store and pass to CodeMirrorEditor
- [ ] Implement aiSuggest() wrapper calling backend AI service
- [ ] Run full repo validation and build per guidelines