import { Tutorial } from "@/types/tutorial"

export const queryEditorBasicsTutorial: Tutorial = {
  id: "query-editor-basics",
  name: "Query Editor Basics",
  description: "Learn how to write, run, and manage SQL queries in the editor",
  category: "basics",
  difficulty: "beginner",
  estimatedMinutes: 5,
  steps: [
    {
      target: "[data-tutorial='query-editor']",
      title: "Welcome to the Query Editor",
      content: `
        <p>This is where you'll write and execute your SQL queries.</p>
        <p>The editor features:</p>
        <ul>
          <li>Syntax highlighting</li>
          <li>Auto-completion</li>
          <li>Error detection</li>
          <li>Multiple query tabs</li>
        </ul>
      `,
      placement: "center",
    },
    {
      target: "[data-tutorial='query-input']",
      title: "Writing Queries",
      content: `
        <p>Click in the editor and start typing your SQL query.</p>
        <p>Try typing <code>SELECT</code> to see auto-completion in action.</p>
      `,
      placement: "top",
      action: {
        type: "input",
        instruction: "Type a SQL query in the editor",
      },
    },
    {
      target: "[data-tutorial='autocomplete']",
      title: "Smart Auto-Completion",
      content: `
        <p>As you type, the editor suggests:</p>
        <ul>
          <li>SQL keywords</li>
          <li>Table names from your database</li>
          <li>Column names</li>
          <li>Functions and operators</li>
        </ul>
        <p>Press <kbd>Tab</kbd> or <kbd>Enter</kbd> to accept a suggestion.</p>
      `,
      placement: "bottom",
    },
    {
      target: "[data-tutorial='run-button']",
      title: "Running Queries",
      content: `
        <p>Click the <strong>Run</strong> button (or press <kbd>Cmd/Ctrl + Enter</kbd>) to execute your query.</p>
        <p>You can also run a selected portion of your query by highlighting it first.</p>
      `,
      placement: "bottom",
      action: {
        type: "click",
        instruction: "Click the Run button to execute the query",
      },
    },
    {
      target: "[data-tutorial='query-results']",
      title: "Viewing Results",
      content: `
        <p>Query results appear in the panel below the editor.</p>
        <p>You can:</p>
        <ul>
          <li>Sort columns by clicking headers</li>
          <li>Filter results</li>
          <li>Copy data</li>
          <li>Export to CSV, JSON, or Excel</li>
        </ul>
      `,
      placement: "top",
    },
    {
      target: "[data-tutorial='export-button']",
      title: "Exporting Data",
      content: `
        <p>Click the export button to download your results.</p>
        <p>Choose from multiple formats:</p>
        <ul>
          <li>CSV - Great for spreadsheets</li>
          <li>JSON - Perfect for APIs</li>
          <li>Excel - Professional reports</li>
        </ul>
      `,
      placement: "left",
    },
    {
      target: "[data-tutorial='query-history']",
      title: "Query History",
      content: `
        <p>Every query you run is automatically saved to your history.</p>
        <p>Quickly re-run previous queries or save them for later.</p>
      `,
      placement: "right",
    },
  ],
}
