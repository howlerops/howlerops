package sync

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConflictResolver_ResolveConnectionConflict(t *testing.T) {
	logger := logrus.New()
	resolver := NewConflictResolver(logger)

	tests := []struct {
		name           string
		serverVersion  *ConnectionTemplate
		clientVersion  *ConnectionTemplate
		expectedWinner string
		expectedReason string
	}{
		{
			name: "Client newer - client wins",
			serverVersion: &ConnectionTemplate{
				ID:          "conn1",
				Name:        "Server Connection",
				UpdatedAt:   time.Now().Add(-1 * time.Hour),
				SyncVersion: 5,
			},
			clientVersion: &ConnectionTemplate{
				ID:          "conn1",
				Name:        "Client Connection",
				UpdatedAt:   time.Now(),
				SyncVersion: 4,
			},
			expectedWinner: "client",
			expectedReason: "client_newer",
		},
		{
			name: "Server newer - server wins",
			serverVersion: &ConnectionTemplate{
				ID:          "conn1",
				Name:        "Server Connection",
				UpdatedAt:   time.Now(),
				SyncVersion: 5,
			},
			clientVersion: &ConnectionTemplate{
				ID:          "conn1",
				Name:        "Client Connection",
				UpdatedAt:   time.Now().Add(-1 * time.Hour),
				SyncVersion: 4,
			},
			expectedWinner: "server",
			expectedReason: "server_newer",
		},
		{
			name: "Same timestamp - server wins",
			serverVersion: &ConnectionTemplate{
				ID:          "conn1",
				Name:        "Server Connection",
				UpdatedAt:   time.Date(2025, 10, 23, 10, 0, 0, 0, time.UTC),
				SyncVersion: 5,
			},
			clientVersion: &ConnectionTemplate{
				ID:          "conn1",
				Name:        "Client Connection",
				UpdatedAt:   time.Date(2025, 10, 23, 10, 0, 0, 0, time.UTC),
				SyncVersion: 4,
			},
			expectedWinner: "server",
			expectedReason: "server_newer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			winner, metadata, err := resolver.ResolveConnectionConflict(tt.serverVersion, tt.clientVersion)

			require.NoError(t, err)
			require.NotNil(t, metadata)

			if tt.expectedWinner == "client" {
				assert.Equal(t, tt.clientVersion.ID, winner.ID)
				assert.Equal(t, tt.clientVersion.Name, winner.Name)
				assert.Equal(t, "client_wins", metadata.Resolution)
			} else {
				assert.Equal(t, tt.serverVersion.ID, winner.ID)
				assert.Equal(t, tt.serverVersion.Name, winner.Name)
				assert.Equal(t, "server_wins", metadata.Resolution)
			}

			assert.Equal(t, tt.expectedReason, metadata.Reason)
			assert.Equal(t, tt.serverVersion.SyncVersion, metadata.ServerVersion)
			assert.Equal(t, tt.clientVersion.SyncVersion, metadata.ClientVersion)
		})
	}
}

func TestConflictResolver_ResolveConnectionConflict_DifferentIDs(t *testing.T) {
	logger := logrus.New()
	resolver := NewConflictResolver(logger)

	serverVersion := &ConnectionTemplate{
		ID:          "conn1",
		Name:        "Server Connection",
		UpdatedAt:   time.Now(),
		SyncVersion: 5,
	}

	clientVersion := &ConnectionTemplate{
		ID:          "conn2",
		Name:        "Client Connection",
		UpdatedAt:   time.Now(),
		SyncVersion: 4,
	}

	_, _, err := resolver.ResolveConnectionConflict(serverVersion, clientVersion)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "resource IDs do not match")
}

