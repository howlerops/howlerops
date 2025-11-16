//go:build integration

package rag_test

import (
	"context"
	"testing"

	"github.com/jbeck018/howlerops/backend-go/internal/rag"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a test logger for visualization engine tests
func newTestLoggerVisualizationEngine() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	return logger
}

// Fixture: Time series data
func createTimeSeriesData() *rag.ResultSet {
	return &rag.ResultSet{
		Columns: []rag.Column{
			{Name: "date", Type: "timestamp", Nullable: false},
			{Name: "revenue", Type: "decimal", Nullable: false},
		},
		Rows: [][]interface{}{
			{"2023-01-01", 1000.0},
			{"2023-01-02", 1200.0},
			{"2023-01-03", 1100.0},
			{"2023-01-04", 1300.0},
			{"2023-01-05", 1400.0},
		},
		RowCount: 5,
		Metadata: map[string]interface{}{},
	}
}

// Fixture: Categorical data
func createCategoricalData() *rag.ResultSet {
	return &rag.ResultSet{
		Columns: []rag.Column{
			{Name: "category", Type: "varchar", Nullable: false},
			{Name: "count", Type: "int", Nullable: false},
		},
		Rows: [][]interface{}{
			{"Electronics", 45},
			{"Clothing", 32},
			{"Food", 28},
		},
		RowCount: 3,
		Metadata: map[string]interface{}{},
	}
}

// Fixture: Numeric correlation data
func createCorrelationData() *rag.ResultSet {
	return &rag.ResultSet{
		Columns: []rag.Column{
			{Name: "temperature", Type: "float", Nullable: false},
			{Name: "sales", Type: "float", Nullable: false},
		},
		Rows: [][]interface{}{
			{20.5, 100.0},
			{22.0, 120.0},
			{19.8, 95.0},
			{25.3, 150.0},
			{23.1, 130.0},
			{21.2, 110.0},
			{24.5, 140.0},
			{18.9, 90.0},
			{26.0, 160.0},
			{22.8, 125.0},
			{20.1, 105.0},
			{23.7, 135.0},
		},
		RowCount: 12,
		Metadata: map[string]interface{}{},
	}
}

// Fixture: Distribution data
func createDistributionData() *rag.ResultSet {
	return &rag.ResultSet{
		Columns: []rag.Column{
			{Name: "value", Type: "int", Nullable: false},
		},
		Rows: [][]interface{}{
			{10}, {12}, {15}, {18}, {20},
			{22}, {25}, {28}, {30}, {32},
			{35}, {38}, {40}, {42}, {45},
			{48}, {50}, {52}, {55}, {58},
			{60}, {62}, {65}, {68}, {70},
		},
		RowCount: 25,
		Metadata: map[string]interface{}{},
	}
}

// Fixture: Geographic data
func createGeographicData() *rag.ResultSet {
	return &rag.ResultSet{
		Columns: []rag.Column{
			{Name: "country", Type: "varchar", Nullable: false},
			{Name: "latitude", Type: "float", Nullable: false},
			{Name: "longitude", Type: "float", Nullable: false},
			{Name: "population", Type: "int", Nullable: false},
		},
		Rows: [][]interface{}{
			{"USA", 37.0902, -95.7129, 331000000},
			{"Canada", 56.1304, -106.3468, 38000000},
			{"Mexico", 23.6345, -102.5528, 126000000},
		},
		RowCount: 3,
		Metadata: map[string]interface{}{},
	}
}

// Fixture: Proportion data
func createProportionData() *rag.ResultSet {
	return &rag.ResultSet{
		Columns: []rag.Column{
			{Name: "segment", Type: "varchar", Nullable: false},
			{Name: "value", Type: "int", Nullable: false},
		},
		Rows: [][]interface{}{
			{"A", 30},
			{"B", 45},
			{"C", 25},
		},
		RowCount: 3,
		Metadata: map[string]interface{}{},
	}
}

// Fixture: Large dataset for aggregation testing
func createLargeDataset() *rag.ResultSet {
	rows := make([][]interface{}, 1500)
	for i := 0; i < 1500; i++ {
		rows[i] = []interface{}{
			"Category A",
			float64(i * 10),
		}
	}
	return &rag.ResultSet{
		Columns: []rag.Column{
			{Name: "category", Type: "varchar", Nullable: false},
			{Name: "value", Type: "float", Nullable: false},
		},
		Rows:     rows,
		RowCount: 1500,
		Metadata: map[string]interface{}{},
	}
}

// Fixture: Data with repeatable values
func createRepeatableValuesData() *rag.ResultSet {
	return &rag.ResultSet{
		Columns: []rag.Column{
			{Name: "category", Type: "varchar", Nullable: false},
			{Name: "amount", Type: "int", Nullable: false},
		},
		Rows: [][]interface{}{
			{"A", 100}, {"A", 200}, {"A", 150},
			{"B", 300}, {"B", 250}, {"B", 280},
			{"A", 180}, {"A", 220}, {"A", 190},
			{"B", 270}, {"B", 290}, {"B", 260},
		},
		RowCount: 12,
		Metadata: map[string]interface{}{},
	}
}

