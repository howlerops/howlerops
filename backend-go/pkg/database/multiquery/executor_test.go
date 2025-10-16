package multiquery

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

type mockDatabase struct {
	lastQuery string
}

func (m *mockDatabase) Execute(ctx context.Context, query string, args ...interface{}) (*QueryResult, error) {
	m.lastQuery = query
	return &QueryResult{
		Columns:  []string{"id"},
		Rows:     [][]interface{}{{1}},
		RowCount: 1,
		Duration: time.Millisecond,
	}, nil
}

func TestExecuteSingleStripsConnectionPrefixes(t *testing.T) {
	exec := NewExecutor(&Config{}, logrus.New())

	db := &mockDatabase{}
	parsed := &ParsedQuery{
		OriginalSQL:         "SELECT * FROM @Prod-Leviosa.accounts LIMIT 10",
		RequiredConnections: []string{"Prod-Leviosa"},
		Segments: []QuerySegment{
			{ConnectionID: "Prod-Leviosa"},
		},
	}

	result, err := exec.executeSingle(context.Background(), parsed, map[string]Database{
		"Prod-Leviosa": db,
	}, nil)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "SELECT * FROM accounts LIMIT 10", db.lastQuery)
}

func TestExecuteSinglePreservesPlainQueries(t *testing.T) {
	exec := NewExecutor(&Config{}, logrus.New())

	db := &mockDatabase{}
	parsed := &ParsedQuery{
		OriginalSQL:         "SELECT * FROM accounts LIMIT 10",
		RequiredConnections: []string{},
	}

	result, err := exec.executeSingle(context.Background(), parsed, map[string]Database{
		"default": db,
	}, nil)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "SELECT * FROM accounts LIMIT 10", db.lastQuery)
}