func TestConflictResolver_ResolveQueryConflict(t *testing.T) {
	logger := logrus.New()
	resolver := NewConflictResolver(logger)

	tests := []struct {
		name           string
		serverVersion  *SavedQuery
		clientVersion  *SavedQuery
		expectedWinner string
		expectedReason string
	}{
		{
			name: "Client newer - client wins",
			serverVersion: &SavedQuery{
				ID:          "query1",
				Name:        "Server Query",
				Query:       "SELECT * FROM users",
				UpdatedAt:   time.Now().Add(-1 * time.Hour),
				SyncVersion: 3,
			},
			clientVersion: &SavedQuery{
				ID:          "query1",
				Name:        "Client Query",
				Query:       "SELECT * FROM customers",
				UpdatedAt:   time.Now(),
				SyncVersion: 2,
			},
			expectedWinner: "client",
			expectedReason: "client_newer",
		},
		{
			name: "Server newer - server wins",
			serverVersion: &SavedQuery{
				ID:          "query1",
				Name:        "Server Query",
				Query:       "SELECT * FROM users",
				UpdatedAt:   time.Now(),
				SyncVersion: 3,
			},
			clientVersion: &SavedQuery{
				ID:          "query1",
				Name:        "Client Query",
				Query:       "SELECT * FROM customers",
				UpdatedAt:   time.Now().Add(-1 * time.Hour),
				SyncVersion: 2,
			},
			expectedWinner: "server",
			expectedReason: "server_newer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			winner, metadata, err := resolver.ResolveQueryConflict(tt.serverVersion, tt.clientVersion)

			require.NoError(t, err)
			require.NotNil(t, metadata)

			if tt.expectedWinner == "client" {
				assert.Equal(t, tt.clientVersion.ID, winner.ID)
				assert.Equal(t, tt.clientVersion.Query, winner.Query)
				assert.Equal(t, "client_wins", metadata.Resolution)
			} else {
				assert.Equal(t, tt.serverVersion.ID, winner.ID)
				assert.Equal(t, tt.serverVersion.Query, winner.Query)
				assert.Equal(t, "server_wins", metadata.Resolution)
			}

			assert.Equal(t, tt.expectedReason, metadata.Reason)
		})
	}
}

func TestConflictResolver_DetectConnectionConflict(t *testing.T) {
	logger := logrus.New()
	resolver := NewConflictResolver(logger)

	tests := []struct {
		name             string
		serverVersion    *ConnectionTemplate
		clientVersion    *ConnectionTemplate
		expectConflict   bool
	}{
		{
			name: "Server version ahead - conflict",
			serverVersion: &ConnectionTemplate{
				ID:          "conn1",
				SyncVersion: 5,
			},
			clientVersion: &ConnectionTemplate{
				ID:          "conn1",
				SyncVersion: 3,
			},
			expectConflict: true,
		},
		{
			name: "Same version - no conflict",
			serverVersion: &ConnectionTemplate{
				ID:          "conn1",
				SyncVersion: 5,
			},
			clientVersion: &ConnectionTemplate{
				ID:          "conn1",
				SyncVersion: 5,
			},
			expectConflict: false,
		},
		{
			name: "Client version ahead - no conflict (shouldn't happen)",
			serverVersion: &ConnectionTemplate{
				ID:          "conn1",
				SyncVersion: 3,
			},
			clientVersion: &ConnectionTemplate{
				ID:          "conn1",
				SyncVersion: 5,
			},
			expectConflict: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conflict := resolver.DetectConnectionConflict(tt.serverVersion, tt.clientVersion)
			assert.Equal(t, tt.expectConflict, conflict)
		})
	}
}

func TestConflictResolver_DetectQueryConflict(t *testing.T) {
	logger := logrus.New()
	resolver := NewConflictResolver(logger)

	tests := []struct {
		name             string
		serverVersion    *SavedQuery
		clientVersion    *SavedQuery
		expectConflict   bool
	}{
		{
			name: "Server version ahead - conflict",
			serverVersion: &SavedQuery{
				ID:          "query1",
				SyncVersion: 7,
			},
			clientVersion: &SavedQuery{
				ID:          "query1",
				SyncVersion: 5,
			},
			expectConflict: true,
		},
		{
			name: "Same version - no conflict",
			serverVersion: &SavedQuery{
				ID:          "query1",
				SyncVersion: 5,
			},
			clientVersion: &SavedQuery{
				ID:          "query1",
				SyncVersion: 5,
			},
			expectConflict: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conflict := resolver.DetectQueryConflict(tt.serverVersion, tt.clientVersion)
			assert.Equal(t, tt.expectConflict, conflict)
		})
	}
}

