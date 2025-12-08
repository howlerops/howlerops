# HowlerOps AI Prompt Engineering Analysis & Improvements

**Date:** 2025-01-22
**Scope:** Comprehensive review of all AI prompts, patterns, and context injection strategies

---

## Executive Summary

This analysis reviews HowlerOps' AI prompt engineering across:
- SQL generation prompts (OpenAI, Anthropic, Ollama)
- SQL error fixing prompts
- RAG-enhanced SQL generation
- Inline code completion
- Chat/assistance prompts
- Schema context formatting

**Key Findings:**
- ✅ Strong foundation with JSON-structured outputs
- ⚠️ Inconsistent prompt quality across providers
- ⚠️ Schema context can be more token-efficient
- ⚠️ Missing few-shot examples for complex queries
- ⚠️ Limited model-specific optimizations
- ⚠️ Inline completion prompts need refinement

---

## 1. System Prompts Analysis

### Current State

#### OpenAI System Prompt
```go
// Location: backend-go/internal/ai/openai.go:288
"You are an expert SQL developer. Generate clean, efficient SQL queries and
provide clear explanations."
```

**Issues:**
- Generic and lacks specificity
- No constraint enforcement
- Missing output format guidance
- Doesn't mention database expertise areas

#### Anthropic System Prompt
```go
// Location: backend-go/internal/ai/anthropic.go:292
"You are an expert SQL developer. Generate clean, efficient SQL queries and
provide clear explanations. Always format your responses as valid JSON when
requested."
```

**Issues:**
- Better than OpenAI but still generic
- "When requested" is ambiguous - should be explicit
- No guidance on edge cases

#### Ollama System Prompt
```go
// Location: backend-go/internal/ai/ollama.go:23
"You are an expert SQL developer. Generate clean, efficient SQL queries and
provide clear explanations. Always format your responses as valid JSON when
requested."
```

**Issues:**
- Identical to Anthropic despite different model capabilities
- Ollama models may need more explicit instructions

### Improved System Prompts

#### Universal SQL Generation System Prompt (All Providers)

```
You are an expert SQL architect specializing in multi-database systems (PostgreSQL,
MySQL, SQL Server, SQLite, Oracle). Your expertise includes:

- Query optimization for performance
- Database-specific syntax and features
- Schema-aware query generation
- Security best practices (preventing SQL injection)
- Complex JOIN logic and subquery optimization

CRITICAL RULES:
1. ALWAYS respond with valid JSON in this exact structure:
   {
     "query": "<SQL code only>",
     "explanation": "<clear explanation>",
     "confidence": <0.0-1.0>,
     "suggestions": ["<optimization tip>", ...],
     "warnings": ["<potential issue>", ...]
   }

2. Generate syntactically correct SQL for the specified database type
3. Use proper indentation and formatting
4. Include helpful comments for complex logic
5. Prioritize readability and maintainability
6. Consider performance implications
7. Flag security concerns in warnings array
8. Set confidence based on schema completeness and query complexity

NEVER include markdown formatting, explanatory text outside JSON, or apologies.
```

**Improvements:**
- ✅ Explicit multi-database expertise
- ✅ Clear output format with mandatory structure
- ✅ Security awareness built-in
- ✅ Confidence scoring guidance
- ✅ Strict constraint enforcement

#### Provider-Specific Optimizations

**OpenAI (GPT-4):**
```
<Base system prompt above>

ADDITIONAL CONTEXT:
- You have extensive training on modern SQL standards (SQL:2016+)
- Leverage your knowledge of database internals for optimization
- Use chain-of-thought reasoning for complex queries
```

**Anthropic (Claude):**
```
<Base system prompt above>

ADDITIONAL CONTEXT:
- Think step-by-step through query construction
- Explicitly state assumptions about schema relationships
- Provide detailed reasoning for JOIN conditions
- Flag ambiguous requirements that need clarification
```

**Ollama (Local models):**
```
<Base system prompt above>

ADDITIONAL INSTRUCTIONS:
- Keep explanations concise and focused
- Prioritize correctness over advanced optimizations
- Use common SQL patterns and avoid exotic features
- If uncertain about syntax, state limitations clearly
```

---

## 2. SQL Generation Prompts

### Current OpenAI Generate Prompt

**Location:** `backend-go/internal/ai/openai.go:408-434`

```go
"Generate a SQL query for the following request:

Request: %s

Database Schema:
%s

Please provide:
1. The SQL query
2. An explanation of what the query does
3. Any important notes or considerations

Format your response as JSON with the following structure:
{
  \"query\": \"SELECT ...\",
  \"explanation\": \"This query...\",
  \"confidence\": 0.95,
  \"suggestions\": [\"Consider adding an index on...\", \"...\"]
}"
```

**Issues:**
- ❌ No database type specified
- ❌ Missing few-shot examples
- ❌ No performance hints
- ❌ No security considerations
- ❌ Weak constraint enforcement

### Improved SQL Generation Prompt (Universal)

