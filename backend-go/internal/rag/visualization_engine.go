package rag

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

// ChartType represents different chart types
type ChartType string

const (
	ChartTypeLine      ChartType = "line"
	ChartTypeBar       ChartType = "bar"
	ChartTypePie       ChartType = "pie"
	ChartTypeScatter   ChartType = "scatter"
	ChartTypeHeatmap   ChartType = "heatmap"
	ChartTypeHistogram ChartType = "histogram"
	ChartTypeArea      ChartType = "area"
	ChartTypeGeo       ChartType = "geo"
	ChartTypeTreemap   ChartType = "treemap"
	ChartTypeSankey    ChartType = "sankey"
)

// VisualizationEngine handles intelligent visualization generation
type VisualizationEngine struct {
	detector    *ChartDetector
	recommender *VizRecommender
	generator   *ChartGenerator
	nlProcessor *NLToViz
	logger      *logrus.Logger
}

// ResultSet represents query result data
type ResultSet struct {
	Columns  []Column               `json:"columns"`
	Rows     [][]interface{}        `json:"rows"`
	RowCount int                    `json:"row_count"`
	Metadata map[string]interface{} `json:"metadata"`
}

// Column represents a column in the result set
type Column struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
}

// VizConfig represents visualization configuration
type VizConfig struct {
	ChartType   ChartType              `json:"chart_type"`
	Title       string                 `json:"title"`
	XAxis       AxisConfig             `json:"x_axis"`
	YAxis       AxisConfig             `json:"y_axis"`
	Series      []SeriesConfig         `json:"series"`
	Options     map[string]interface{} `json:"options"`
	Aggregation AggregationConfig      `json:"aggregation,omitempty"`
	Filters     []FilterConfig         `json:"filters,omitempty"`
}

// AxisConfig represents axis configuration
type AxisConfig struct {
	Column string   `json:"column"`
	Label  string   `json:"label"`
	Type   string   `json:"type"` // linear, category, time, log
	Format string   `json:"format,omitempty"`
	Min    *float64 `json:"min,omitempty"`
	Max    *float64 `json:"max,omitempty"`
}

// SeriesConfig represents a data series configuration
type SeriesConfig struct {
	Name        string                 `json:"name"`
	Column      string                 `json:"column"`
	Type        string                 `json:"type,omitempty"`
	Color       string                 `json:"color,omitempty"`
	Aggregation string                 `json:"aggregation,omitempty"`
	Options     map[string]interface{} `json:"options,omitempty"`
}

// AggregationConfig represents data aggregation configuration
type AggregationConfig struct {
	GroupBy    []string       `json:"group_by"`
	Metrics    []MetricConfig `json:"metrics"`
	TimeWindow string         `json:"time_window,omitempty"`
}

// MetricConfig represents a metric configuration
type MetricConfig struct {
	Column   string `json:"column"`
	Function string `json:"function"` // sum, avg, count, max, min
	Alias    string `json:"alias,omitempty"`
}

