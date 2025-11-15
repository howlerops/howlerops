package main

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/sql-studio/backend-go/pkg/ai"
	"github.com/sql-studio/backend-go/pkg/database"
	"github.com/sql-studio/backend-go/pkg/database/multiquery"
)

// AIQueryAgentRequest represents a request to the AI query agent workflow.
type AIQueryAgentRequest struct {
	SessionID     string                   `json:"sessionId"`
	Message       string                   `json:"message"`
	Provider      string                   `json:"provider"`
	Model         string                   `json:"model"`
	ConnectionID  string                   `json:"connectionId,omitempty"`
	ConnectionIDs []string                 `json:"connectionIds,omitempty"`
	SchemaContext string                   `json:"schemaContext,omitempty"`
	Context       string                   `json:"context,omitempty"`
	History       []AIMemoryMessagePayload `json:"history,omitempty"`
	SystemPrompt  string                   `json:"systemPrompt,omitempty"`
	Temperature   float64                  `json:"temperature,omitempty"`
	MaxTokens     int                      `json:"maxTokens,omitempty"`
	MaxRows       int                      `json:"maxRows,omitempty"`
	Page          int                      `json:"page,omitempty"`     // NEW: Current page number (1-indexed)
	PageSize      int                      `json:"pageSize,omitempty"` // NEW: Rows per page
}