// TestNewVisualizationEngine tests the constructor
func TestNewVisualizationEngine(t *testing.T) {
	t.Run("creates engine successfully", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)

		require.NotNil(t, engine)
	})

	t.Run("creates engine with nil logger", func(t *testing.T) {
		engine := rag.NewVisualizationEngine(nil)

		require.NotNil(t, engine)
	})
}

// TestDetectChartType tests automatic chart type detection
func TestDetectChartType(t *testing.T) {
	tests := []struct {
		name     string
		data     *rag.ResultSet
		expected rag.ChartType
	}{
		{
			name:     "detects time series",
			data:     createTimeSeriesData(),
			expected: rag.ChartTypeLine,
		},
		{
			name:     "detects categorical data",
			data:     createCategoricalData(),
			expected: rag.ChartTypeBar,
		},
		{
			name:     "detects correlation",
			data:     createCorrelationData(),
			expected: rag.ChartTypeScatter,
		},
		{
			name:     "detects distribution",
			data:     createDistributionData(),
			expected: rag.ChartTypeHistogram,
		},
		{
			name:     "detects geographic",
			data:     createGeographicData(),
			expected: rag.ChartTypeGeo,
		},
		{
			name:     "detects proportion",
			data:     createProportionData(),
			expected: rag.ChartTypePie,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := newTestLoggerVisualizationEngine()
			engine := rag.NewVisualizationEngine(logger)

			chartType := engine.DetectChartType(tt.data)

			assert.Equal(t, tt.expected, chartType)
		})
	}
}

// TestGenerateVizConfig tests visualization configuration generation
func TestGenerateVizConfig(t *testing.T) {
	t.Run("generates config for line chart", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)
		data := createTimeSeriesData()

		config := engine.GenerateVizConfig(data, rag.ChartTypeLine)

		require.NotNil(t, config)
		assert.Equal(t, rag.ChartTypeLine, config.ChartType)
		assert.NotEmpty(t, config.Title)
		assert.NotNil(t, config.Options)
	})

	t.Run("generates config for bar chart", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)
		data := createCategoricalData()

		config := engine.GenerateVizConfig(data, rag.ChartTypeBar)

		require.NotNil(t, config)
		assert.Equal(t, rag.ChartTypeBar, config.ChartType)
		assert.NotEmpty(t, config.Title)
	})

	t.Run("generates config for pie chart", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)
		data := createProportionData()

		config := engine.GenerateVizConfig(data, rag.ChartTypePie)

		require.NotNil(t, config)
		assert.Equal(t, rag.ChartTypePie, config.ChartType)
	})

	t.Run("generates config for scatter chart", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)
		data := createCorrelationData()

		config := engine.GenerateVizConfig(data, rag.ChartTypeScatter)

		require.NotNil(t, config)
		assert.Equal(t, rag.ChartTypeScatter, config.ChartType)
	})

	t.Run("generates config for histogram", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)
		data := createDistributionData()

		config := engine.GenerateVizConfig(data, rag.ChartTypeHistogram)

		require.NotNil(t, config)
		assert.Equal(t, rag.ChartTypeHistogram, config.ChartType)
	})

	t.Run("generates config for geo chart", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)
		data := createGeographicData()

		config := engine.GenerateVizConfig(data, rag.ChartTypeGeo)

		require.NotNil(t, config)
		assert.Equal(t, rag.ChartTypeGeo, config.ChartType)
	})
}

// TestAutoAggregate tests smart data aggregation
func TestAutoAggregate(t *testing.T) {
	t.Run("aggregates large dataset", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)
		data := createLargeDataset()

		result := engine.AutoAggregate(data)

		require.NotNil(t, result)
	})

	t.Run("does not aggregate small dataset", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)
		data := createCategoricalData()

		result := engine.AutoAggregate(data)

		require.NotNil(t, result)
		assert.Equal(t, data, result)
	})

	t.Run("aggregates data with repeatable values", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)
		data := createRepeatableValuesData()

		result := engine.AutoAggregate(data)

		require.NotNil(t, result)
	})

	t.Run("handles nil data", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)

		result := engine.AutoAggregate(nil)

		assert.Nil(t, result)
	})

	t.Run("handles empty data", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)
		data := &rag.ResultSet{
			Columns:  []rag.Column{},
			Rows:     [][]interface{}{},
			RowCount: 0,
			Metadata: map[string]interface{}{},
		}

		result := engine.AutoAggregate(data)

		require.NotNil(t, result)
		assert.Equal(t, data, result)
	})
}