```go
func (p *provider) buildGeneratePrompt(req *SQLRequest) string {
    var prompt strings.Builder

    // === TASK SPECIFICATION ===
    prompt.WriteString("TASK: Generate a SQL query based on the user's natural language request.\n\n")

    // === DATABASE CONTEXT ===
    if req.DatabaseType != "" {
        prompt.WriteString(fmt.Sprintf("DATABASE TYPE: %s\n", req.DatabaseType))
        prompt.WriteString(fmt.Sprintf("Use %s-specific syntax and features.\n\n", req.DatabaseType))
    }

    // === SCHEMA INFORMATION ===
    if req.Schema != "" {
        prompt.WriteString("=== DATABASE SCHEMA ===\n")
        prompt.WriteString(req.Schema)
        prompt.WriteString("\n\n")
    }

    // === USER REQUEST ===
    prompt.WriteString("=== USER REQUEST ===\n")
    prompt.WriteString(req.Prompt)
    prompt.WriteString("\n\n")

    // === HISTORICAL CONTEXT (if available) ===
    if len(req.SimilarQueries) > 0 {
        prompt.WriteString("=== SIMILAR PAST QUERIES ===\n")
        prompt.WriteString("The user has run similar queries:\n")
        for i, sq := range req.SimilarQueries[:min(3, len(req.SimilarQueries))] {
            prompt.WriteString(fmt.Sprintf("%d. %s\n   Similarity: %.0f%%\n",
                i+1, sq.Query, sq.Similarity*100))
        }
        prompt.WriteString("\n")
    }

    // === BUSINESS RULES (if available) ===
    if len(req.BusinessRules) > 0 {
        prompt.WriteString("=== BUSINESS RULES ===\n")
        prompt.WriteString("Ensure the query adheres to:\n")
        for _, rule := range req.BusinessRules {
            prompt.WriteString(fmt.Sprintf("- %s\n", rule.Description))
        }
        prompt.WriteString("\n")
    }

    // === FEW-SHOT EXAMPLES ===
    prompt.WriteString("=== EXAMPLE OUTPUT FORMAT ===\n")
    prompt.WriteString(generateFewShotExamples(req.DatabaseType))
    prompt.WriteString("\n\n")

    // === REQUIREMENTS ===
    prompt.WriteString("=== REQUIREMENTS ===\n")
    prompt.WriteString("✓ Generate syntactically correct SQL for " + req.DatabaseType + "\n")
    prompt.WriteString("✓ Optimize for query performance\n")
    prompt.WriteString("✓ Use explicit JOIN conditions (never rely on implicit joins)\n")
    prompt.WriteString("✓ Qualify column names with table names/aliases when joining\n")
    prompt.WriteString("✓ Use meaningful table aliases (e.g., 'u' for users, 'o' for orders)\n")
    prompt.WriteString("✓ Add appropriate WHERE clauses if context suggests filtering\n")
    prompt.WriteString("✓ Consider index usage in suggestions\n")
    prompt.WriteString("✓ Flag potential performance issues in warnings\n")
    prompt.WriteString("✓ Set confidence based on schema completeness\n")
    prompt.WriteString("✓ Include helpful comments for complex logic\n\n")

    // === OUTPUT FORMAT ===
    prompt.WriteString("=== OUTPUT FORMAT (STRICTLY REQUIRED) ===\n")
    prompt.WriteString("Respond with ONLY this JSON structure (no markdown, no extra text):\n")
    prompt.WriteString(`{
  "query": "<complete SQL query>",
  "explanation": "<clear explanation of what the query does and why>",
  "confidence": <0.0-1.0>,
  "suggestions": [
    "<optimization tip or best practice>",
    "<index recommendation if applicable>"
  ],
  "warnings": [
    "<potential performance issue>",
    "<security concern if any>"
  ]
}`)

    return prompt.String()
}

// Generate database-specific few-shot examples
func generateFewShotExamples(dbType string) string {
    var examples strings.Builder

    examples.WriteString("Example 1 (Simple query):\n")
    examples.WriteString("Request: \"Show all active users\"\n")
    examples.WriteString("Response:\n")
    examples.WriteString(`{
  "query": "SELECT id, username, email, created_at\nFROM users\nWHERE status = 'active'\nORDER BY created_at DESC;",
  "explanation": "Retrieves all columns for active users, ordered by creation date (newest first).",
  "confidence": 0.95,
  "suggestions": [
    "Consider adding an index on (status, created_at) for better performance",
    "Add LIMIT clause if pagination is needed"
  ],
  "warnings": []
}`)

    examples.WriteString("\n\nExample 2 (JOIN query):\n")
    examples.WriteString("Request: \"Show users with their order counts\"\n")
    examples.WriteString("Response:\n")
    examples.WriteString(`{
  "query": "SELECT \n    u.id,\n    u.username,\n    u.email,\n    COUNT(o.id) AS order_count\nFROM users u\nLEFT JOIN orders o ON o.user_id = u.id\nGROUP BY u.id, u.username, u.email\nORDER BY order_count DESC;",
  "explanation": "Joins users with orders using LEFT JOIN to include users with zero orders. Groups by user to count orders per user.",
  "confidence": 0.90,
  "suggestions": [
    "Ensure indexes exist on users.id and orders.user_id",
    "Consider materialized view if this query runs frequently"
  ],
  "warnings": [
    "LEFT JOIN ensures users with no orders are included (order_count = 0)"
  ]
}`)

    // Add database-specific example
    if dbType == "PostgreSQL" {
        examples.WriteString("\n\nExample 3 (PostgreSQL-specific):\n")
        examples.WriteString("Request: \"Get JSON array of user emails\"\n")
        examples.WriteString("Response:\n")
        examples.WriteString(`{
  "query": "SELECT json_agg(email) AS emails\nFROM users\nWHERE status = 'active';",
  "explanation": "Uses PostgreSQL's json_agg() to aggregate emails into a JSON array.",
  "confidence": 0.95,
  "suggestions": [
    "Use jsonb_agg() instead if you need to process the JSON further"
  ],
  "warnings": []
}`)
    }

    return examples.String()
}
```

**Improvements:**
- ✅ Clear section markers for readability
- ✅ Database type specification
- ✅ Historical context integration
- ✅ Business rule enforcement
- ✅ Few-shot examples with explanations
- ✅ Explicit requirements checklist
- ✅ Performance and security guidance
- ✅ Strict output format enforcement

---

## 3. SQL Error Fixing Prompts

### Current Anthropic Fix Prompt

**Location:** `backend-go/internal/ai/anthropic.go:427-456`

```go
"Fix the following SQL query that has an error:

Original Query:
```sql
%s
```

Error Message:
%s

Database Schema:
%s

Please provide your response as JSON..."
```

**Issues:**
- ❌ No error type categorization
- ❌ Missing common error patterns
- ❌ No root cause analysis guidance
- ❌ Doesn't leverage schema for context

### Improved SQL Fix Prompt

```go
func (p *provider) buildFixPrompt(req *SQLRequest) string {
    var prompt strings.Builder

    prompt.WriteString("TASK: Fix a SQL query that produced an error.\n\n")

    // === ERROR CONTEXT ===
    prompt.WriteString("=== ERROR CONTEXT ===\n")
    prompt.WriteString(fmt.Sprintf("Database Type: %s\n", req.DatabaseType))
    prompt.WriteString(fmt.Sprintf("Error Message:\n%s\n\n", req.Error))

    // Categorize error type
    errorCategory := categorizeError(req.Error)
    prompt.WriteString(fmt.Sprintf("Error Category: %s\n\n", errorCategory))

    // === ORIGINAL QUERY ===
    prompt.WriteString("=== ORIGINAL QUERY (HAS ERROR) ===\n")
    prompt.WriteString("```sql\n")
    prompt.WriteString(req.Query)
    prompt.WriteString("\n```\n\n")

    // === SCHEMA CONTEXT ===
    if req.Schema != "" {
        prompt.WriteString("=== DATABASE SCHEMA ===\n")
        prompt.WriteString(req.Schema)
        prompt.WriteString("\n\n")
    }

    // === DEBUGGING GUIDANCE ===
    prompt.WriteString("=== DEBUGGING STEPS ===\n")
    prompt.WriteString("1. Identify the root cause of the error\n")
    prompt.WriteString("2. Check table/column names against the schema\n")
    prompt.WriteString("3. Verify SQL syntax for " + req.DatabaseType + "\n")
    prompt.WriteString("4. Ensure JOIN conditions are correct\n")
    prompt.WriteString("5. Validate data types in comparisons\n")
    prompt.WriteString("6. Check for missing GROUP BY columns\n")
    prompt.WriteString("7. Verify aggregate function usage\n\n")

    // === COMMON ERROR PATTERNS ===
    prompt.WriteString("=== COMMON ERROR PATTERNS ===\n")
    prompt.WriteString(getCommonErrorPatterns(errorCategory))
    prompt.WriteString("\n\n")

    // === FIX REQUIREMENTS ===
    prompt.WriteString("=== REQUIREMENTS ===\n")
    prompt.WriteString("✓ Fix the specific error mentioned\n")
    prompt.WriteString("✓ Explain what caused the error\n")
    prompt.WriteString("✓ Explain how the fix resolves it\n")
    prompt.WriteString("✓ Provide prevention tips for future\n")
    prompt.WriteString("✓ Ensure the fixed query is optimized\n")
    prompt.WriteString("✓ Maintain the original query intent\n")
    prompt.WriteString("✓ Set confidence based on fix certainty\n\n")

    // === OUTPUT FORMAT ===
    prompt.WriteString("=== OUTPUT FORMAT (STRICTLY REQUIRED) ===\n")
    prompt.WriteString("Respond with ONLY this JSON structure:\n")
    prompt.WriteString(`{
  "query": "<corrected SQL query>",
  "explanation": "<root cause analysis> | <how the fix resolves it>",
  "confidence": <0.0-1.0>,
  "suggestions": [
    "<best practice to prevent this error>",
    "<related optimization opportunity>"
  ],
  "warnings": [
    "<any remaining concerns>",
    "<potential side effects of the fix>"
  ]
}`)

    return prompt.String()
}