func TestConflictResolver_ShouldRejectUpdate(t *testing.T) {
	logger := logrus.New()
	resolver := NewConflictResolver(logger)

	tests := []struct {
		name              string
		serverUpdatedAt   time.Time
		clientUpdatedAt   time.Time
		serverSyncVersion int
		clientSyncVersion int
		expectReject      bool
	}{
		{
			name:              "Client behind - reject",
			serverUpdatedAt:   time.Now(),
			clientUpdatedAt:   time.Now().Add(-1 * time.Hour),
			serverSyncVersion: 5,
			clientSyncVersion: 3,
			expectReject:      true,
		},
		{
			name:              "Client same version - don't reject",
			serverUpdatedAt:   time.Now(),
			clientUpdatedAt:   time.Now(),
			serverSyncVersion: 5,
			clientSyncVersion: 5,
			expectReject:      false,
		},
		{
			name:              "Client ahead - don't reject",
			serverUpdatedAt:   time.Now().Add(-1 * time.Hour),
			clientUpdatedAt:   time.Now(),
			serverSyncVersion: 3,
			clientSyncVersion: 5,
			expectReject:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldReject, reason := resolver.ShouldRejectUpdate(
				tt.serverUpdatedAt,
				tt.clientUpdatedAt,
				tt.serverSyncVersion,
				tt.clientSyncVersion,
			)

			assert.Equal(t, tt.expectReject, shouldReject)
			if shouldReject {
				assert.NotEmpty(t, reason)
				assert.Contains(t, reason, "Update rejected")
			} else {
				assert.Empty(t, reason)
			}
		})
	}
}

func TestConflictResolver_MergeMetadata(t *testing.T) {
	logger := logrus.New()
	resolver := NewConflictResolver(logger)

	tests := []struct {
		name           string
		serverMetadata map[string]string
		clientMetadata map[string]string
		expected       map[string]string
	}{
		{
			name: "Client overrides server",
			serverMetadata: map[string]string{
				"key1": "server_value1",
				"key2": "server_value2",
			},
			clientMetadata: map[string]string{
				"key2": "client_value2",
				"key3": "client_value3",
			},
			expected: map[string]string{
				"key1": "server_value1",
				"key2": "client_value2", // Client wins
				"key3": "client_value3",
			},
		},
		{
			name:           "Empty server metadata",
			serverMetadata: map[string]string{},
			clientMetadata: map[string]string{
				"key1": "client_value1",
			},
			expected: map[string]string{
				"key1": "client_value1",
			},
		},
		{
			name: "Empty client metadata",
			serverMetadata: map[string]string{
				"key1": "server_value1",
			},
			clientMetadata: map[string]string{},
			expected: map[string]string{
				"key1": "server_value1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged := resolver.MergeMetadata(tt.serverMetadata, tt.clientMetadata)
			assert.Equal(t, tt.expected, merged)
		})
	}
}

func TestConflictResolver_CreateConflictInfo(t *testing.T) {
	logger := logrus.New()
	resolver := NewConflictResolver(logger)

	metadata := &ConflictMetadata{
		Resolution:    "client_wins",
		Reason:        "client_newer",
		ServerVersion: 5,
		ClientVersion: 4,
		ConflictedAt:  time.Now(),
	}

	info := resolver.CreateConflictInfo("connection", "conn1", metadata)

	assert.Equal(t, "connection", info.ResourceType)
	assert.Equal(t, "conn1", info.ResourceID)
	assert.Equal(t, "client_wins", info.Metadata.Resolution)
	assert.Equal(t, "client_newer", info.Metadata.Reason)
	assert.Equal(t, 5, info.Metadata.ServerVersion)
	assert.Equal(t, 4, info.Metadata.ClientVersion)
}