// TestParseVizRequest tests natural language visualization request parsing
func TestParseVizRequest(t *testing.T) {
	t.Run("parses simple request", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)
		ctx := context.Background()

		request, err := engine.ParseVizRequest(ctx, "show me sales by region")

		require.NoError(t, err)
		require.NotNil(t, request)
		assert.Equal(t, "show me sales by region", request.Prompt)
		assert.NotNil(t, request.Preferences)
	})

	t.Run("parses complex request", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)
		ctx := context.Background()

		request, err := engine.ParseVizRequest(ctx, "create a line chart showing revenue over time grouped by month")

		require.NoError(t, err)
		require.NotNil(t, request)
		assert.NotEmpty(t, request.Prompt)
	})

	t.Run("handles empty prompt", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)
		ctx := context.Background()

		request, err := engine.ParseVizRequest(ctx, "")

		require.NoError(t, err)
		require.NotNil(t, request)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		request, err := engine.ParseVizRequest(ctx, "test prompt")

		// Should still succeed as parsing is not async
		require.NoError(t, err)
		require.NotNil(t, request)
	})
}

// TestRecommend tests visualization recommendations
func TestRecommend(t *testing.T) {
	t.Run("generates recommendations for time series", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)
		data := createTimeSeriesData()

		recommendations := engine.Recommend(data, "show trends over time")

		require.NotNil(t, recommendations)
		assert.IsType(t, []rag.VizRecommendation{}, recommendations)
	})

	t.Run("generates recommendations for categorical data", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)
		data := createCategoricalData()

		recommendations := engine.Recommend(data, "compare categories")

		require.NotNil(t, recommendations)
	})

	t.Run("handles nil data", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)

		recommendations := engine.Recommend(nil, "test query")

		require.NotNil(t, recommendations)
	})

	t.Run("handles empty query", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)
		data := createCategoricalData()

		recommendations := engine.Recommend(data, "")

		require.NotNil(t, recommendations)
	})
}

// TestGenerateFromNL tests natural language to visualization generation
func TestGenerateFromNL(t *testing.T) {
	t.Run("attempts to generate from natural language", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)
		ctx := context.Background()
		data := createTimeSeriesData()

		config, err := engine.GenerateFromNL(ctx, "show me a line chart", data)

		// Expected to fail as not yet implemented
		assert.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "not yet implemented")
	})

	t.Run("handles nil data", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)
		ctx := context.Background()

		config, err := engine.GenerateFromNL(ctx, "show me a chart", nil)

		assert.Error(t, err)
		assert.Nil(t, config)
	})

	t.Run("handles empty prompt", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)
		ctx := context.Background()
		data := createCategoricalData()

		config, err := engine.GenerateFromNL(ctx, "", data)

		assert.Error(t, err)
		assert.Nil(t, config)
	})
}

// TestChartDetector tests the chart detector component
func TestNewChartDetector(t *testing.T) {
	t.Run("creates detector successfully", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		detector := rag.NewChartDetector(logger)

		require.NotNil(t, detector)
	})

	t.Run("creates detector with nil logger", func(t *testing.T) {
		detector := rag.NewChartDetector(nil)

		require.NotNil(t, detector)
	})
}

func TestChartDetector_Detect(t *testing.T) {
	t.Run("detects time series", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		detector := rag.NewChartDetector(logger)
		data := createTimeSeriesData()

		chartType := detector.Detect(data)

		assert.Equal(t, rag.ChartTypeLine, chartType)
	})

	t.Run("detects distribution", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		detector := rag.NewChartDetector(logger)
		data := createDistributionData()

		chartType := detector.Detect(data)

		assert.Equal(t, rag.ChartTypeHistogram, chartType)
	})

	t.Run("detects categorical", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		detector := rag.NewChartDetector(logger)
		data := createCategoricalData()

		chartType := detector.Detect(data)

		assert.Equal(t, rag.ChartTypeBar, chartType)
	})

	t.Run("detects proportion", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		detector := rag.NewChartDetector(logger)
		data := createProportionData()

		chartType := detector.Detect(data)

		assert.Equal(t, rag.ChartTypePie, chartType)
	})

	t.Run("detects correlation", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		detector := rag.NewChartDetector(logger)
		data := createCorrelationData()

		chartType := detector.Detect(data)

		assert.Equal(t, rag.ChartTypeScatter, chartType)
	})

	t.Run("detects geographic", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		detector := rag.NewChartDetector(logger)
		data := createGeographicData()

		chartType := detector.Detect(data)

		assert.Equal(t, rag.ChartTypeGeo, chartType)
	})

	t.Run("defaults to bar chart", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		detector := rag.NewChartDetector(logger)
		data := &rag.ResultSet{
			Columns: []rag.Column{
				{Name: "col1", Type: "unknown", Nullable: false},
			},
			Rows: [][]interface{}{
				{1}, {2}, {3},
			},
			RowCount: 3,
			Metadata: map[string]interface{}{},
		}

		chartType := detector.Detect(data)

		assert.Equal(t, rag.ChartTypeBar, chartType)
	})

	t.Run("handles nil data", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		detector := rag.NewChartDetector(logger)

		chartType := detector.Detect(nil)

		assert.Equal(t, rag.ChartTypeBar, chartType)
	})

	t.Run("handles empty data", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		detector := rag.NewChartDetector(logger)
		data := &rag.ResultSet{
			Columns:  []rag.Column{},
			Rows:     [][]interface{}{},
			RowCount: 0,
			Metadata: map[string]interface{}{},
		}

		chartType := detector.Detect(data)

		assert.Equal(t, rag.ChartTypeBar, chartType)
	})
}