// Categorize error for targeted guidance
func categorizeError(errorMsg string) string {
    errorMsgLower := strings.ToLower(errorMsg)

    if strings.Contains(errorMsgLower, "syntax error") {
        return "SYNTAX_ERROR"
    } else if strings.Contains(errorMsgLower, "column") &&
              (strings.Contains(errorMsgLower, "not found") ||
               strings.Contains(errorMsgLower, "unknown")) {
        return "UNKNOWN_COLUMN"
    } else if strings.Contains(errorMsgLower, "table") &&
              (strings.Contains(errorMsgLower, "not found") ||
               strings.Contains(errorMsgLower, "doesn't exist")) {
        return "UNKNOWN_TABLE"
    } else if strings.Contains(errorMsgLower, "group by") {
        return "GROUP_BY_ERROR"
    } else if strings.Contains(errorMsgLower, "ambiguous") {
        return "AMBIGUOUS_COLUMN"
    } else if strings.Contains(errorMsgLower, "type") ||
              strings.Contains(errorMsgLower, "cast") {
        return "TYPE_MISMATCH"
    } else if strings.Contains(errorMsgLower, "join") {
        return "JOIN_ERROR"
    }

    return "GENERAL_ERROR"
}

// Provide specific guidance based on error category
func getCommonErrorPatterns(category string) string {
    patterns := map[string]string{
        "SYNTAX_ERROR": `- Missing commas in SELECT list
- Unclosed parentheses or quotes
- Reserved keywords used without escaping
- Invalid operator usage`,

        "UNKNOWN_COLUMN": `- Column name typo or case sensitivity
- Column from wrong table in JOIN
- Missing table qualifier in multi-table query
- Column not in GROUP BY when using aggregates`,

        "UNKNOWN_TABLE": `- Table name typo or case sensitivity
- Missing schema prefix
- Table doesn't exist in current database
- CTE or subquery not properly defined`,

        "GROUP_BY_ERROR": `- SELECT column not in GROUP BY or aggregate
- Grouping by wrong column set
- Missing columns that determine uniqueness`,

        "AMBIGUOUS_COLUMN": `- Column exists in multiple joined tables
- Missing table qualifier (table.column)
- Duplicate aliases causing confusion`,

        "TYPE_MISMATCH": `- Comparing incompatible data types
- String used where number expected
- Date format issues
- NULL handling in comparisons`,

        "JOIN_ERROR": `- Missing ON clause
- Wrong join type (INNER vs LEFT)
- Incorrect join condition
- Cartesian product from missing condition`,
    }

    if pattern, exists := patterns[category]; exists {
        return pattern
    }
    return "Analyze error message for specific guidance"
}
```

**Improvements:**
- ✅ Error categorization for targeted fixing
- ✅ Common error pattern reference
- ✅ Step-by-step debugging guidance
- ✅ Root cause analysis emphasis
- ✅ Prevention tips requirement
- ✅ Schema-aware validation

---

## 4. Schema Context Optimization

### Current Schema Context

**Location:** `frontend/src/lib/ai-schema-context.ts:226-298`

**Issues:**
- ❌ Verbose - includes all column details
- ❌ Not token-optimized
- ❌ Missing relationship information
- ❌ Doesn't prioritize important tables
- ❌ No smart truncation

### Improved Schema Context Strategy

```typescript
export class AISchemaContextBuilder {
  /**
   * Generate token-optimized schema context with smart prioritization
   */
  static generateOptimizedSchemaContext(
    context: MultiDatabaseContext,
    userPrompt: string,
    maxTokens: number = 2000
  ): string {
    // Analyze prompt to identify relevant tables
    const relevantTables = this.identifyRelevantTables(userPrompt, context);

    let schemaContext = '';
    let estimatedTokens = 0;

    if (context.mode === 'single') {
      const db = context.databases[0];
      schemaContext += `DB: ${db.database} (${db.connectionName})\n\n`;

      // Priority 1: Tables mentioned in prompt
      const priorityTables = this.filterPriorityTables(
        db.schemas,
        relevantTables.high
      );
      schemaContext += this.formatTablesCompact(priorityTables, 'HIGH_PRIORITY');

      // Priority 2: Related tables (via foreign keys)
      const relatedTables = this.findRelatedTables(priorityTables, db.schemas);
      schemaContext += this.formatTablesCompact(relatedTables, 'RELATED');

      // Priority 3: Other tables (if space allows)
      if (estimatedTokens < maxTokens * 0.7) {
        const otherTables = this.filterRemainingTables(
          db.schemas,
          [...priorityTables, ...relatedTables]
        );
        schemaContext += this.formatTablesMinimal(otherTables);
      }

    } else {
      // Multi-DB mode - even more aggressive compression
      schemaContext += `Multi-DB Mode (@conn.table syntax)\n\n`;

      for (const db of context.databases) {
        const dbTables = this.filterPriorityTables(
          db.schemas,
          relevantTables.high
        );

        if (dbTables.length > 0) {
          schemaContext += `@${db.connectionName}:\n`;
          schemaContext += this.formatTablesUltraCompact(dbTables);
        }
      }
    }

    return schemaContext;
  }

  /**
   * Identify tables likely relevant to user's prompt
   */
  private static identifyRelevantTables(
    prompt: string,
    context: MultiDatabaseContext
  ): { high: Set<string>, medium: Set<string> } {
    const high = new Set<string>();
    const medium = new Set<string>();
    const promptLower = prompt.toLowerCase();

    // Extract potential table references from prompt
    const commonWords = new Set(['get', 'show', 'find', 'list', 'count',
                                  'select', 'from', 'with', 'their', 'all']);
    const words = promptLower.split(/\s+/).filter(w =>
      w.length > 2 && !commonWords.has(w)
    );

    // Check each database's tables
    for (const db of context.databases) {
      for (const schema of db.schemas) {
        for (const table of schema.tables) {
          const tableName = table.name.toLowerCase();

          // High priority: exact or plural match
          if (words.includes(tableName) ||
              words.includes(tableName + 's') ||
              words.includes(tableName.slice(0, -1))) {
            high.add(table.name);
          }

          // Medium priority: substring match
          else if (words.some(w => tableName.includes(w) || w.includes(tableName))) {
            medium.add(table.name);
          }
        }
      }
    }

    return { high, medium };
  }