// FilterConfig represents a filter configuration
type FilterConfig struct {
	Column   string      `json:"column"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// VizRequest represents a visualization request
type VizRequest struct {
	Prompt      string                 `json:"prompt"`
	Data        *ResultSet             `json:"data,omitempty"`
	Preferences map[string]interface{} `json:"preferences,omitempty"`
}

// VizRecommendation represents a visualization recommendation
type VizRecommendation struct {
	ChartType  ChartType  `json:"chart_type"`
	Confidence float32    `json:"confidence"`
	Reason     string     `json:"reason"`
	Config     *VizConfig `json:"config"`
	Preview    string     `json:"preview,omitempty"`
}

// NewVisualizationEngine creates a new visualization engine
func NewVisualizationEngine(logger *logrus.Logger) *VisualizationEngine {
	return &VisualizationEngine{
		detector:    NewChartDetector(logger),
		recommender: NewVizRecommender(logger),
		generator:   NewChartGenerator(logger),
		nlProcessor: NewNLToViz(logger),
		logger:      logger,
	}
}

// DetectChartType automatically detects the best chart type for data
func (ve *VisualizationEngine) DetectChartType(data *ResultSet) ChartType {
	return ve.detector.Detect(data)
}

// GenerateVizConfig generates visualization configuration
func (ve *VisualizationEngine) GenerateVizConfig(data *ResultSet, chartType ChartType) *VizConfig {
	return ve.generator.Generate(data, chartType)
}

// AutoAggregate performs smart data aggregation
func (ve *VisualizationEngine) AutoAggregate(data *ResultSet) *ResultSet {
	// Detect if aggregation is needed
	if !ve.needsAggregation(data) {
		return data
	}

	// Determine aggregation strategy
	strategy := ve.determineAggregationStrategy(data)

	// Apply aggregation
	return ve.applyAggregation(data, strategy)
}

// ParseVizRequest parses natural language visualization request
func (ve *VisualizationEngine) ParseVizRequest(ctx context.Context, prompt string) (*VizRequest, error) {
	return ve.nlProcessor.Parse(ctx, prompt)
}

// Recommend generates visualization recommendations
func (ve *VisualizationEngine) Recommend(data *ResultSet, query string) []VizRecommendation {
	return ve.recommender.Recommend(data, query)
}

// GenerateFromNL generates visualization from natural language
func (ve *VisualizationEngine) GenerateFromNL(ctx context.Context, prompt string, data *ResultSet) (*VizConfig, error) {
	// Parse the visualization intent
	intent, err := ve.nlProcessor.ParseIntent(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// Generate chart based on intent
	_ = ve.nlProcessor.GenerateChart(intent, data)

	// TODO: Properly implement chart generation from NL
	return nil, fmt.Errorf("NL chart generation not yet implemented")
}

// Helper methods

func (ve *VisualizationEngine) needsAggregation(data *ResultSet) bool {
	// Check if data needs aggregation based on size and patterns
	if data.RowCount > 1000 {
		return true
	}

	// Check for repeated values that could be grouped
	for _, col := range data.Columns {
		if ve.hasRepeatableValues(data, col.Name) {
			return true
		}
	}

	return false
}

func (ve *VisualizationEngine) determineAggregationStrategy(data *ResultSet) *AggregationConfig {
	config := &AggregationConfig{
		GroupBy: []string{},
		Metrics: []MetricConfig{},
	}

	// Identify categorical columns for grouping
	for _, col := range data.Columns {
		if ve.isCategorical(col) {
			config.GroupBy = append(config.GroupBy, col.Name)
			if len(config.GroupBy) >= 2 {
				break
			}
		}
	}

	// Identify numeric columns for metrics
	for _, col := range data.Columns {
		if ve.isNumeric(col) {
			config.Metrics = append(config.Metrics, MetricConfig{
				Column:   col.Name,
				Function: "sum",
				Alias:    fmt.Sprintf("total_%s", col.Name),
			})
		}
	}

	// Check for time-based aggregation
	if timeCol := ve.findTimeColumn(data.Columns); timeCol != "" {
		config.GroupBy = []string{timeCol}
		config.TimeWindow = "day"
	}

	return config
}

func (ve *VisualizationEngine) applyAggregation(data *ResultSet, config *AggregationConfig) *ResultSet {
	// This would implement actual aggregation logic
	// For now, return the original data
	return data
}

func (ve *VisualizationEngine) hasRepeatableValues(data *ResultSet, column string) bool {
	// Check if column has repeatable values
	colIndex := -1
	for i, col := range data.Columns {
		if col.Name == column {
			colIndex = i
			break
		}
	}

	if colIndex < 0 {
		return false
	}

	uniqueValues := make(map[interface{}]bool)
	for _, row := range data.Rows {
		if colIndex < len(row) {
			uniqueValues[row[colIndex]] = true
		}
	}

	// If unique values are less than 50% of rows, likely repeatable
	return len(uniqueValues) < data.RowCount/2
}

func (ve *VisualizationEngine) isCategorical(col Column) bool {
	return col.Type == "string" || col.Type == "varchar" || col.Type == "text"
}

func (ve *VisualizationEngine) isNumeric(col Column) bool {
	numericTypes := []string{"int", "integer", "float", "decimal", "numeric", "double", "bigint"}
	colType := strings.ToLower(col.Type)
	for _, numType := range numericTypes {
		if strings.Contains(colType, numType) {
			return true
		}
	}
	return false
}

func (ve *VisualizationEngine) findTimeColumn(columns []Column) string {
	timeIndicators := []string{"date", "time", "timestamp", "created", "updated"}

	for _, col := range columns {
		colNameLower := strings.ToLower(col.Name)
		colTypeLower := strings.ToLower(col.Type)

		for _, indicator := range timeIndicators {
			if strings.Contains(colNameLower, indicator) || strings.Contains(colTypeLower, indicator) {
				return col.Name
			}
		}
	}

	return ""
}

// ChartDetector detects appropriate chart types
type ChartDetector struct {
	rules  []DetectionRule
	logger *logrus.Logger
}

// DetectionRule represents a chart detection rule
type DetectionRule struct {
	Name      string
	Condition func(*ResultSet) bool
	ChartType ChartType
	Priority  int
}

// NewChartDetector creates a new chart detector
func NewChartDetector(logger *logrus.Logger) *ChartDetector {
	detector := &ChartDetector{
		logger: logger,
		rules:  []DetectionRule{},
	}

	// Add detection rules
	detector.addDefaultRules()

	return detector
}

// Detect detects the best chart type for data
func (cd *ChartDetector) Detect(data *ResultSet) ChartType {
	// Check each rule
	for _, rule := range cd.rules {
		if rule.Condition(data) {
			cd.logger.WithFields(logrus.Fields{
				"rule":       rule.Name,
				"chart_type": rule.ChartType,
			}).Debug("Chart type detected")
			return rule.ChartType
		}
	}

	// Default to bar chart
	return ChartTypeBar
}

func (cd *ChartDetector) addDefaultRules() {
	cd.rules = []DetectionRule{
		{
			Name:      "time_series",
			Condition: cd.detectTimeSeries,
			ChartType: ChartTypeLine,
			Priority:  1,
		},
		{
			Name:      "distribution",
			Condition: cd.detectDistribution,
			ChartType: ChartTypeHistogram,
			Priority:  2,
		},
		{
			Name:      "categorical_comparison",
			Condition: cd.detectCategorical,
			ChartType: ChartTypeBar,
			Priority:  3,
		},
		{
			Name:      "proportion",
			Condition: cd.detectProportion,
			ChartType: ChartTypePie,
			Priority:  4,
		},
		{
			Name:      "correlation",
			Condition: cd.detectCorrelation,
			ChartType: ChartTypeScatter,
			Priority:  5,
		},
		{
			Name:      "geographic",
			Condition: cd.detectGeospatial,
			ChartType: ChartTypeGeo,
			Priority:  6,
		},
	}
}

func (cd *ChartDetector) detectTimeSeries(data *ResultSet) bool {
	// Check if data has time column and numeric values
	hasTimeColumn := false
	hasNumericColumn := false

	for _, col := range data.Columns {
		colType := strings.ToLower(col.Type)
		if strings.Contains(colType, "date") || strings.Contains(colType, "time") {
			hasTimeColumn = true
		}
		if strings.Contains(colType, "int") || strings.Contains(colType, "float") ||
			strings.Contains(colType, "decimal") || strings.Contains(colType, "numeric") {
			hasNumericColumn = true
		}
	}

	return hasTimeColumn && hasNumericColumn
}

func (cd *ChartDetector) detectDistribution(data *ResultSet) bool {
	// Check if data represents a distribution
	if len(data.Columns) == 1 || (len(data.Columns) == 2 && data.RowCount > 20) {
		// Single numeric column or value-frequency pair
		return true
	}
	return false
}

func (cd *ChartDetector) detectCategorical(data *ResultSet) bool {
	// Check if data has categorical and numeric columns
	hasCategorical := false
	hasNumeric := false

	for _, col := range data.Columns {
		if col.Type == "string" || col.Type == "varchar" {
			hasCategorical = true
		}
		if strings.Contains(strings.ToLower(col.Type), "int") ||
			strings.Contains(strings.ToLower(col.Type), "float") {
			hasNumeric = true
		}
	}

	return hasCategorical && hasNumeric && data.RowCount < 50
}

func (cd *ChartDetector) detectProportion(data *ResultSet) bool {
	// Check if data represents parts of a whole
	if len(data.Columns) == 2 && data.RowCount < 10 {
		// Category and value columns with few rows
		return true
	}
	return false
}

func (cd *ChartDetector) detectCorrelation(data *ResultSet) bool {
	// Check if data has two numeric columns
	numericColumns := 0
	for _, col := range data.Columns {
		if strings.Contains(strings.ToLower(col.Type), "int") ||
			strings.Contains(strings.ToLower(col.Type), "float") {
			numericColumns++
		}
	}
	return numericColumns >= 2 && data.RowCount > 10
}

func (cd *ChartDetector) detectGeospatial(data *ResultSet) bool {
	// Check for geographic indicators
	geoIndicators := []string{"lat", "lon", "latitude", "longitude", "country", "state", "city", "region"}

	for _, col := range data.Columns {
		colNameLower := strings.ToLower(col.Name)
		for _, indicator := range geoIndicators {
			if strings.Contains(colNameLower, indicator) {
				return true
			}
		}
	}

	return false
}

// Additional helper types

// VizRecommender generates visualization recommendations
type VizRecommender struct {
	logger *logrus.Logger
}

func NewVizRecommender(logger *logrus.Logger) *VizRecommender {
	return &VizRecommender{logger: logger}
}

func (vr *VizRecommender) Recommend(data *ResultSet, query string) []VizRecommendation {
	// Generate recommendations based on data and query
	recommendations := []VizRecommendation{}

	// This would implement actual recommendation logic
	// For now, return sample recommendations

	return recommendations
}

// ChartGenerator generates chart configurations
type ChartGenerator struct {
	logger *logrus.Logger
}

func NewChartGenerator(logger *logrus.Logger) *ChartGenerator {
	return &ChartGenerator{logger: logger}
}

func (cg *ChartGenerator) Generate(data *ResultSet, chartType ChartType) *VizConfig {
	// Generate configuration based on chart type
	config := &VizConfig{
		ChartType: chartType,
		Title:     "Generated Chart",
		Options:   make(map[string]interface{}),
	}

	// This would implement actual generation logic
	// For now, return basic config

	return config
}

// NLToViz handles natural language to visualization conversion
type NLToViz struct {
	logger *logrus.Logger
}

// VizIntent represents parsed visualization intent
type VizIntent struct {
	ChartType   string                 `json:"chart_type"`
	Aggregation string                 `json:"aggregation"`
	GroupBy     []string               `json:"group_by"`
	Metrics     []string               `json:"metrics"`
	Filters     []string               `json:"filters"`
	Options     map[string]interface{} `json:"options"`
}

// Chart represents a generated chart
type Chart struct {
	*VizConfig
}

func NewNLToViz(logger *logrus.Logger) *NLToViz {
	return &NLToViz{logger: logger}
}

func (nlv *NLToViz) Parse(ctx context.Context, prompt string) (*VizRequest, error) {
	// Parse natural language request
	request := &VizRequest{
		Prompt:      prompt,
		Preferences: make(map[string]interface{}),
	}

	// This would implement actual NLP parsing
	// For now, return basic request

	return request, nil
}

func (nlv *NLToViz) ParseIntent(ctx context.Context, prompt string) (*VizIntent, error) {
	// Parse visualization intent from prompt
	intent := &VizIntent{
		Options: make(map[string]interface{}),
	}

	// This would implement actual intent parsing
	// For now, return basic intent

	return intent, nil
}

func (nlv *NLToViz) GenerateChart(intent *VizIntent, data *ResultSet) *Chart {
	// Generate chart from intent and data
	chart := &Chart{
		VizConfig: &VizConfig{
			ChartType: ChartTypeBar,
			Title:     "Generated Chart",
			Options:   make(map[string]interface{}),
		},
	}

	// This would implement actual chart generation
	// For now, return basic chart

	return chart
}

func (nlv *NLToViz) ApplyDefaults(chart *Chart) *Chart {
	// Apply smart defaults to chart
	if chart.Title == "" {
		chart.Title = "Data Visualization"
	}

	// This would implement more defaults
	// For now, return chart as-is

	return chart
}