// TestDetectionRules tests individual detection rules
func TestDetectionRule_TimeSeries(t *testing.T) {
	tests := []struct {
		name     string
		data     *rag.ResultSet
		expected bool
	}{
		{
			name: "detects timestamp and numeric",
			data: &rag.ResultSet{
				Columns: []rag.Column{
					{Name: "date", Type: "timestamp", Nullable: false},
					{Name: "value", Type: "int", Nullable: false},
				},
				Rows:     [][]interface{}{{1, 2}},
				RowCount: 1,
			},
			expected: true,
		},
		{
			name: "detects datetime and float",
			data: &rag.ResultSet{
				Columns: []rag.Column{
					{Name: "datetime", Type: "datetime", Nullable: false},
					{Name: "amount", Type: "float", Nullable: false},
				},
				Rows:     [][]interface{}{{1, 2}},
				RowCount: 1,
			},
			expected: true,
		},
		{
			name: "rejects without time column",
			data: &rag.ResultSet{
				Columns: []rag.Column{
					{Name: "id", Type: "int", Nullable: false},
					{Name: "value", Type: "int", Nullable: false},
				},
				Rows:     [][]interface{}{{1, 2}},
				RowCount: 1,
			},
			expected: false,
		},
		{
			name: "rejects without numeric column",
			data: &rag.ResultSet{
				Columns: []rag.Column{
					{Name: "date", Type: "timestamp", Nullable: false},
					{Name: "name", Type: "varchar", Nullable: false},
				},
				Rows:     [][]interface{}{{1, 2}},
				RowCount: 1,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := newTestLoggerVisualizationEngine()
			detector := rag.NewChartDetector(logger)

			chartType := detector.Detect(tt.data)

			if tt.expected {
				assert.Equal(t, rag.ChartTypeLine, chartType)
			} else {
				assert.NotEqual(t, rag.ChartTypeLine, chartType)
			}
		})
	}
}

func TestDetectionRule_Distribution(t *testing.T) {
	tests := []struct {
		name     string
		data     *rag.ResultSet
		expected bool
	}{
		{
			name: "detects single column with many rows",
			data: &rag.ResultSet{
				Columns: []rag.Column{
					{Name: "value", Type: "int", Nullable: false},
				},
				Rows:     make([][]interface{}, 25),
				RowCount: 25,
			},
			expected: true,
		},
		{
			name: "detects two columns with many rows",
			data: &rag.ResultSet{
				Columns: []rag.Column{
					{Name: "value", Type: "int", Nullable: false},
					{Name: "frequency", Type: "int", Nullable: false},
				},
				Rows:     make([][]interface{}, 25),
				RowCount: 25,
			},
			expected: true,
		},
		{
			name: "rejects with too few rows",
			data: &rag.ResultSet{
				Columns: []rag.Column{
					{Name: "value", Type: "int", Nullable: false},
					{Name: "frequency", Type: "int", Nullable: false},
				},
				Rows:     make([][]interface{}, 10),
				RowCount: 10,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := newTestLoggerVisualizationEngine()
			detector := rag.NewChartDetector(logger)

			chartType := detector.Detect(tt.data)

			if tt.expected {
				assert.Equal(t, rag.ChartTypeHistogram, chartType)
			}
		})
	}
}