  /**
   * Ultra-compact table format: table(col1,col2,col3)
   */
  private static formatTablesUltraCompact(tables: TableInfo[]): string {
    return tables.map(t => {
      const cols = t.columns
        .slice(0, 5)
        .map(c => c.name)
        .join(',');
      const more = t.columns.length > 5 ? ',+' + (t.columns.length - 5) : '';
      return `  ${t.name}(${cols}${more})`;
    }).join('\n') + '\n\n';
  }

  /**
   * Compact format with types: table: col1:type, col2:type
   */
  private static formatTablesCompact(tables: TableInfo[], priority: string): string {
    if (tables.length === 0) return '';

    let output = `${priority} TABLES:\n`;
    for (const table of tables) {
      output += `${table.name}:\n`;

      // Show PK columns first
      const pkColumns = table.columns.filter(c => c.primaryKey);
      const regularColumns = table.columns.filter(c => !c.primaryKey).slice(0, 8);

      for (const col of [...pkColumns, ...regularColumns]) {
        const pk = col.primaryKey ? ' [PK]' : '';
        output += `  ${col.name}: ${col.dataType}${pk}\n`;
      }

      if (table.columns.length > pkColumns.length + 8) {
        output += `  ... +${table.columns.length - pkColumns.length - 8} more\n`;
      }

      // Show foreign keys
      if (table.foreignKeys.length > 0) {
        output += `  FKs: `;
        output += table.foreignKeys
          .map(fk => `${fk.column}→${fk.referencedTable}.${fk.referencedColumn}`)
          .join(', ');
        output += '\n';
      }

      output += '\n';
    }

    return output;
  }

  /**
   * Find tables related via foreign keys
   */
  private static findRelatedTables(
    baseTables: TableInfo[],
    allSchemas: SchemaInfo[]
  ): TableInfo[] {
    const baseTableNames = new Set(baseTables.map(t => t.name));
    const relatedTables = new Set<string>();

    // Find tables referenced by base tables
    for (const table of baseTables) {
      for (const fk of table.foreignKeys) {
        relatedTables.add(fk.referencedTable);
      }
    }

    // Find tables that reference base tables
    for (const schema of allSchemas) {
      for (const table of schema.tables) {
        if (baseTableNames.has(table.name)) continue;

        for (const fk of table.foreignKeys) {
          if (baseTableNames.has(fk.referencedTable)) {
            relatedTables.add(table.name);
          }
        }
      }
    }

    // Return table objects
    const result: TableInfo[] = [];
    for (const schema of allSchemas) {
      for (const table of schema.tables) {
        if (relatedTables.has(table.name)) {
          result.push(table);
        }
      }
    }

    return result;
  }
}
```

**Improvements:**
- ✅ Token-aware context building
- ✅ Smart table prioritization based on prompt
- ✅ Relationship-aware (FK navigation)
- ✅ Multi-tier compression strategies
- ✅ Ultra-compact format for token limits

---

## 5. Inline Completion Prompts

### Current Inline Completion

**Location:** `frontend/src/lib/wails-ai-api.ts:145-164`

```typescript
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
```

**Issues:**
- ⚠️ Works but could be more specific
- ⚠️ No SQL-specific guidance
- ⚠️ Doesn't leverage schema context
- ⚠️ 200 char limit too restrictive for SQL

### Improved Inline Completion Prompt

```typescript
export async function aiSuggest(
  prefix: string,
  suffix: string,
  language: string = 'sql',
  schemaContext?: string
): Promise<string> {
  try {
    const { GenericChat } = await import('../../wailsjs/go/main/App')

    // Build context-aware system prompt
    const system = buildInlineSystemPrompt(language);
    const prompt = buildInlineUserPrompt(prefix, suffix, language, schemaContext);

    const resp = await GenericChat({
      prompt,
      context: schemaContext || '',
      system,
      provider: '',
      model: '',
      maxTokens: language === 'sql' ? 256 : 128,  // More tokens for SQL
      temperature: 0.1,  // Low temperature for deterministic completions
      metadata: { intent: 'inline-completion', language }
    })

    const suggestion = (resp?.content || '').trim()
    if (!suggestion) return ''

    // Clean response
    return cleanInlineCompletion(suggestion, language)
  } catch (error) {
    console.error('Inline AI suggestion failed:', error)
    return ''
  }
}

function buildInlineSystemPrompt(language: string): string {
  const basePrompt = `You are an expert inline code completion engine for ${language.toUpperCase()}.

CRITICAL RULES:
1. Output ONLY the code to insert at the cursor
2. NO explanations, comments, or markdown formatting
3. NO greetings, apologies, or conversational text
4. Continue naturally from the prefix context
5. Consider the suffix context (code after cursor)
6. Match the indentation and style of existing code
7. Generate syntactically correct code`;

  if (language === 'sql') {
    return basePrompt + `

SQL-SPECIFIC RULES:
8. Use proper SQL formatting (keywords uppercase, aliases lowercase)
9. Complete table names, column names, or conditions
10. For SELECT: suggest column names from schema
11. For JOIN: suggest ON conditions based on foreign keys
12. For WHERE: suggest common filter patterns
13. Keep completions under 3 lines unless completing a full clause
14. Use explicit column names (avoid SELECT *)
15. Always use table aliases in JOIN queries

EXAMPLES:
Prefix: "SELECT u.id, u.username, "
→ Complete with: "u.email, u.created_at"

Prefix: "FROM users u\nJOIN orders o ON "
→ Complete with: "o.user_id = u.id"

Prefix: "WHERE status = 'active' AND "
→ Complete with: "created_at > NOW() - INTERVAL '30 days'"`;
  }

  return basePrompt;
}

function buildInlineUserPrompt(
  prefix: string,
  suffix: string,
  language: string,
  schemaContext?: string
): string {
  let prompt = '';

  // Add schema context for SQL
  if (language === 'sql' && schemaContext) {
    prompt += '=== AVAILABLE SCHEMA ===\n';
    prompt += schemaContext;
    prompt += '\n\n';
  }

  prompt += '=== CODE CONTEXT ===\n';
  prompt += `Language: ${language}\n\n`;

  prompt += '--- CODE BEFORE CURSOR ---\n';
  prompt += prefix || '(empty)';
  prompt += '\n\n';

  prompt += '--- CODE AFTER CURSOR ---\n';
  prompt += suffix || '(empty)';
  prompt += '\n\n';

  prompt += '=== TASK ===\n';
  prompt += 'Complete the code at the cursor position.\n';
  prompt += 'Return ONLY the text to insert (no markdown, no explanation).\n';

  if (language === 'sql') {
    prompt += 'Use the schema context to suggest accurate table/column names.\n';
  }

  return prompt;
}