// AIQueryAgentResponse aggregates the generated artefacts for a single turn.
type AIQueryAgentResponse struct {
	SessionID   string                 `json:"sessionId"`
	TurnID      string                 `json:"turnId"`
	Provider    string                 `json:"provider"`
	Model       string                 `json:"model"`
	Messages    []AIQueryAgentMessage  `json:"messages"`
	Error       string                 `json:"error,omitempty"`
	DurationMs  int64                  `json:"durationMs"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ExecutedSQL string                 `json:"executedSql,omitempty"`
}

// AIQueryAgentMessage represents a single agent response (orchestrator, sql generator, etc).
type AIQueryAgentMessage struct {
	ID          string                     `json:"id"`
	Agent       string                     `json:"agent"`
	Role        string                     `json:"role"`
	Title       string                     `json:"title,omitempty"`
	Content     string                     `json:"content"`
	CreatedAt   int64                      `json:"createdAt"`
	Attachments []AIQueryAgentAttachment   `json:"attachments,omitempty"`
	Metadata    map[string]interface{}     `json:"metadata,omitempty"`
	Warnings    []string                   `json:"warnings,omitempty"`
	Error       string                     `json:"error,omitempty"`
	Provider    string                     `json:"provider,omitempty"`
	Model       string                     `json:"model,omitempty"`
	TokensUsed  int                        `json:"tokensUsed,omitempty"`
	ElapsedMs   int64                      `json:"elapsedMs,omitempty"`
	Context     map[string]json.RawMessage `json:"context,omitempty"`
}

// AIQueryAgentAttachment represents rich content associated with a message.
type AIQueryAgentAttachment struct {
	Type       string                         `json:"type"`
	SQL        *AIQueryAgentSQLAttachment     `json:"sql,omitempty"`
	Result     *AIQueryAgentResultAttachment  `json:"result,omitempty"`
	Chart      *AIQueryAgentChartAttachment   `json:"chart,omitempty"`
	Report     *AIQueryAgentReportAttachment  `json:"report,omitempty"`
	Insight    *AIQueryAgentInsightAttachment `json:"insight,omitempty"`
	RawPayload map[string]interface{}         `json:"rawPayload,omitempty"`
}

// AIQueryAgentSQLAttachment contains generated SQL information.
type AIQueryAgentSQLAttachment struct {
	Query        string   `json:"query"`
	Explanation  string   `json:"explanation,omitempty"`
	Confidence   float64  `json:"confidence,omitempty"`
	ConnectionID string   `json:"connectionId,omitempty"`
	Warnings     []string `json:"warnings,omitempty"`
}

// AIQueryAgentResultAttachment contains a lightweight data preview.
type AIQueryAgentResultAttachment struct {
	Columns         []string                 `json:"columns"`
	Rows            []map[string]interface{} `json:"rows"`
	RowCount        int64                    `json:"rowCount"`
	ExecutionTimeMs int64                    `json:"executionTimeMs"`
	Limited         bool                     `json:"limited"`
	ConnectionID    string                   `json:"connectionId,omitempty"`
	// Pagination metadata
	TotalRows  int64 `json:"totalRows,omitempty"`  // NEW: Total rows available
	Page       int   `json:"page,omitempty"`       // NEW: Current page
	PageSize   int   `json:"pageSize,omitempty"`   // NEW: Page size
	TotalPages int   `json:"totalPages,omitempty"` // NEW: Total pages
	HasMore    bool  `json:"hasMore,omitempty"`    // NEW: More pages available
}

// AIQueryAgentChartAttachment represents a chart suggestion produced by the agent.
type AIQueryAgentChartAttachment struct {
	Type          string           `json:"type"`
	XField        string           `json:"xField"`
	YFields       []string         `json:"yFields"`
	SeriesField   string           `json:"seriesField,omitempty"`
	Title         string           `json:"title,omitempty"`
	Description   string           `json:"description,omitempty"`
	Recommended   bool             `json:"recommended"`
	PreviewValues []map[string]any `json:"previewValues,omitempty"`
}

// AIQueryAgentReportAttachment is a formatted report.
type AIQueryAgentReportAttachment struct {
	Format string `json:"format"`
	Body   string `json:"body"`
	Title  string `json:"title,omitempty"`
}

// AIQueryAgentInsightAttachment holds structured insights.
type AIQueryAgentInsightAttachment struct {
	Highlights []string `json:"highlights"`
}

// ReadOnlyQueryResult represents a guarded SELECT output.
type ReadOnlyQueryResult struct {
	Columns         []string                 `json:"columns"`
	Rows            []map[string]interface{} `json:"rows"`
	RowCount        int64                    `json:"rowCount"`
	ExecutionTimeMs int64                    `json:"executionTimeMs"`
	Limited         bool                     `json:"limited"`
	ConnectionID    string                   `json:"connectionId"`
	// Pagination metadata
	TotalRows  int64 `json:"totalRows,omitempty"`  // Total rows available (unpaged)
	Page       int   `json:"page,omitempty"`       // Current page number
	PageSize   int   `json:"pageSize,omitempty"`   // Rows per page
	TotalPages int   `json:"totalPages,omitempty"` // Total pages available
	HasMore    bool  `json:"hasMore,omitempty"`    // More pages available
	Offset     int   `json:"offset,omitempty"`     // Current offset
}

type queryAgentEvent struct {
	SessionID string               `json:"sessionId"`
	TurnID    string               `json:"turnId"`
	Status    string               `json:"status"`
	Message   *AIQueryAgentMessage `json:"message,omitempty"`
	Error     string               `json:"error,omitempty"`
}

type orchestratorPlan struct {
	Reply       string `json:"reply"`
	RequiresSQL bool   `json:"requires_sql"`
	Reason      string `json:"reason,omitempty"`
}

// StreamAIQueryAgent coordinates a multi-agent workflow to satisfy a user query.
func (a *App) StreamAIQueryAgent(req AIQueryAgentRequest) (*AIQueryAgentResponse, error) {
	start := time.Now()

	if a.aiService == nil {
		return nil, fmt.Errorf("AI service not configured")
	}

	message := strings.TrimSpace(req.Message)
	if message == "" {
		return nil, fmt.Errorf("prompt cannot be empty")
	}

	provider := strings.TrimSpace(req.Provider)
	if provider == "" {
		return nil, fmt.Errorf("provider is required")
	}

	model := strings.TrimSpace(req.Model)
	req.Provider = provider
	req.Model = model
	req.Message = message
	if req.MaxRows <= 0 {
		req.MaxRows = 200
	}
	if req.MaxTokens <= 0 {
		req.MaxTokens = 1200
	}
	if req.Temperature <= 0 {
		req.Temperature = 0.2
	}

	connectionID := selectPrimaryConnection(req.ConnectionID, req.ConnectionIDs)
	historyContext := buildHistoryContext(req.History, 6)
	fullContext := strings.TrimSpace(strings.Join(filterNonEmpty([]string{
		req.SchemaContext,
		req.Context,
		historyContext,
	}), "\n\n"))

	turnID := uuid.NewString()
	response := &AIQueryAgentResponse{
		SessionID:   req.SessionID,
		TurnID:      turnID,
		Provider:    provider,
		Model:       model,
		Messages:    make([]AIQueryAgentMessage, 0, 6),
		Metadata:    map[string]interface{}{},
		ExecutedSQL: "",
	}

	a.emitQueryAgentEvent(queryAgentEvent{
		SessionID: req.SessionID,
		TurnID:    turnID,
		Status:    "started",
	})

	userMessage := AIQueryAgentMessage{
		ID:        uuid.NewString(),
		Agent:     "user",
		Role:      "user",
		Content:   message,
		CreatedAt: time.Now().UnixMilli(),
	}
	response.Messages = append(response.Messages, userMessage)
	a.emitQueryAgentEvent(queryAgentEvent{
		SessionID: req.SessionID,
		TurnID:    turnID,
		Status:    "message",
		Message:   &userMessage,
	})
	plan, planMsg, planErr := a.buildOrchestratorPlan(req, message, fullContext)
	if planErr != nil {
		a.logger.WithError(planErr).Debug("orchestrator plan unavailable, proceeding with default SQL workflow")
	}
	if planMsg != nil {
		response.Messages = append(response.Messages, *planMsg)
		a.emitQueryAgentEvent(queryAgentEvent{
			SessionID: req.SessionID,
			TurnID:    turnID,
			Status:    "message",
			Message:   planMsg,
		})
	}

	shouldRunSQL := connectionID != ""
	if plan != nil {
		shouldRunSQL = plan.RequiresSQL && connectionID != ""
	}

	if !shouldRunSQL {
		if plan != nil && plan.RequiresSQL && connectionID == "" {
			info := AIQueryAgentMessage{
				ID:        uuid.NewString(),
				Agent:     "orchestrator",
				Role:      "assistant",
				Title:     "Connection Required",
				Content:   "I need an active database connection to run queries. Connect a database and try again.",
				CreatedAt: time.Now().UnixMilli(),
			}
			response.Messages = append(response.Messages, info)
			a.emitQueryAgentEvent(queryAgentEvent{
				SessionID: req.SessionID,
				TurnID:    turnID,
				Status:    "message",
				Message:   &info,
			})
		}

		response.DurationMs = time.Since(start).Milliseconds()
		a.emitQueryAgentEvent(queryAgentEvent{
			SessionID: req.SessionID,
			TurnID:    turnID,
			Status:    "completed",
		})
		return response, nil
	}

	sqlStart := time.Now()
	sqlResp, err := a.GenerateSQLFromNaturalLanguage(NLQueryRequest{
		Prompt:       message,
		ConnectionID: connectionID,
		Context:      fullContext,
		Provider:     provider,
		Model:        model,
		MaxTokens:    req.MaxTokens,
		Temperature:  req.Temperature,
	})
	if err != nil {
		return a.failQueryAgent(response, req, turnID, fmt.Errorf("sql generation failed: %w", err))
	}

	if sqlResp == nil || strings.TrimSpace(sqlResp.SQL) == "" {
		return a.failQueryAgent(response, req, turnID, fmt.Errorf("ai provider did not return SQL"))
	}

	sqlText := strings.TrimSpace(sqlResp.SQL)
	if !isLikelySQLStatement(sqlText) {
		errorMessage := AIQueryAgentMessage{
			ID:    uuid.NewString(),
			Agent: "sql_generator",
			Role:  "assistant",
			Title: "Provider Error",
			Content: fmt.Sprintf(
				"The AI provider returned an error instead of SQL:\n\n%s\n\nPlease retry in a moment, switch models/providers, or verify your usage limits.",
				sqlText,
			),
			CreatedAt: time.Now().UnixMilli(),
			Error:     sqlText,
		}

		response.Messages = append(response.Messages, errorMessage)
		a.emitQueryAgentEvent(queryAgentEvent{
			SessionID: req.SessionID,
			TurnID:    turnID,
			Status:    "message",
			Message:   &errorMessage,
		})

		response.DurationMs = time.Since(start).Milliseconds()
		a.emitQueryAgentEvent(queryAgentEvent{
			SessionID: req.SessionID,
			TurnID:    turnID,
			Status:    "completed",
		})
		return response, nil
	}

	sqlMessage := AIQueryAgentMessage{
		ID:        uuid.NewString(),
		Agent:     "sql_generator",
		Role:      "assistant",
		Title:     "SQL Generator",
		Content:   "Generated SQL based on the request.",
		CreatedAt: time.Now().UnixMilli(),
		ElapsedMs: time.Since(sqlStart).Milliseconds(),
		Attachments: []AIQueryAgentAttachment{
			{
				Type: "sql",
				SQL: &AIQueryAgentSQLAttachment{
					Query:        sqlResp.SQL,
					Explanation:  sqlResp.Explanation,
					Confidence:   sqlResp.Confidence,
					ConnectionID: connectionID,
				},
			},
		},
	}
	response.Messages = append(response.Messages, sqlMessage)
	response.ExecutedSQL = sqlResp.SQL
	a.emitQueryAgentEvent(queryAgentEvent{
		SessionID: req.SessionID,
		TurnID:    turnID,
		Status:    "message",
		Message:   &sqlMessage,
	})

	var preview *ReadOnlyQueryResult
	if connectionID != "" && strings.TrimSpace(sqlResp.SQL) != "" {
		queryStart := time.Now()
		// Calculate offset from page and pageSize
		offset := 0
		pageSize := req.MaxRows
		if req.Page > 0 && req.PageSize > 0 {
			pageSize = req.PageSize
			offset = (req.Page - 1) * pageSize
		}
		preview, err = a.ExecuteReadOnlyQueryWithPagination(connectionID, sqlResp.SQL, pageSize, offset, 30*time.Second)
		if err != nil {
			warning := fmt.Sprintf("Query execution failed: %v", err)
			errorMessage := AIQueryAgentMessage{
				ID:        uuid.NewString(),
				Agent:     "query_executor",
				Role:      "assistant",
				Title:     "Query Execution Error",
				Content:   warning,
				CreatedAt: time.Now().UnixMilli(),
				Error:     warning,
			}
			response.Messages = append(response.Messages, errorMessage)
			a.emitQueryAgentEvent(queryAgentEvent{
				SessionID: req.SessionID,
				TurnID:    turnID,
				Status:    "message",
				Message:   &errorMessage,
			})
			preview = nil
		} else {
			resultMessage := AIQueryAgentMessage{
				ID:        uuid.NewString(),
				Agent:     "query_executor",
				Role:      "assistant",
				Title:     "Read-Only Preview",
				Content:   summariseResult(preview),
				CreatedAt: time.Now().UnixMilli(),
				ElapsedMs: time.Since(queryStart).Milliseconds(),
				Attachments: []AIQueryAgentAttachment{
					{
						Type:   "result",
						Result: convertToResultAttachment(preview),
					},
				},
			}
			response.Messages = append(response.Messages, resultMessage)
			a.emitQueryAgentEvent(queryAgentEvent{
				SessionID: req.SessionID,
				TurnID:    turnID,
				Status:    "message",
				Message:   &resultMessage,
			})
		}
	}

	if preview != nil && preview.RowCount > 0 {
		if analysisMessage, analysisErr := a.generateAnalystMessage(req, sqlResp.SQL, preview); analysisErr == nil {
			response.Messages = append(response.Messages, analysisMessage)
			a.emitQueryAgentEvent(queryAgentEvent{
				SessionID: req.SessionID,
				TurnID:    turnID,
				Status:    "message",
				Message:   &analysisMessage,
			})
		} else {
			a.logger.WithError(analysisErr).Warn("data analyst agent failed")
		}

		if chartMessage, chartErr := a.generateChartMessage(req, sqlResp.SQL, preview); chartErr == nil && len(chartMessage.Attachments) > 0 {
			response.Messages = append(response.Messages, chartMessage)
			a.emitQueryAgentEvent(queryAgentEvent{
				SessionID: req.SessionID,
				TurnID:    turnID,
				Status:    "message",
				Message:   &chartMessage,
			})
		} else if chartErr != nil {
			a.logger.WithError(chartErr).Debug("chart agent skipped")
		}

		if reportMessage, reportErr := a.generateReportMessage(req, sqlResp.SQL, preview); reportErr == nil {
			response.Messages = append(response.Messages, reportMessage)
			a.emitQueryAgentEvent(queryAgentEvent{
				SessionID: req.SessionID,
				TurnID:    turnID,
				Status:    "message",
				Message:   &reportMessage,
			})
		} else if reportErr != nil {
			a.logger.WithError(reportErr).Debug("report agent skipped")
		}
	}

	if explainMessage, explainErr := a.generateExplanationMessage(req, sqlResp.SQL); explainErr == nil {
		response.Messages = append(response.Messages, explainMessage)
		a.emitQueryAgentEvent(queryAgentEvent{
			SessionID: req.SessionID,
			TurnID:    turnID,
			Status:    "message",
			Message:   &explainMessage,
		})
	} else if explainErr != nil {
		a.logger.WithError(explainErr).Warn("query explainer agent failed")
	}

	response.DurationMs = time.Since(start).Milliseconds()

	a.emitQueryAgentEvent(queryAgentEvent{
		SessionID: req.SessionID,
		TurnID:    turnID,
		Status:    "completed",
	})

	return response, nil
}

func (a *App) failQueryAgent(resp *AIQueryAgentResponse, req AIQueryAgentRequest, turnID string, err error) (*AIQueryAgentResponse, error) {
	resp.Error = err.Error()
	a.emitQueryAgentEvent(queryAgentEvent{
		SessionID: req.SessionID,
		TurnID:    turnID,
		Status:    "error",
		Error:     err.Error(),
	})
	return resp, err
}

func (a *App) generateAnalystMessage(req AIQueryAgentRequest, sql string, preview *ReadOnlyQueryResult) (AIQueryAgentMessage, error) {
	promptRows := formatRowsForPrompt(preview.Rows, preview.Columns, 15, 150)
	system := "You are a data analyst agent for Howlerops. Provide concise, actionable insights in bullet points (max 5) without restating obvious facts. Highlight anomalies, trends, or correlations."
	user := fmt.Sprintf(`User question: %s

SQL:
%s

Previewed rows:
%s

Summarize key findings in under 120 words.`, req.Message, sql, promptRows)

	resp, err := a.aiService.Chat(a.ctx, &ai.ChatRequest{
		Prompt:      user,
		System:      system,
		Provider:    req.Provider,
		Model:       req.Model,
		MaxTokens:   minInt(req.MaxTokens, 600),
		Temperature: clamp(req.Temperature+0.1, 0.1, 0.6),
		Metadata: map[string]string{
			"agent": "data_analyst",
		},
	})
	if err != nil {
		return AIQueryAgentMessage{}, err
	}

	insights := splitInsights(resp.Content)
	msg := AIQueryAgentMessage{
		ID:         uuid.NewString(),
		Agent:      "data_analyst",
		Role:       "assistant",
		Title:      "Data Insights",
		Content:    resp.Content,
		CreatedAt:  time.Now().UnixMilli(),
		Provider:   resp.Provider,
		Model:      resp.Model,
		TokensUsed: resp.TokensUsed,
		Attachments: []AIQueryAgentAttachment{
			{
				Type: "insight",
				Insight: &AIQueryAgentInsightAttachment{
					Highlights: insights,
				},
			},
		},
	}
	return msg, nil
}

func (a *App) generateChartMessage(req AIQueryAgentRequest, sql string, preview *ReadOnlyQueryResult) (AIQueryAgentMessage, error) {
	if len(preview.Columns) == 0 || len(preview.Rows) == 0 {
		return AIQueryAgentMessage{}, fmt.Errorf("no data for chart suggestion")
	}

	promptRows := formatRowsForPrompt(preview.Rows, preview.Columns, 20, 120)
	system := "You are a chart generation assistant. Respond with valid JSON describing a recommended chart for the provided data. Choose from types: line, area, bar, column, pie, donut, scatter. Ensure fields exist in the data."
	user := fmt.Sprintf(`User question: %s

SQL:
%s

Previewed rows:
%s

Respond with JSON like:
{
  "type": "bar",
  "xField": "column_name",
  "yFields": ["numeric_column"],
  "seriesField": "optional_series",
  "title": "Chart title",
  "description": "Brief caption"
}`, req.Message, sql, promptRows)

	resp, err := a.aiService.Chat(a.ctx, &ai.ChatRequest{
		Prompt:      user,
		System:      system,
		Provider:    req.Provider,
		Model:       req.Model,
		MaxTokens:   400,
		Temperature: clamp(req.Temperature, 0.1, 0.5),
		Metadata: map[string]string{
			"agent": "chart_generator",
		},
	})
	if err != nil {
		return AIQueryAgentMessage{}, err
	}

	spec, parseErr := parseChartSpec(resp.Content)
	if parseErr != nil {
		return AIQueryAgentMessage{}, parseErr
	}

	msg := AIQueryAgentMessage{
		ID:         uuid.NewString(),
		Agent:      "chart_generator",
		Role:       "assistant",
		Title:      "Chart Suggestion",
		Content:    spec.Description,
		CreatedAt:  time.Now().UnixMilli(),
		Provider:   resp.Provider,
		Model:      resp.Model,
		TokensUsed: resp.TokensUsed,
		Attachments: []AIQueryAgentAttachment{
			{
				Type: "chart",
				Chart: &AIQueryAgentChartAttachment{
					Type:        spec.Type,
					XField:      spec.XField,
					YFields:     spec.YFields,
					SeriesField: spec.SeriesField,
					Title:       spec.Title,
					Description: spec.Description,
					Recommended: true,
				},
			},
		},
	}
	return msg, nil
}

func (a *App) generateReportMessage(req AIQueryAgentRequest, sql string, preview *ReadOnlyQueryResult) (AIQueryAgentMessage, error) {
	promptRows := formatRowsForPrompt(preview.Rows, preview.Columns, 25, 140)
	system := "You are a report generator. Produce a concise Markdown report summarizing the data. Include sections: Objective, Key Metrics, Observations, Recommendations. Keep it under 200 words."
	user := fmt.Sprintf(`User request: %s

SQL:
%s

Previewed rows:
%s`, req.Message, sql, promptRows)

	resp, err := a.aiService.Chat(a.ctx, &ai.ChatRequest{
		Prompt:      user,
		System:      system,
		Provider:    req.Provider,
		Model:       req.Model,
		MaxTokens:   minInt(req.MaxTokens, 700),
		Temperature: clamp(req.Temperature, 0.2, 0.6),
		Metadata: map[string]string{
			"agent": "report_generator",
		},
	})
	if err != nil {
		return AIQueryAgentMessage{}, err
	}

	msg := AIQueryAgentMessage{
		ID:         uuid.NewString(),
		Agent:      "report_generator",
		Role:       "assistant",
		Title:      "Draft Report",
		Content:    resp.Content,
		CreatedAt:  time.Now().UnixMilli(),
		Provider:   resp.Provider,
		Model:      resp.Model,
		TokensUsed: resp.TokensUsed,
		Attachments: []AIQueryAgentAttachment{
			{
				Type: "report",
				Report: &AIQueryAgentReportAttachment{
					Format: "markdown",
					Body:   resp.Content,
					Title:  "AI Generated Report",
				},
			},
		},
	}
	return msg, nil
}

func (a *App) generateExplanationMessage(req AIQueryAgentRequest, sql string) (AIQueryAgentMessage, error) {
	system := "You are a SQL explainer agent. Explain what the query does in plain language, focusing on intent, filters, joins, and outputs. Avoid repeating the SQL verbatim."
	user := fmt.Sprintf(`Explain the following SQL query for a technical user:

%s`, sql)

	resp, err := a.aiService.Chat(a.ctx, &ai.ChatRequest{
		Prompt:      user,
		System:      system,
		Provider:    req.Provider,
		Model:       req.Model,
		MaxTokens:   400,
		Temperature: clamp(req.Temperature, 0.1, 0.4),
		Metadata: map[string]string{
			"agent": "query_explainer",
		},
	})
	if err != nil {
		return AIQueryAgentMessage{}, err
	}

	msg := AIQueryAgentMessage{
		ID:         uuid.NewString(),
		Agent:      "query_explainer",
		Role:       "assistant",
		Title:      "Query Explanation",
		Content:    resp.Content,
		CreatedAt:  time.Now().UnixMilli(),
		Provider:   resp.Provider,
		Model:      resp.Model,
		TokensUsed: resp.TokensUsed,
	}
	return msg, nil
}

func (a *App) emitQueryAgentEvent(event queryAgentEvent) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, "ai:query-agent:stream", event)
}

func selectPrimaryConnection(primary string, list []string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	for _, candidate := range list {
		if strings.TrimSpace(candidate) != "" {
			return candidate
		}
	}
	return ""
}

func buildHistoryContext(history []AIMemoryMessagePayload, limit int) string {
	if len(history) == 0 || limit <= 0 {
		return ""
	}

	count := len(history)
	if count > limit {
		history = history[count-limit:]
	}

	builder := strings.Builder{}
	builder.WriteString("Recent conversation:\n")
	for _, msg := range history {
		role := strings.ToUpper(msg.Role)
		if role == "" {
			role = "USER"
		}
		builder.WriteString(fmt.Sprintf("[%s] %s\n", role, strings.TrimSpace(msg.Content)))
	}
	return builder.String()
}

func filterNonEmpty(values []string) []string {
	result := make([]string, 0, len(values))
	for _, val := range values {
		if trimmed := strings.TrimSpace(val); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// ExecuteReadOnlyQuery validates and executes a SELECT-only query with safety controls.
func (a *App) ExecuteReadOnlyQuery(connectionID string, query string, maxRows int, timeout time.Duration) (*ReadOnlyQueryResult, error) {
	if a.databaseService == nil {
		return nil, fmt.Errorf("database service not available")
	}
	isMulti := isMultiDatabaseSQL(query)
	if strings.TrimSpace(connectionID) == "" && !isMulti {
		return nil, fmt.Errorf("connection ID is required")
	}

	clean := strings.TrimSpace(query)
	if clean == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	if isMultiDatabaseSQL(clean) {
		return a.executeMultiReadOnlyQuery(clean, maxRows, timeout)
	}

	if !isSelectOnly(clean) {
		return nil, fmt.Errorf("only read-only SELECT statements are permitted")
	}

	enforced, limited := enforceRowLimit(clean, maxRows)

	options := &database.QueryOptions{
		Timeout:  timeout,
		ReadOnly: true,
		Limit:    maxRows,
	}

	result, err := a.databaseService.ExecuteQuery(connectionID, enforced, options)
	if err != nil {
		return nil, err
	}

	rows := convertRows(result.Columns, result.Rows)

	return &ReadOnlyQueryResult{
		Columns:         append([]string(nil), result.Columns...),
		Rows:            rows,
		RowCount:        result.RowCount,
		ExecutionTimeMs: result.Duration.Milliseconds(),
		Limited:         limited || (maxRows > 0 && result.RowCount >= int64(maxRows)),
		ConnectionID:    connectionID,
	}, nil
}

// ExecuteReadOnlyQueryWithPagination executes a SELECT query with pagination support.
func (a *App) ExecuteReadOnlyQueryWithPagination(connectionID string, query string, pageSize int, offset int, timeout time.Duration) (*ReadOnlyQueryResult, error) {
	if a.databaseService == nil {
		return nil, fmt.Errorf("database service not available")
	}
	isMulti := isMultiDatabaseSQL(query)
	if strings.TrimSpace(connectionID) == "" && !isMulti {
		return nil, fmt.Errorf("connection ID is required")
	}

	clean := strings.TrimSpace(query)
	if clean == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	if isMultiDatabaseSQL(clean) {
		return a.executeMultiReadOnlyQueryWithPagination(clean, pageSize, offset, timeout)
	}

	if !isSelectOnly(clean) {
		return nil, fmt.Errorf("only read-only SELECT statements are permitted")
	}

	// Don't enforce limit in the query itself - use QueryOptions
	options := &database.QueryOptions{
		Timeout:  timeout,
		ReadOnly: true,
		Limit:    pageSize,
		Offset:   offset,
	}

	result, err := a.databaseService.ExecuteQuery(connectionID, clean, options)
	if err != nil {
		return nil, err
	}

	rows := convertRows(result.Columns, result.Rows)

	// Calculate pagination metadata
	totalRows := result.TotalRows
	if totalRows == 0 {
		totalRows = result.RowCount // Fallback if TotalRows not set
	}

	page := 1
	if pageSize > 0 && offset >= 0 {
		page = (offset / pageSize) + 1
	}

	totalPages := 0
	hasMore := false
	if pageSize > 0 && totalRows > 0 {
		totalPages = int((totalRows + int64(pageSize) - 1) / int64(pageSize))
		hasMore = page < totalPages
	}

	return &ReadOnlyQueryResult{
		Columns:         append([]string(nil), result.Columns...),
		Rows:            rows,
		RowCount:        result.RowCount,
		ExecutionTimeMs: result.Duration.Milliseconds(),
		Limited:         false, // Not limited when paginating
		ConnectionID:    connectionID,
		TotalRows:       totalRows,
		Page:            page,
		PageSize:        pageSize,
		TotalPages:      totalPages,
		HasMore:         hasMore,
		Offset:          offset,
	}, nil
}

func (a *App) executeMultiReadOnlyQueryWithPagination(query string, pageSize int, offset int, timeout time.Duration) (*ReadOnlyQueryResult, error) {
	options := &multiquery.Options{
		Timeout:  timeout,
		Strategy: multiquery.StrategyAuto,
		Limit:    pageSize,
		Offset:   offset,
	}

	resp, err := a.databaseService.ExecuteMultiDatabaseQuery(query, options)
	if err != nil {
		return nil, err
	}

	rows := convertRows(resp.Columns, resp.Rows)
	var durationMs int64
	if parsed, err := time.ParseDuration(resp.Duration); err == nil {
		durationMs = parsed.Milliseconds()
	}

	connectionLabel := strings.Join(resp.ConnectionsUsed, ",")

	// Calculate pagination metadata
	totalRows := resp.RowCount
	page := 1
	if pageSize > 0 && offset >= 0 {
		page = (offset / pageSize) + 1
	}

	totalPages := 0
	hasMore := false
	if pageSize > 0 && totalRows > 0 {
		totalPages = int((totalRows + int64(pageSize) - 1) / int64(pageSize))
		hasMore = page < totalPages
	}

	return &ReadOnlyQueryResult{
		Columns:         append([]string(nil), resp.Columns...),
		Rows:            rows,
		RowCount:        resp.RowCount,
		ExecutionTimeMs: durationMs,
		Limited:         false,
		ConnectionID:    connectionLabel,
		TotalRows:       totalRows,
		Page:            page,
		PageSize:        pageSize,
		TotalPages:      totalPages,
		HasMore:         hasMore,
		Offset:          offset,
	}, nil
}

func (a *App) executeMultiReadOnlyQuery(query string, maxRows int, timeout time.Duration) (*ReadOnlyQueryResult, error) {
	options := &multiquery.Options{
		Timeout:  timeout,
		Strategy: multiquery.StrategyAuto,
		Limit:    maxRows,
	}

	resp, err := a.databaseService.ExecuteMultiDatabaseQuery(query, options)
	if err != nil {
		return nil, err
	}

	rows := convertRows(resp.Columns, resp.Rows)
	var durationMs int64
	if parsed, err := time.ParseDuration(resp.Duration); err == nil {
		durationMs = parsed.Milliseconds()
	}

	connectionLabel := strings.Join(resp.ConnectionsUsed, ",")
	limited := maxRows > 0 && resp.RowCount >= int64(maxRows)

	return &ReadOnlyQueryResult{
		Columns:         append([]string(nil), resp.Columns...),
		Rows:            rows,
		RowCount:        resp.RowCount,
		ExecutionTimeMs: durationMs,
		Limited:         limited,
		ConnectionID:    connectionLabel,
	}, nil
}

var limitRegex = regexp.MustCompile(`(?i)\blimit\s+\d+`)

func enforceRowLimit(query string, maxRows int) (string, bool) {
	if maxRows <= 0 {
		return query, false
	}

	if limitRegex.MatchString(query) {
		return query, false
	}

	terminator := ""
	if strings.HasSuffix(query, ";") {
		query = strings.TrimSuffix(query, ";")
		terminator = ";"
	}

	return fmt.Sprintf("%s\nLIMIT %d%s", query, maxRows, terminator), true
}

func isSelectOnly(query string) bool {
	normalized := removeSQLComments(strings.TrimSpace(query))
	upper := strings.ToUpper(normalized)

	if strings.Count(upper, ";") > 0 && !strings.HasSuffix(strings.TrimSpace(upper), ";") {
		return false
	}

	if !(strings.HasPrefix(upper, "SELECT") || strings.HasPrefix(upper, "WITH") || strings.HasPrefix(upper, "SHOW") || strings.HasPrefix(upper, "DESCRIBE") || strings.HasPrefix(upper, "EXPLAIN")) {
		return false
	}

	disallowed := []string{
		"INSERT ",
		"UPDATE ",
		"DELETE ",
		"DROP ",
		"CREATE ",
		"ALTER ",
		"TRUNCATE ",
		"MERGE ",
		"GRANT ",
		"REVOKE ",
		"ATTACH ",
		"DETACH ",
	}

	for _, token := range disallowed {
		if strings.Contains(upper, token) {
			return false
		}
	}

	return true
}

func isMultiDatabaseSQL(query string) bool {
	return strings.Contains(query, "@")
}

func removeSQLComments(query string) string {
	lines := strings.Split(query, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if idx := strings.Index(line, "--"); idx >= 0 {
			line = line[:idx]
		}
		filtered = append(filtered, line)
	}
	clean := strings.Join(filtered, "\n")
	blockComment := regexp.MustCompile(`/\*.*?\*/`)
	return blockComment.ReplaceAllString(clean, " ")
}

func convertRows(columns []string, rows [][]interface{}) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		entry := make(map[string]interface{}, len(columns))
		for idx, col := range columns {
			var value interface{}
			if idx < len(row) {
				value = normalizeValue(row[idx])
			}
			entry[col] = value
		}
		result = append(result, entry)
	}
	return result
}

func convertToResultAttachment(preview *ReadOnlyQueryResult) *AIQueryAgentResultAttachment {
	if preview == nil {
		return nil
	}
	return &AIQueryAgentResultAttachment{
		Columns:         append([]string(nil), preview.Columns...),
		Rows:            preview.Rows,
		RowCount:        preview.RowCount,
		ExecutionTimeMs: preview.ExecutionTimeMs,
		Limited:         preview.Limited,
		ConnectionID:    preview.ConnectionID,
		TotalRows:       preview.TotalRows,
		Page:            preview.Page,
		PageSize:        preview.PageSize,
		TotalPages:      preview.TotalPages,
		HasMore:         preview.HasMore,
	}
}

func normalizeValue(value interface{}) interface{} {
	switch v := value.(type) {
	case time.Time:
		return v.Format(time.RFC3339)
	case []byte:
		return string(v)
	default:
		return v
	}
}

func summariseResult(preview *ReadOnlyQueryResult) string {
	if preview == nil {
		return "No preview available."
	}
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("Returned %d row(s) across %d column(s).", preview.RowCount, len(preview.Columns)))
	if preview.Limited {
		builder.WriteString(" Results were truncated to respect the safety limit.")
	}
	return builder.String()
}

func formatRowsForPrompt(rows []map[string]interface{}, columns []string, maxRows, maxCellChars int) string {
	if len(rows) == 0 {
		return "No rows available."
	}

	limit := minInt(maxRows, len(rows))
	lines := make([]string, 0, limit+1)

	header := strings.Join(columns, " | ")
	lines = append(lines, header)

	for i := 0; i < limit; i++ {
		row := rows[i]
		cells := make([]string, len(columns))
		for idx, col := range columns {
			value := ""
			if raw, exists := row[col]; exists && raw != nil {
				value = fmt.Sprintf("%v", raw)
				if len(value) > maxCellChars {
					value = value[:maxCellChars] + "…"
				}
			} else {
				value = "NULL"
			}
			cells[idx] = value
		}
		lines = append(lines, strings.Join(cells, " | "))
	}

	if len(rows) > limit {
		lines = append(lines, fmt.Sprintf("… %d more row(s) omitted.", len(rows)-limit))
	}

	return strings.Join(lines, "\n")
}

type chartSpec struct {
	Type        string   `json:"type"`
	XField      string   `json:"xField"`
	YFields     []string `json:"yFields"`
	SeriesField string   `json:"seriesField,omitempty"`
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
}

func parseChartSpec(output string) (*chartSpec, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return nil, fmt.Errorf("empty chart spec response")
	}

	start := strings.Index(output, "{")
	end := strings.LastIndex(output, "}")
	if start != -1 && end != -1 && end >= start {
		output = output[start : end+1]
	}

	var spec chartSpec
	if err := json.Unmarshal([]byte(output), &spec); err != nil {
		return nil, fmt.Errorf("failed to parse chart spec: %w", err)
	}

	spec.Type = strings.ToLower(strings.TrimSpace(spec.Type))
	spec.XField = strings.TrimSpace(spec.XField)
	spec.SeriesField = strings.TrimSpace(spec.SeriesField)
	spec.Title = strings.TrimSpace(spec.Title)
	spec.Description = strings.TrimSpace(spec.Description)

	if spec.Type == "" || spec.XField == "" || len(spec.YFields) == 0 {
		return nil, fmt.Errorf("incomplete chart specification")
	}

	return &spec, nil
}

func splitInsights(content string) []string {
	lines := strings.Split(content, "\n")
	insights := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(strings.TrimLeft(line, "-*•"))
		if trimmed != "" {
			insights = append(insights, trimmed)
		}
	}
	if len(insights) == 0 && strings.TrimSpace(content) != "" {
		insights = append(insights, strings.TrimSpace(content))
	}
	return insights
}

func clamp(value, min, max float64) float64 {
	return math.Max(min, math.Min(max, value))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func isLikelySQLStatement(sql string) bool {
	if sql == "" {
		return false
	}

	normalized := strings.ToUpper(strings.TrimSpace(sql))
	if len(normalized) == 0 {
		return false
	}

	prefixes := []string{
		"SELECT",
		"WITH",
		"INSERT",
		"UPDATE",
		"DELETE",
		"CREATE",
		"ALTER",
		"DROP",
		"SHOW",
		"DESCRIBE",
		"EXPLAIN",
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(normalized, prefix) {
			return true
		}
	}

	return false
}

func (a *App) buildOrchestratorPlan(req AIQueryAgentRequest, message, context string) (*orchestratorPlan, *AIQueryAgentMessage, error) {
	if a.aiService == nil {
		return nil, nil, fmt.Errorf("ai service unavailable")
	}

	systemPrompt := "You orchestrate database assistants for Howlerops. Reply ONLY with compact JSON: {\"reply\": string, \"requires_sql\": boolean}. Use reply for a helpful natural-language response. Set requires_sql to true only if running database SQL is necessary and a connection is available. Never add extra text or markdown."
	resp, err := a.aiService.Chat(a.ctx, &ai.ChatRequest{
		Prompt:      fmt.Sprintf("User message:\n%s\n\nContext:\n%s", message, context),
		System:      systemPrompt,
		Provider:    req.Provider,
		Model:       req.Model,
		MaxTokens:   minInt(req.MaxTokens, 300),
		Temperature: clamp(req.Temperature, 0.1, 0.5),
		Metadata: map[string]string{
			"agent": "orchestrator",
		},
	})
	if err != nil || resp == nil {
		if err == nil {
			err = fmt.Errorf("empty orchestrator response")
		}
		return nil, nil, err
	}

	content := strings.TrimSpace(resp.Content)
	if content == "" {
		return nil, nil, fmt.Errorf("empty orchestrator response")
	}

	var plan orchestratorPlan
	if json.Unmarshal([]byte(content), &plan) != nil {
		plan = orchestratorPlan{
			Reply:       content,
			RequiresSQL: strings.TrimSpace(req.ConnectionID) != "",
		}
	}

	plan.Reply = strings.TrimSpace(plan.Reply)
	if plan.Reply == "" {
		plan.Reply = "I'm ready to help with your request."
	}

	orchestratorMessage := AIQueryAgentMessage{
		ID:         uuid.NewString(),
		Agent:      "orchestrator",
		Role:       "assistant",
		Title:      "Assistant",
		Content:    plan.Reply,
		CreatedAt:  time.Now().UnixMilli(),
		Provider:   resp.Provider,
		Model:      resp.Model,
		TokensUsed: resp.TokensUsed,
	}

	return &plan, &orchestratorMessage, nil
}