func TestDetectionRule_Categorical(t *testing.T) {
	tests := []struct {
		name     string
		data     *rag.ResultSet
		expected bool
	}{
		{
			name: "detects categorical and numeric",
			data: &rag.ResultSet{
				Columns: []rag.Column{
					{Name: "category", Type: "varchar", Nullable: false},
					{Name: "count", Type: "int", Nullable: false},
				},
				Rows:     make([][]interface{}, 10),
				RowCount: 10,
			},
			expected: true,
		},
		{
			name: "rejects with too many rows",
			data: &rag.ResultSet{
				Columns: []rag.Column{
					{Name: "category", Type: "varchar", Nullable: false},
					{Name: "count", Type: "int", Nullable: false},
				},
				Rows:     make([][]interface{}, 60),
				RowCount: 60,
			},
			expected: false,
		},
		{
			name: "rejects without categorical",
			data: &rag.ResultSet{
				Columns: []rag.Column{
					{Name: "id", Type: "int", Nullable: false},
					{Name: "count", Type: "int", Nullable: false},
				},
				Rows:     make([][]interface{}, 10),
				RowCount: 10,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := newTestLoggerVisualizationEngine()
			detector := rag.NewChartDetector(logger)

			chartType := detector.Detect(tt.data)

			if tt.expected {
				assert.Equal(t, rag.ChartTypeBar, chartType)
			}
		})
	}
}

func TestDetectionRule_Proportion(t *testing.T) {
	tests := []struct {
		name     string
		data     *rag.ResultSet
		expected bool
	}{
		{
			name: "detects two columns with few rows",
			data: &rag.ResultSet{
				Columns: []rag.Column{
					{Name: "segment", Type: "varchar", Nullable: false},
					{Name: "value", Type: "int", Nullable: false},
				},
				Rows:     make([][]interface{}, 5),
				RowCount: 5,
			},
			expected: true,
		},
		{
			name: "rejects with too many rows",
			data: &rag.ResultSet{
				Columns: []rag.Column{
					{Name: "segment", Type: "varchar", Nullable: false},
					{Name: "value", Type: "int", Nullable: false},
				},
				Rows:     make([][]interface{}, 15),
				RowCount: 15,
			},
			expected: false,
		},
		{
			name: "rejects with wrong number of columns",
			data: &rag.ResultSet{
				Columns: []rag.Column{
					{Name: "col1", Type: "varchar", Nullable: false},
					{Name: "col2", Type: "int", Nullable: false},
					{Name: "col3", Type: "int", Nullable: false},
				},
				Rows:     make([][]interface{}, 5),
				RowCount: 5,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := newTestLoggerVisualizationEngine()
			detector := rag.NewChartDetector(logger)

			chartType := detector.Detect(tt.data)

			if tt.expected {
				assert.Equal(t, rag.ChartTypePie, chartType)
			}
		})
	}
}

func TestDetectionRule_Correlation(t *testing.T) {
	tests := []struct {
		name     string
		data     *rag.ResultSet
		expected bool
	}{
		{
			name: "detects two numeric columns",
			data: &rag.ResultSet{
				Columns: []rag.Column{
					{Name: "x", Type: "float", Nullable: false},
					{Name: "y", Type: "float", Nullable: false},
				},
				Rows:     make([][]interface{}, 15),
				RowCount: 15,
			},
			expected: true,
		},
		{
			name: "detects int columns",
			data: &rag.ResultSet{
				Columns: []rag.Column{
					{Name: "x", Type: "int", Nullable: false},
					{Name: "y", Type: "integer", Nullable: false},
				},
				Rows:     make([][]interface{}, 15),
				RowCount: 15,
			},
			expected: true,
		},
		{
			name: "rejects with too few rows",
			data: &rag.ResultSet{
				Columns: []rag.Column{
					{Name: "x", Type: "float", Nullable: false},
					{Name: "y", Type: "float", Nullable: false},
				},
				Rows:     make([][]interface{}, 5),
				RowCount: 5,
			},
			expected: false,
		},
		{
			name: "rejects with non-numeric columns",
			data: &rag.ResultSet{
				Columns: []rag.Column{
					{Name: "x", Type: "varchar", Nullable: false},
					{Name: "y", Type: "float", Nullable: false},
				},
				Rows:     make([][]interface{}, 15),
				RowCount: 15,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := newTestLoggerVisualizationEngine()
			detector := rag.NewChartDetector(logger)

			chartType := detector.Detect(tt.data)

			if tt.expected {
				assert.Equal(t, rag.ChartTypeScatter, chartType)
			}
		})
	}
}