function cleanInlineCompletion(suggestion: string, language: string): string {
  let cleaned = suggestion.trim();

  // Remove markdown code blocks
  cleaned = cleaned.replace(/^```[\w]*\n?/gm, '');
  cleaned = cleaned.replace(/\n?```$/gm, '');

  // Remove common explanatory prefixes
  const prefixesToRemove = [
    /^(Here's|Here is|This is|This completes)/i,
    /^(The code|The completion|The suggestion)/i,
    /^(You can|You should|Consider)/i,
  ];

  for (const pattern of prefixesToRemove) {
    cleaned = cleaned.replace(pattern, '');
  }

  // For SQL, clean up extra whitespace but preserve formatting
  if (language === 'sql') {
    // Remove leading/trailing blank lines
    cleaned = cleaned.replace(/^\n+/, '').replace(/\n+$/, '');
  }

  return cleaned.trim();
}
```

**Improvements:**
- ✅ Language-specific rules (SQL gets special treatment)
- ✅ Schema-aware SQL completions
- ✅ Concrete examples in system prompt
- ✅ Better context formatting
- ✅ Smarter response cleaning
- ✅ Higher token limit for SQL

---

## 6. RAG-Enhanced SQL Generation

### Current RAG Prompt Pattern

**Location:** `backend-go/internal/rag/smart_sql_generator.go:93-137`

**Analysis:**
The RAG system builds context but doesn't have explicit prompts. The context includes:
- Similar historical queries
- Business rules
- Performance hints
- Schema relationships

**Issue:** Context is passed to LLM provider but prompts don't explicitly guide the model on how to use RAG context.

### Improved RAG Integration

```go
// Enhance SQL generation prompt with RAG context guidance
func buildRAGEnhancedPrompt(
    userPrompt string,
    ragContext *QueryContext,
    req *SQLRequest,
) string {
    var prompt strings.Builder

    prompt.WriteString("TASK: Generate SQL query using historical context and business rules.\n\n")

    // === USER REQUEST ===
    prompt.WriteString("=== USER REQUEST ===\n")
    prompt.WriteString(userPrompt)
    prompt.WriteString("\n\n")

    // === HISTORICAL CONTEXT ===
    if len(ragContext.SimilarQueries) > 0 {
        prompt.WriteString("=== SIMILAR PAST QUERIES (LEARN FROM THESE) ===\n")
        prompt.WriteString("The user has successfully run these similar queries:\n\n")

        for i, sq := range ragContext.SimilarQueries[:min(3, len(ragContext.SimilarQueries))] {
            prompt.WriteString(fmt.Sprintf("%d. Similarity: %.0f%%\n", i+1, sq.Similarity*100))
            prompt.WriteString(fmt.Sprintf("   Query: %s\n", sq.Query))
            if sq.Description != "" {
                prompt.WriteString(fmt.Sprintf("   Purpose: %s\n", sq.Description))
            }
            prompt.WriteString("\n")
        }

        prompt.WriteString("GUIDANCE: Use these queries as patterns but adapt to the current request.\n\n")
    }

    // === BUSINESS RULES ===
    if len(ragContext.BusinessRules) > 0 {
        prompt.WriteString("=== BUSINESS RULES (MUST FOLLOW) ===\n")
        prompt.WriteString("The generated query MUST adhere to these rules:\n\n")

        for i, rule := range ragContext.BusinessRules {
            prompt.WriteString(fmt.Sprintf("%d. %s\n", i+1, rule.Description))
            if len(rule.Conditions) > 0 {
                prompt.WriteString("   Required conditions:\n")
                for _, cond := range rule.Conditions {
                    prompt.WriteString(fmt.Sprintf("   - %s\n", cond))
                }
            }
            prompt.WriteString("\n")
        }

        prompt.WriteString("CRITICAL: Violating these rules will cause data errors.\n\n")
    }

    // === PERFORMANCE HINTS ===
    if len(ragContext.PerformanceHints) > 0 {
        prompt.WriteString("=== PERFORMANCE OPTIMIZATION HINTS ===\n")
        prompt.WriteString("Apply these optimizations where applicable:\n\n")

        for i, hint := range ragContext.PerformanceHints {
            prompt.WriteString(fmt.Sprintf("%d. %s\n", i+1, hint.Description))
            if hint.Example != "" {
                prompt.WriteString(fmt.Sprintf("   Example: %s\n", hint.Example))
            }
            prompt.WriteString("\n")
        }
    }

    // === SCHEMA CONTEXT ===
    if len(ragContext.RelevantSchemas) > 0 {
        prompt.WriteString("=== RELEVANT SCHEMA ===\n")
        prompt.WriteString(formatRelevantSchemas(ragContext.RelevantSchemas))
        prompt.WriteString("\n")
    }

    // === QUERY PLANNING GUIDANCE ===
    if ragContext.RequiresPlanning {
        prompt.WriteString("=== QUERY COMPLEXITY NOTICE ===\n")
        prompt.WriteString("This query requires multiple steps. Consider using:\n")
        prompt.WriteString("- CTEs (WITH clauses) for clarity\n")
        prompt.WriteString("- Subqueries for intermediate results\n")
        prompt.WriteString("- Window functions for analytics\n\n")
    }

    // === REQUIREMENTS ===
    prompt.WriteString("=== SYNTHESIS REQUIREMENTS ===\n")
    prompt.WriteString("✓ Incorporate patterns from similar historical queries\n")
    prompt.WriteString("✓ Enforce all business rules without exception\n")
    prompt.WriteString("✓ Apply performance hints where relevant\n")
    prompt.WriteString("✓ Use the provided schema for accurate references\n")
    prompt.WriteString("✓ Explain how you used the historical context\n")
    prompt.WriteString("✓ Note any business rules applied in the explanation\n\n")

    // === OUTPUT FORMAT ===
    prompt.WriteString("=== OUTPUT FORMAT ===\n")
    prompt.WriteString(`{
  "query": "<optimized SQL query>",
  "explanation": "<what the query does> | <historical patterns used> | <business rules applied>",
  "confidence": <0.0-1.0, higher if similar patterns exist>,
  "suggestions": ["<optimization tip>", ...],
  "warnings": ["<business rule consideration>", ...]
}`)

    return prompt.String()
}

func formatRelevantSchemas(schemas []SchemaTable) string {
    var output strings.Builder

    for _, schema := range schemas {
        output.WriteString(fmt.Sprintf("%s:\n", schema.TableName))

        // Show columns with relationships
        for _, col := range schema.Columns {
            colInfo := fmt.Sprintf("  %s (%s)", col.Name, col.Type)
            if col.IsPrimaryKey {
                colInfo += " [PK]"
            }
            if col.ForeignKey != nil {
                colInfo += fmt.Sprintf(" → %s.%s",
                    col.ForeignKey.ReferencedTable,
                    col.ForeignKey.ReferencedColumn)
            }
            output.WriteString(colInfo + "\n")
        }

        output.WriteString("\n")
    }

    return output.String()
}
```

**Improvements:**
- ✅ Explicit guidance on using historical queries
- ✅ Business rule enforcement with criticality
- ✅ Performance hint application
- ✅ Context synthesis requirements
- ✅ Explanation of how context was used

---

## 7. Chat Assistant Prompts

### Current Chat System Prompts

**OpenAI:** "You are a helpful assistant for Howlerops. Provide concise, accurate answers. Use Markdown formatting when it improves clarity."

**Anthropic:** "You are a helpful assistant for Howlerops. Provide thoughtful, concise answers and include actionable guidance when relevant."

**Issues:**
- ❌ Too generic
- ❌ Doesn't establish database expertise
- ❌ No guidance on code examples
- ❌ Missing constraint enforcement

### Improved Chat System Prompt

```
You are an expert database assistant for HowlerOps - a multi-database query and management platform.

YOUR EXPERTISE:
- SQL across PostgreSQL, MySQL, SQL Server, SQLite, Oracle
- Database design, optimization, and best practices
- Query troubleshooting and performance tuning
- Schema design and normalization
- Index strategies and query planning
- Database-specific features and syntax

YOUR COMMUNICATION STYLE:
- Provide clear, concise, and actionable guidance
- Use technical precision but remain accessible
- Include code examples with proper formatting
- Explain the "why" behind recommendations
- Highlight potential pitfalls and edge cases
- Offer alternatives when multiple approaches exist

RESPONSE STRUCTURE:
1. Direct answer to the question
2. Code example if applicable (with comments)
3. Explanation of key concepts
4. Best practices or optimization tips
5. Related considerations or warnings

FORMATTING RULES:
✓ Use Markdown formatting for clarity
✓ Use ```sql blocks for SQL code
✓ Use **bold** for important terms
✓ Use `inline code` for table/column names
✓ Use bullet points for lists
✓ Keep paragraphs short (2-3 sentences max)

CODE EXAMPLES:
✓ Include realistic, runnable examples
✓ Add helpful comments
✓ Show both simple and advanced approaches when relevant
✓ Highlight database-specific syntax differences

CONSTRAINTS:
✗ Never suggest SQL injection vulnerabilities
✗ Never recommend disabling security features
✗ Never propose dropping production data without warnings
✗ Never assume user's database state - ask if unclear

WHEN UNCERTAIN:
- Explicitly state assumptions
- Ask clarifying questions
- Provide multiple options with tradeoffs
- Acknowledge limitations in your knowledge

Your goal: Empower users to write better SQL, understand their databases,
and solve problems efficiently.
```

**Improvements:**
- ✅ Establishes clear expertise domain
- ✅ Defines communication style
- ✅ Structured response format
- ✅ Explicit formatting rules
- ✅ Safety constraints built-in
- ✅ Uncertainty handling guidance

---

## 8. Model-Specific Optimizations

### OpenAI (GPT-4o)

**Strengths:** Strong reasoning, good at complex queries, JSON adherence
**Optimizations:**
- Use structured outputs API for guaranteed JSON
- Leverage function calling for schema introspection
- Include chain-of-thought for complex queries
- Higher temperature acceptable (0.3-0.5)

```go
// OpenAI-specific enhancements
type openaiStructuredRequest struct {
    Model             string                 `json:"model"`
    Messages          []openaiChatMessage    `json:"messages"`
    ResponseFormat    *openaiResponseFormat  `json:"response_format,omitempty"`
    Temperature       float64                `json:"temperature"`
}

type openaiResponseFormat struct {
    Type       string          `json:"type"`
    JSONSchema *openaiSchema   `json:"json_schema,omitempty"`
}

// Use structured outputs for guaranteed JSON
func (p *openaiProvider) GenerateSQL(ctx context.Context, req *SQLRequest) (*SQLResponse, error) {
    // Define strict JSON schema
    schema := &openaiSchema{
        Name: "sql_response",
        Schema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "query":       {"type": "string"},
                "explanation": {"type": "string"},
                "confidence":  {"type": "number", "minimum": 0, "maximum": 1},
                "suggestions": {"type": "array", "items": {"type": "string"}},
                "warnings":    {"type": "array", "items": {"type": "string"}},
            },
            "required": []string{"query", "explanation", "confidence"},
            "additionalProperties": false,
        },
    }

    // Use response_format for guaranteed JSON structure
    requestBody := openaiStructuredRequest{
        Model:    req.Model,
        Messages: buildMessages(req),
        ResponseFormat: &openaiResponseFormat{
            Type:       "json_schema",
            JSONSchema: schema,
        },
        Temperature: 0.3,
    }

    // Make request...
}
```

### Anthropic (Claude)

**Strengths:** Excellent instruction following, nuanced understanding, longer context
**Optimizations:**
- Use extended system prompts (Claude handles longer prompts well)
- Explicit thinking sections for complex queries
- Provide detailed examples
- Lower temperature for consistency (0.1-0.2)

```go
// Claude-specific thinking prompt
func (p *anthropicProvider) buildGeneratePromptWithThinking(req *SQLRequest) string {
    var prompt strings.Builder

    prompt.WriteString("<task>Generate SQL query from natural language</task>\n\n")

    prompt.WriteString("<request>\n")
    prompt.WriteString(req.Prompt)
    prompt.WriteString("\n</request>\n\n")

    if req.Schema != "" {
        prompt.WriteString("<schema>\n")
        prompt.WriteString(req.Schema)
        prompt.WriteString("\n</schema>\n\n")
    }

    // Claude-specific: explicit thinking section
    prompt.WriteString("<instructions>\n")
    prompt.WriteString("Before generating the SQL, think through:\n")
    prompt.WriteString("1. What tables are needed?\n")
    prompt.WriteString("2. What columns should be selected?\n")
    prompt.WriteString("3. What JOINs are required?\n")
    prompt.WriteString("4. What filters or conditions apply?\n")
    prompt.WriteString("5. How can this be optimized?\n\n")

    prompt.WriteString("Then provide your response in this JSON format:\n")
    prompt.WriteString("</instructions>\n\n")

    prompt.WriteString("<output_format>\n")
    prompt.WriteString(`{
  "thinking": "<your step-by-step reasoning>",
  "query": "<final SQL query>",
  "explanation": "<clear explanation>",
  "confidence": <0.0-1.0>,
  "suggestions": [],
  "warnings": []
}`)
    prompt.WriteString("\n</output_format>\n")

    return prompt.String()
}
```

### Ollama (Local Models)

**Strengths:** Fast, private, customizable
**Limitations:** Smaller models, less reasoning capability, may need more guidance
**Optimizations:**
- Simpler, more direct prompts
- More explicit examples
- Shorter context windows
- Clear step-by-step instructions
- Lower expectations for complex reasoning

```go
// Ollama-specific simplified prompt
func (p *OllamaProvider) buildGeneratePromptSimplified(req *SQLRequest) string {
    var prompt strings.Builder

    prompt.WriteString("Task: Write SQL query\n\n")

    prompt.WriteString("Request: ")
    prompt.WriteString(req.Prompt)
    prompt.WriteString("\n\n")

    // Simplified schema (only essentials)
    if req.Schema != "" {
        prompt.WriteString("Tables:\n")
        prompt.WriteString(simplifySchema(req.Schema))
        prompt.WriteString("\n\n")
    }

    // Concrete example
    prompt.WriteString("Example:\n")
    prompt.WriteString("Request: Show all active users\n")
    prompt.WriteString("SQL: SELECT * FROM users WHERE status = 'active';\n\n")

    // Very explicit output format
    prompt.WriteString("Your turn. Write ONLY JSON:\n")
    prompt.WriteString(`{"query": "...", "explanation": "...", "confidence": 0.9}`)

    return prompt.String()
}

func simplifySchema(schema string) string {
    // Extract just table and column names, drop types and details
    // Format: table_name: col1, col2, col3
    // This reduces token usage significantly
}
```

---

## 9. Error Handling & Retry Strategies

### Current State
No explicit retry logic with prompt refinement visible in the code.

### Improved Retry with Prompt Feedback

```go
// Retry failed SQL generation with error feedback
func (s *serviceImpl) GenerateSQLWithRetry(
    ctx context.Context,
    req *SQLRequest,
    maxRetries int,
) (*SQLResponse, error) {
    var lastError error
    var lastResponse *SQLResponse

    for attempt := 0; attempt < maxRetries; attempt++ {
        // Add previous error to context for retry
        if attempt > 0 && lastError != nil {
            req.PreviousAttempt = &PreviousAttempt{
                Query: lastResponse.Query,
                Error: lastError.Error(),
            }
        }

        response, err := s.GenerateSQL(ctx, req)

        if err == nil {
            // Success - validate the SQL
            if validationErr := s.validateSQL(response.Query, req.DatabaseType); validationErr == nil {
                return response, nil
            } else {
                // SQL is syntactically invalid - retry with validation error
                lastError = validationErr
                lastResponse = response
                continue
            }
        }

        lastError = err
        lastResponse = response
    }

    return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastError)
}

// Enhanced prompt with previous attempt feedback
func buildGeneratePromptWithFeedback(req *SQLRequest) string {
    prompt := buildGeneratePrompt(req)

    if req.PreviousAttempt != nil {
        prompt += "\n\n=== PREVIOUS ATTEMPT (FAILED) ===\n"
        prompt += "Previous Query:\n"
        prompt += req.PreviousAttempt.Query
        prompt += "\n\nError:\n"
        prompt += req.PreviousAttempt.Error
        prompt += "\n\nIMPORTANT: Learn from this error and generate a corrected version.\n"
    }

    return prompt
}
```

---

## 10. Token Efficiency Strategies

### Prompt Compression Techniques

1. **Schema Compression:**
   - Priority-based table inclusion
   - Column truncation with "..." indicators
   - Type abbreviations (VARCHAR(255) → vc255)
   - Remove redundant information

2. **Example Compression:**
   - Use 1-2 examples max for few-shot
   - Compact formatting
   - Essential information only

3. **Instruction Compression:**
   - Use symbols (✓ ✗ instead of "must" and "must not")
   - Bulleted lists instead of paragraphs
   - Section markers (===) for clarity without verbosity

4. **Dynamic Token Allocation:**
```go
func allocateTokenBudget(
    maxTokens int,
    hasSchema bool,
    hasSimilarQueries bool,
    hasBusinessRules bool,
) TokenBudget {
    budget := TokenBudget{
        SystemPrompt:    int(float64(maxTokens) * 0.15),
        UserPrompt:      int(float64(maxTokens) * 0.10),
        Schema:          0,
        SimilarQueries:  0,
        BusinessRules:   0,
        Examples:        int(float64(maxTokens) * 0.10),
        Instructions:    int(float64(maxTokens) * 0.10),
        OutputSpace:     int(float64(maxTokens) * 0.35),
    }

    remaining := maxTokens - (budget.SystemPrompt + budget.UserPrompt +
                             budget.Examples + budget.Instructions +
                             budget.OutputSpace)

    // Allocate remaining tokens to optional components
    if hasSchema {
        budget.Schema = int(float64(remaining) * 0.50)
        remaining -= budget.Schema
    }

    if hasSimilarQueries {
        budget.SimilarQueries = int(float64(remaining) * 0.30)
        remaining -= budget.SimilarQueries
    }

    if hasBusinessRules {
        budget.BusinessRules = remaining
    }

    return budget
}
```

---

## 11. Few-Shot Example Library

### SQL Generation Examples

```go
var SQLGenerationExamples = map[string][]Example{
    "simple_select": {
        {
            Request: "Get all active users",
            SQL: "SELECT id, username, email, created_at\nFROM users\nWHERE status = 'active'\nORDER BY created_at DESC;",
            Explanation: "Retrieves active users ordered by creation date",
        },
    },

    "join_query": {
        {
            Request: "Show users with their order counts",
            SQL: `SELECT
    u.id,
    u.username,
    u.email,
    COUNT(o.id) AS order_count
FROM users u
LEFT JOIN orders o ON o.user_id = u.id
GROUP BY u.id, u.username, u.email
ORDER BY order_count DESC;`,
            Explanation: "LEFT JOIN ensures users with zero orders are included",
        },
    },

    "aggregate_query": {
        {
            Request: "Calculate total revenue by month",
            SQL: `SELECT
    DATE_TRUNC('month', order_date) AS month,
    SUM(total_amount) AS revenue,
    COUNT(*) AS order_count,
    AVG(total_amount) AS avg_order_value
FROM orders
WHERE order_date >= NOW() - INTERVAL '12 months'
GROUP BY DATE_TRUNC('month', order_date)
ORDER BY month DESC;`,
            Explanation: "Aggregates orders by month with revenue metrics",
        },
    },

    "subquery": {
        {
            Request: "Find users who have never placed an order",
            SQL: `SELECT id, username, email
FROM users
WHERE id NOT IN (
    SELECT DISTINCT user_id
    FROM orders
    WHERE user_id IS NOT NULL
)
ORDER BY created_at DESC;`,
            Explanation: "Uses NOT IN subquery to find users without orders",
        },
    },

    "cte": {
        {
            Request: "Show top 5 customers by revenue with their details",
            SQL: `WITH customer_revenue AS (
    SELECT
        user_id,
        SUM(total_amount) AS total_revenue,
        COUNT(*) AS order_count
    FROM orders
    GROUP BY user_id
)
SELECT
    u.id,
    u.username,
    u.email,
    cr.total_revenue,
    cr.order_count
FROM customer_revenue cr
JOIN users u ON u.id = cr.user_id
ORDER BY cr.total_revenue DESC
LIMIT 5;`,
            Explanation: "Uses CTE for readable multi-step query",
        },
    },
}

// Select relevant examples based on query complexity
func selectRelevantExamples(prompt string, maxExamples int) []Example {
    promptLower := strings.ToLower(prompt)

    var selected []Example

    // Detect query type
    if strings.Contains(promptLower, "join") ||
       strings.Contains(promptLower, "with their") {
        selected = append(selected, SQLGenerationExamples["join_query"]...)
    }

    if strings.Contains(promptLower, "count") ||
       strings.Contains(promptLower, "total") ||
       strings.Contains(promptLower, "sum") {
        selected = append(selected, SQLGenerationExamples["aggregate_query"]...)
    }

    if strings.Contains(promptLower, "never") ||
       strings.Contains(promptLower, "not") {
        selected = append(selected, SQLGenerationExamples["subquery"]...)
    }

    // Default to simple if no matches
    if len(selected) == 0 {
        selected = SQLGenerationExamples["simple_select"]
    }

    // Limit to maxExamples
    if len(selected) > maxExamples {
        selected = selected[:maxExamples]
    }

    return selected
}
```

---

## 12. Confidence Scoring Guidance

### Enhanced Confidence Calculation

```go
// Calculate confidence score based on multiple factors
func calculateConfidence(
    req *SQLRequest,
    ragContext *QueryContext,
    generated *GeneratedSQL,
) float64 {
    score := 1.0

    // Factor 1: Schema completeness (-0.3 if no schema)
    if req.Schema == "" {
        score -= 0.3
    }

    // Factor 2: Historical similarity (+ up to 0.2)
    if len(ragContext.SimilarQueries) > 0 {
        maxSimilarity := 0.0
        for _, sq := range ragContext.SimilarQueries {
            if sq.Similarity > maxSimilarity {
                maxSimilarity = sq.Similarity
            }
        }
        score += (maxSimilarity - 0.5) * 0.4  // Scale 0.5-1.0 to 0.0-0.2
    } else {
        score -= 0.1  // No historical context
    }

    // Factor 3: Query complexity (- up to 0.2 for complex)
    complexity := analyzeQueryComplexity(generated.Query)
    if complexity == "complex" {
        score -= 0.2
    } else if complexity == "moderate" {
        score -= 0.1
    }

    // Factor 4: Ambiguity detection (- up to 0.3)
    if detectAmbiguity(req.Prompt) {
        score -= 0.2
    }

    // Factor 5: Business rule coverage (+ 0.1 if all covered)
    if len(ragContext.BusinessRules) > 0 {
        if allBusinessRulesCovered(generated.Query, ragContext.BusinessRules) {
            score += 0.1
        } else {
            score -= 0.15
        }
    }

    // Clamp to [0.0, 1.0]
    if score < 0.0 {
        score = 0.0
    } else if score > 1.0 {
        score = 1.0
    }

    return score
}

func analyzeQueryComplexity(query string) string {
    queryLower := strings.ToLower(query)

    complexity := 0

    // Count complexity indicators
    complexity += strings.Count(queryLower, "join")
    complexity += strings.Count(queryLower, "select") - 1  // Subqueries
    complexity += strings.Count(queryLower, "with")         // CTEs
    complexity += strings.Count(queryLower, "union")
    complexity += strings.Count(queryLower, "having")
    complexity += strings.Count(queryLower, "case when")

    if complexity <= 1 {
        return "simple"
    } else if complexity <= 3 {
        return "moderate"
    }
    return "complex"
}

func detectAmbiguity(prompt string) bool {
    ambiguousTerms := []string{
        "some", "few", "many", "recent", "old",
        "cheap", "expensive", "fast", "slow",
    }

    promptLower := strings.ToLower(prompt)

    for _, term := range ambiguousTerms {
        if strings.Contains(promptLower, term) {
            return true
        }
    }

    return false
}
```

---

## Implementation Priority

### Phase 1: High Impact (Implement First)
1. ✅ Universal SQL generation system prompt
2. ✅ Improved SQL generation prompts with few-shot examples
3. ✅ Enhanced SQL fix prompts with error categorization
4. ✅ Schema context optimization and prioritization

### Phase 2: Medium Impact
5. ✅ Inline completion improvements
6. ✅ Chat assistant system prompt
7. ✅ Provider-specific optimizations
8. ✅ Confidence scoring enhancements

### Phase 3: Advanced Features
9. ✅ RAG-enhanced prompt patterns
10. ✅ Retry strategies with feedback
11. ✅ Token budget management
12. ✅ Few-shot example library

---

## Testing Strategy

### Prompt Testing Framework

```go
type PromptTest struct {
    Name           string
    UserPrompt     string
    Schema         string
    DatabaseType   string
    ExpectedSQL    string
    MinConfidence  float64
    ShouldContain  []string
    ShouldNotContain []string
}

var promptTests = []PromptTest{
    {
        Name:         "Simple SELECT with WHERE",
        UserPrompt:   "Get all active users",
        Schema:       "users(id, username, email, status)",
        DatabaseType: "PostgreSQL",
        ExpectedSQL:  "SELECT * FROM users WHERE status = 'active'",
        MinConfidence: 0.9,
        ShouldContain: []string{"SELECT", "FROM users", "WHERE status"},
    },
    {
        Name:         "JOIN query",
        UserPrompt:   "Show users with their order counts",
        Schema:       "users(id, username); orders(id, user_id)",
        DatabaseType: "PostgreSQL",
        MinConfidence: 0.85,
        ShouldContain: []string{"JOIN", "COUNT", "GROUP BY"},
        ShouldNotContain: []string{"CROSS JOIN"},
    },
    // Add 50+ test cases covering common patterns
}

func TestPromptQuality(t *testing.T) {
    for _, test := range promptTests {
        t.Run(test.Name, func(t *testing.T) {
            response, err := generateSQL(test)
            require.NoError(t, err)

            // Check confidence
            assert.GreaterOrEqual(t, response.Confidence, test.MinConfidence)

            // Check SQL contains expected elements
            for _, expected := range test.ShouldContain {
                assert.Contains(t, response.Query, expected)
            }

            // Check SQL doesn't contain problematic elements
            for _, unexpected := range test.ShouldNotContain {
                assert.NotContains(t, response.Query, unexpected)
            }
        })
    }
}
```

---

## Metrics & Monitoring

Track these metrics to measure prompt effectiveness:

1. **Accuracy Metrics:**
   - First-attempt success rate
   - Syntax error rate
   - Semantic correctness (requires user feedback)
   - Average confidence scores

2. **Efficiency Metrics:**
   - Average tokens used per request
   - Response time by provider
   - Retry rate
   - Token cost per successful query

3. **User Satisfaction:**
   - Query acceptance rate (user executes generated SQL)
   - Edit rate (user modifies before executing)
   - Error rate after execution

4. **Provider Comparison:**
   - Accuracy by provider
   - Speed by provider
   - Cost per successful query
   - Best provider for query type

---

## Conclusion

This comprehensive analysis provides:
- ✅ Specific prompt improvements with before/after examples
- ✅ Model-specific optimizations
- ✅ Token-efficient context strategies
- ✅ Enhanced error handling patterns
- ✅ Few-shot example library
- ✅ Confidence scoring framework
- ✅ Testing and monitoring approach

**Next Steps:**
1. Implement Phase 1 improvements (highest impact)
2. A/B test new prompts vs. old prompts
3. Measure accuracy and user satisfaction improvements
4. Iterate based on real-world performance data
5. Build prompt versioning system for continuous improvement