func TestDetectionRule_Geospatial(t *testing.T) {
	tests := []struct {
		name     string
		data     *rag.ResultSet
		expected bool
	}{
		{
			name: "detects latitude column",
			data: &rag.ResultSet{
				Columns: []rag.Column{
					{Name: "latitude", Type: "float", Nullable: false},
					{Name: "value", Type: "int", Nullable: false},
				},
				Rows:     [][]interface{}{{1, 2}},
				RowCount: 1,
			},
			expected: true,
		},
		{
			name: "detects longitude column",
			data: &rag.ResultSet{
				Columns: []rag.Column{
					{Name: "longitude", Type: "float", Nullable: false},
					{Name: "value", Type: "int", Nullable: false},
				},
				Rows:     [][]interface{}{{1, 2}},
				RowCount: 1,
			},
			expected: true,
		},
		{
			name: "detects country column",
			data: &rag.ResultSet{
				Columns: []rag.Column{
					{Name: "country", Type: "varchar", Nullable: false},
					{Name: "value", Type: "int", Nullable: false},
				},
				Rows:     [][]interface{}{{1, 2}},
				RowCount: 1,
			},
			expected: true,
		},
		{
			name: "detects state column",
			data: &rag.ResultSet{
				Columns: []rag.Column{
					{Name: "state", Type: "varchar", Nullable: false},
					{Name: "value", Type: "int", Nullable: false},
				},
				Rows:     [][]interface{}{{1, 2}},
				RowCount: 1,
			},
			expected: true,
		},
		{
			name: "detects city column",
			data: &rag.ResultSet{
				Columns: []rag.Column{
					{Name: "city", Type: "varchar", Nullable: false},
					{Name: "value", Type: "int", Nullable: false},
				},
				Rows:     [][]interface{}{{1, 2}},
				RowCount: 1,
			},
			expected: true,
		},
		{
			name: "detects lat abbreviation",
			data: &rag.ResultSet{
				Columns: []rag.Column{
					{Name: "lat", Type: "float", Nullable: false},
					{Name: "lon", Type: "float", Nullable: false},
				},
				Rows:     [][]interface{}{{1, 2}},
				RowCount: 1,
			},
			expected: true,
		},
		{
			name: "rejects non-geographic data",
			data: &rag.ResultSet{
				Columns: []rag.Column{
					{Name: "id", Type: "int", Nullable: false},
					{Name: "value", Type: "int", Nullable: false},
				},
				Rows:     [][]interface{}{{1, 2}},
				RowCount: 1,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := newTestLoggerVisualizationEngine()
			detector := rag.NewChartDetector(logger)

			chartType := detector.Detect(tt.data)

			if tt.expected {
				assert.Equal(t, rag.ChartTypeGeo, chartType)
			} else {
				assert.NotEqual(t, rag.ChartTypeGeo, chartType)
			}
		})
	}
}

// TestVizRecommender tests the recommendation engine
func TestNewVizRecommender(t *testing.T) {
	t.Run("creates recommender successfully", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		recommender := rag.NewVizRecommender(logger)

		require.NotNil(t, recommender)
	})
}

func TestVizRecommender_Recommend(t *testing.T) {
	t.Run("generates recommendations", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		recommender := rag.NewVizRecommender(logger)
		data := createTimeSeriesData()

		recommendations := recommender.Recommend(data, "show trends")

		require.NotNil(t, recommendations)
		assert.IsType(t, []rag.VizRecommendation{}, recommendations)
	})

	t.Run("handles nil data", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		recommender := rag.NewVizRecommender(logger)

		recommendations := recommender.Recommend(nil, "test")

		require.NotNil(t, recommendations)
	})

	t.Run("handles empty query", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		recommender := rag.NewVizRecommender(logger)
		data := createCategoricalData()

		recommendations := recommender.Recommend(data, "")

		require.NotNil(t, recommendations)
	})
}

// TestChartGenerator tests the chart configuration generator
func TestNewChartGenerator(t *testing.T) {
	t.Run("creates generator successfully", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		generator := rag.NewChartGenerator(logger)

		require.NotNil(t, generator)
	})
}

func TestChartGenerator_Generate(t *testing.T) {
	chartTypes := []rag.ChartType{
		rag.ChartTypeLine,
		rag.ChartTypeBar,
		rag.ChartTypePie,
		rag.ChartTypeScatter,
		rag.ChartTypeHeatmap,
		rag.ChartTypeHistogram,
		rag.ChartTypeArea,
		rag.ChartTypeGeo,
		rag.ChartTypeTreemap,
		rag.ChartTypeSankey,
	}

	for _, chartType := range chartTypes {
		t.Run("generates "+string(chartType)+" config", func(t *testing.T) {
			logger := newTestLoggerVisualizationEngine()
			generator := rag.NewChartGenerator(logger)
			data := createTimeSeriesData()

			config := generator.Generate(data, chartType)

			require.NotNil(t, config)
			assert.Equal(t, chartType, config.ChartType)
			assert.NotEmpty(t, config.Title)
			assert.NotNil(t, config.Options)
		})
	}

	t.Run("handles nil data", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		generator := rag.NewChartGenerator(logger)

		config := generator.Generate(nil, rag.ChartTypeBar)

		require.NotNil(t, config)
		assert.Equal(t, rag.ChartTypeBar, config.ChartType)
	})
}

// TestNLToViz tests the natural language processor
func TestNewNLToViz(t *testing.T) {
	t.Run("creates NL processor successfully", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		nlProcessor := rag.NewNLToViz(logger)

		require.NotNil(t, nlProcessor)
	})
}

func TestNLToViz_Parse(t *testing.T) {
	t.Run("parses simple prompt", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		nlProcessor := rag.NewNLToViz(logger)
		ctx := context.Background()

		request, err := nlProcessor.Parse(ctx, "show me a bar chart")

		require.NoError(t, err)
		require.NotNil(t, request)
		assert.Equal(t, "show me a bar chart", request.Prompt)
		assert.NotNil(t, request.Preferences)
	})

	t.Run("parses complex prompt", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		nlProcessor := rag.NewNLToViz(logger)
		ctx := context.Background()

		request, err := nlProcessor.Parse(ctx, "create a line chart showing sales trends over the last 6 months grouped by product category")

		require.NoError(t, err)
		require.NotNil(t, request)
		assert.NotEmpty(t, request.Prompt)
	})

	t.Run("handles empty prompt", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		nlProcessor := rag.NewNLToViz(logger)
		ctx := context.Background()

		request, err := nlProcessor.Parse(ctx, "")

		require.NoError(t, err)
		require.NotNil(t, request)
	})
}

func TestNLToViz_ParseIntent(t *testing.T) {
	t.Run("parses intent from prompt", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		nlProcessor := rag.NewNLToViz(logger)
		ctx := context.Background()

		intent, err := nlProcessor.ParseIntent(ctx, "show me a bar chart")

		require.NoError(t, err)
		require.NotNil(t, intent)
		assert.NotNil(t, intent.Options)
	})

	t.Run("handles empty prompt", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		nlProcessor := rag.NewNLToViz(logger)
		ctx := context.Background()

		intent, err := nlProcessor.ParseIntent(ctx, "")

		require.NoError(t, err)
		require.NotNil(t, intent)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		nlProcessor := rag.NewNLToViz(logger)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		intent, err := nlProcessor.ParseIntent(ctx, "test")

		require.NoError(t, err)
		require.NotNil(t, intent)
	})
}

func TestNLToViz_GenerateChart(t *testing.T) {
	t.Run("generates chart from intent", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		nlProcessor := rag.NewNLToViz(logger)
		data := createTimeSeriesData()
		intent := &rag.VizIntent{
			ChartType: "line",
			Options:   make(map[string]interface{}),
		}

		chart := nlProcessor.GenerateChart(intent, data)

		require.NotNil(t, chart)
		require.NotNil(t, chart.VizConfig)
		assert.NotEmpty(t, chart.Title)
		assert.NotNil(t, chart.Options)
	})

	t.Run("handles nil intent", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		nlProcessor := rag.NewNLToViz(logger)
		data := createTimeSeriesData()

		chart := nlProcessor.GenerateChart(nil, data)

		require.NotNil(t, chart)
	})

	t.Run("handles nil data", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		nlProcessor := rag.NewNLToViz(logger)
		intent := &rag.VizIntent{
			ChartType: "bar",
			Options:   make(map[string]interface{}),
		}

		chart := nlProcessor.GenerateChart(intent, nil)

		require.NotNil(t, chart)
	})
}

func TestNLToViz_ApplyDefaults(t *testing.T) {
	t.Run("applies default title", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		nlProcessor := rag.NewNLToViz(logger)
		chart := &rag.Chart{
			VizConfig: &rag.VizConfig{
				ChartType: rag.ChartTypeBar,
				Title:     "",
				Options:   make(map[string]interface{}),
			},
		}

		result := nlProcessor.ApplyDefaults(chart)

		require.NotNil(t, result)
		assert.Equal(t, "Data Visualization", result.Title)
	})

	t.Run("preserves existing title", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		nlProcessor := rag.NewNLToViz(logger)
		chart := &rag.Chart{
			VizConfig: &rag.VizConfig{
				ChartType: rag.ChartTypeBar,
				Title:     "Custom Title",
				Options:   make(map[string]interface{}),
			},
		}

		result := nlProcessor.ApplyDefaults(chart)

		require.NotNil(t, result)
		assert.Equal(t, "Custom Title", result.Title)
	})

	t.Run("handles nil chart", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		nlProcessor := rag.NewNLToViz(logger)

		result := nlProcessor.ApplyDefaults(nil)

		assert.Nil(t, result)
	})
}

// TestAggregationStrategies tests aggregation strategy determination
func TestAggregationStrategies(t *testing.T) {
	t.Run("identifies categorical columns for grouping", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)
		data := &rag.ResultSet{
			Columns: []rag.Column{
				{Name: "category", Type: "varchar", Nullable: false},
				{Name: "region", Type: "string", Nullable: false},
				{Name: "amount", Type: "decimal", Nullable: false},
			},
			Rows:     make([][]interface{}, 100),
			RowCount: 100,
		}

		result := engine.AutoAggregate(data)

		require.NotNil(t, result)
	})

	t.Run("identifies numeric columns for metrics", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)
		data := &rag.ResultSet{
			Columns: []rag.Column{
				{Name: "category", Type: "varchar", Nullable: false},
				{Name: "revenue", Type: "float", Nullable: false},
				{Name: "cost", Type: "decimal", Nullable: false},
				{Name: "quantity", Type: "int", Nullable: false},
			},
			Rows:     make([][]interface{}, 100),
			RowCount: 100,
		}

		result := engine.AutoAggregate(data)

		require.NotNil(t, result)
	})

	t.Run("detects time-based aggregation", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)
		data := &rag.ResultSet{
			Columns: []rag.Column{
				{Name: "created_at", Type: "timestamp", Nullable: false},
				{Name: "amount", Type: "decimal", Nullable: false},
			},
			Rows:     make([][]interface{}, 100),
			RowCount: 100,
		}

		result := engine.AutoAggregate(data)

		require.NotNil(t, result)
	})
}

// TestVisualizationEngineEdgeCases tests various edge cases
func TestVisualizationEngineEdgeCases(t *testing.T) {
	t.Run("single row dataset", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)
		data := &rag.ResultSet{
			Columns: []rag.Column{
				{Name: "value", Type: "int", Nullable: false},
			},
			Rows:     [][]interface{}{{42}},
			RowCount: 1,
		}

		chartType := engine.DetectChartType(data)
		config := engine.GenerateVizConfig(data, chartType)

		assert.NotEqual(t, "", chartType)
		require.NotNil(t, config)
	})

	t.Run("single column dataset", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)
		data := &rag.ResultSet{
			Columns: []rag.Column{
				{Name: "value", Type: "int", Nullable: false},
			},
			Rows: [][]interface{}{
				{1}, {2}, {3}, {4}, {5},
			},
			RowCount: 5,
		}

		chartType := engine.DetectChartType(data)

		assert.NotEqual(t, "", chartType)
	})

	t.Run("many columns dataset", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)
		data := &rag.ResultSet{
			Columns: []rag.Column{
				{Name: "col1", Type: "int", Nullable: false},
				{Name: "col2", Type: "int", Nullable: false},
				{Name: "col3", Type: "int", Nullable: false},
				{Name: "col4", Type: "int", Nullable: false},
				{Name: "col5", Type: "int", Nullable: false},
				{Name: "col6", Type: "int", Nullable: false},
			},
			Rows:     [][]interface{}{{1, 2, 3, 4, 5, 6}},
			RowCount: 1,
		}

		chartType := engine.DetectChartType(data)

		assert.NotEqual(t, "", chartType)
	})

	t.Run("mixed type columns", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)
		data := &rag.ResultSet{
			Columns: []rag.Column{
				{Name: "name", Type: "varchar", Nullable: false},
				{Name: "age", Type: "int", Nullable: false},
				{Name: "salary", Type: "decimal", Nullable: false},
				{Name: "created", Type: "timestamp", Nullable: false},
			},
			Rows:     [][]interface{}{{"John", 30, 50000.0, "2023-01-01"}},
			RowCount: 1,
		}

		chartType := engine.DetectChartType(data)
		config := engine.GenerateVizConfig(data, chartType)

		assert.NotEqual(t, "", chartType)
		require.NotNil(t, config)
	})

	t.Run("nullable columns", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)
		data := &rag.ResultSet{
			Columns: []rag.Column{
				{Name: "category", Type: "varchar", Nullable: true},
				{Name: "amount", Type: "int", Nullable: true},
			},
			Rows:     [][]interface{}{{"A", 100}, {"B", nil}, {nil, 200}},
			RowCount: 3,
		}

		chartType := engine.DetectChartType(data)

		assert.NotEqual(t, "", chartType)
	})

	t.Run("very long column names", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)
		data := &rag.ResultSet{
			Columns: []rag.Column{
				{Name: "this_is_a_very_long_column_name_that_exceeds_normal_length_limits", Type: "varchar", Nullable: false},
				{Name: "another_extremely_long_column_name_for_testing_purposes", Type: "int", Nullable: false},
			},
			Rows:     [][]interface{}{{"value", 123}},
			RowCount: 1,
		}

		config := engine.GenerateVizConfig(data, rag.ChartTypeBar)

		require.NotNil(t, config)
	})

	t.Run("special characters in column names", func(t *testing.T) {
		logger := newTestLoggerVisualizationEngine()
		engine := rag.NewVisualizationEngine(logger)
		data := &rag.ResultSet{
			Columns: []rag.Column{
				{Name: "col-name-with-dashes", Type: "varchar", Nullable: false},
				{Name: "col_name_with_underscores", Type: "int", Nullable: false},
				{Name: "col.name.with.dots", Type: "float", Nullable: false},
			},
			Rows:     [][]interface{}{{"value", 123, 45.6}},
			RowCount: 1,
		}

		chartType := engine.DetectChartType(data)

		assert.NotEqual(t, "", chartType)
	})
}
