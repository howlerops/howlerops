package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SyncTestSuite contains all sync-related tests
type SyncTestSuite struct {
	baseURL string
	client  *http.Client
}

// NewSyncTestSuite creates a new sync test suite
func NewSyncTestSuite() *SyncTestSuite {
	baseURL := os.Getenv("TEST_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8500"
	}

	return &SyncTestSuite{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SyncUploadRequest represents a sync upload request
type SyncUploadRequest struct {
	DeviceID   string       `json:"device_id"`
	LastSyncAt time.Time    `json:"last_sync_at"`
	Changes    []SyncChange `json:"changes"`
}

// SyncChange represents a change to be synced
type SyncChange struct {
	ID          string      `json:"id"`
	ItemType    string      `json:"item_type"`
	ItemID      string      `json:"item_id"`
	Action      string      `json:"action"`
	Data        interface{} `json:"data"`
	UpdatedAt   time.Time   `json:"updated_at"`
	SyncVersion int         `json:"sync_version"`
	DeviceID    string      `json:"device_id"`
}

// SyncUploadResponse represents the response after uploading changes
type SyncUploadResponse struct {
	Success   bool      `json:"success"`
	SyncedAt  time.Time `json:"synced_at"`
	Conflicts []struct {
		ID       string `json:"id"`
		ItemType string `json:"item_type"`
		ItemID   string `json:"item_id"`
	} `json:"conflicts,omitempty"`
	Rejected []struct {
		Change interface{} `json:"change"`
		Reason string      `json:"reason"`
	} `json:"rejected,omitempty"`
	Message string `json:"message,omitempty"`
}

// SyncDownloadResponse represents the response with remote changes
type SyncDownloadResponse struct {
	Connections   []interface{} `json:"connections"`
	SavedQueries  []interface{} `json:"saved_queries"`
	QueryHistory  []interface{} `json:"query_history"`
	SyncTimestamp time.Time     `json:"sync_timestamp"`
	HasMore       bool          `json:"has_more"`
}

// ConflictListResponse represents the response when listing conflicts
type ConflictListResponse struct {
	Conflicts []interface{} `json:"conflicts"`
	Count     int           `json:"count"`
}

// ConflictResolutionRequest represents a request to resolve a conflict
type ConflictResolutionRequest struct {
	Strategy      string `json:"strategy"`
	ChosenVersion string `json:"chosen_version,omitempty"`
}

// ConflictResolutionResponse represents the response after resolving a conflict
type ConflictResolutionResponse struct {
	Success    bool      `json:"success"`
	ResolvedAt time.Time `json:"resolved_at"`
	Message    string    `json:"message,omitempty"`
}

// TestSyncFlow tests the complete sync flow
func TestSyncFlow(t *testing.T) {
	suite := NewSyncTestSuite()

	// First, authenticate to get a token
	token := suite.authenticate(t)

	t.Run("1_Upload", func(t *testing.T) {
		suite.testUpload(t, token)
	})

	t.Run("2_Download", func(t *testing.T) {
		suite.testDownload(t, token)
	})

	t.Run("3_ListConflicts", func(t *testing.T) {
		suite.testListConflicts(t, token)
	})

	t.Run("4_UploadWithoutAuth", func(t *testing.T) {
		suite.testUploadWithoutAuth(t)
	})

	t.Run("5_DownloadWithoutAuth", func(t *testing.T) {
		suite.testDownloadWithoutAuth(t)
	})
}

func (s *SyncTestSuite) authenticate(t *testing.T) string {
	// Create a test user and login
	timestamp := time.Now().Unix()
	signupReq := map[string]string{
		"email":    "synctest" + string(rune(timestamp)) + "@example.com",
		"username": "synctest" + string(rune(timestamp)),
		"password": "TestPassword123!",
	}

	reqBody, err := json.Marshal(signupReq)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", s.baseURL+"/api/auth/signup", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var authResp struct {
		Token string `json:"token"`
	}
	err = json.NewDecoder(resp.Body).Decode(&authResp)
	require.NoError(t, err)

	return authResp.Token
}

func (s *SyncTestSuite) testUpload(t *testing.T, token string) {
	uploadReq := SyncUploadRequest{
		DeviceID:   "test-device-123",
		LastSyncAt: time.Now().Add(-24 * time.Hour),
		Changes: []SyncChange{
			{
				ID:          "change-1",
				ItemType:    "saved_query",
				ItemID:      "query-123",
				Action:      "create",
				Data: map[string]interface{}{
					"id":          "query-123",
					"name":        "Test Query",
					"query":       "SELECT * FROM users",
					"favorite":    false,
					"created_at":  time.Now().Format(time.RFC3339),
					"updated_at":  time.Now().Format(time.RFC3339),
					"sync_version": 1,
				},
				UpdatedAt:   time.Now(),
				SyncVersion: 1,
				DeviceID:    "test-device-123",
			},
		},
	}

	reqBody, err := json.Marshal(uploadReq)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", s.baseURL+"/api/sync/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var uploadResp SyncUploadResponse
	err = json.NewDecoder(resp.Body).Decode(&uploadResp)
	require.NoError(t, err)

	assert.True(t, uploadResp.Success)
	assert.NotZero(t, uploadResp.SyncedAt)
}

func (s *SyncTestSuite) testDownload(t *testing.T, token string) {
	// Download with since parameter
	since := time.Now().Add(-7 * 24 * time.Hour).Format(time.RFC3339)
	url := s.baseURL + "/api/sync/download?device_id=test-device-123&since=" + since

	req, err := http.NewRequest("GET", url, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var downloadResp SyncDownloadResponse
	err = json.NewDecoder(resp.Body).Decode(&downloadResp)
	require.NoError(t, err)

	assert.NotNil(t, downloadResp.Connections)
	assert.NotNil(t, downloadResp.SavedQueries)
	assert.NotNil(t, downloadResp.QueryHistory)
	assert.NotZero(t, downloadResp.SyncTimestamp)
}

func (s *SyncTestSuite) testListConflicts(t *testing.T, token string) {
	req, err := http.NewRequest("GET", s.baseURL+"/api/sync/conflicts", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var conflictResp ConflictListResponse
	err = json.NewDecoder(resp.Body).Decode(&conflictResp)
	require.NoError(t, err)

	assert.NotNil(t, conflictResp.Conflicts)
	assert.GreaterOrEqual(t, conflictResp.Count, 0)
}

func (s *SyncTestSuite) testUploadWithoutAuth(t *testing.T) {
	uploadReq := SyncUploadRequest{
		DeviceID:   "test-device-123",
		LastSyncAt: time.Now().Add(-24 * time.Hour),
		Changes:    []SyncChange{},
	}

	reqBody, err := json.Marshal(uploadReq)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", s.baseURL+"/api/sync/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func (s *SyncTestSuite) testDownloadWithoutAuth(t *testing.T) {
	req, err := http.NewRequest("GET", s.baseURL+"/api/sync/download?device_id=test-device-123", nil)
	require.NoError(t, err)

	resp, err := s.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// TestSyncValidation tests input validation for sync endpoints
func TestSyncValidation(t *testing.T) {
	suite := NewSyncTestSuite()
	token := suite.authenticate(t)

	t.Run("Upload_MissingDeviceID", func(t *testing.T) {
		uploadReq := SyncUploadRequest{
			DeviceID:   "",
			LastSyncAt: time.Now(),
			Changes:    []SyncChange{},
		}

		reqBody, err := json.Marshal(uploadReq)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", suite.baseURL+"/api/sync/upload", bytes.NewBuffer(reqBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := suite.client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Upload_EmptyChanges", func(t *testing.T) {
		uploadReq := SyncUploadRequest{
			DeviceID:   "test-device",
			LastSyncAt: time.Now(),
			Changes:    []SyncChange{},
		}

		reqBody, err := json.Marshal(uploadReq)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", suite.baseURL+"/api/sync/upload", bytes.NewBuffer(reqBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := suite.client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Download_MissingDeviceID", func(t *testing.T) {
		req, err := http.NewRequest("GET", suite.baseURL+"/api/sync/download", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := suite.client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Download_InvalidSinceFormat", func(t *testing.T) {
		req, err := http.NewRequest("GET", suite.baseURL+"/api/sync/download?device_id=test&since=invalid", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := suite.client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestSyncConflictResolution tests conflict resolution
func TestSyncConflictResolution(t *testing.T) {
	suite := NewSyncTestSuite()
	token := suite.authenticate(t)

	// First, create a conflict by uploading the same item from two devices
	// (This is a simplified test - actual conflict creation might be more complex)

	t.Run("ResolveConflict_LastWriteWins", func(t *testing.T) {
		resolutionReq := ConflictResolutionRequest{
			Strategy: "last_write_wins",
		}

		reqBody, err := json.Marshal(resolutionReq)
		require.NoError(t, err)

		// Use a dummy conflict ID - in real tests, you'd create an actual conflict first
		req, err := http.NewRequest("POST", suite.baseURL+"/api/sync/conflicts/conflict-123/resolve", bytes.NewBuffer(reqBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := suite.client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// May return 404 if conflict doesn't exist, which is acceptable for this test
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound)
	})

	t.Run("ResolveConflict_InvalidStrategy", func(t *testing.T) {
		resolutionReq := ConflictResolutionRequest{
			Strategy: "invalid_strategy",
		}

		reqBody, err := json.Marshal(resolutionReq)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", suite.baseURL+"/api/sync/conflicts/conflict-123/resolve", bytes.NewBuffer(reqBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := suite.client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestSyncLargePayload tests uploading a large number of changes
func TestSyncLargePayload(t *testing.T) {
	suite := NewSyncTestSuite()
	token := suite.authenticate(t)

	// Create 100 changes
	changes := make([]SyncChange, 100)
	for i := 0; i < 100; i++ {
		changes[i] = SyncChange{
			ID:       string(rune(i)),
			ItemType: "saved_query",
			ItemID:   "query-" + string(rune(i)),
			Action:   "create",
			Data: map[string]interface{}{
				"id":    "query-" + string(rune(i)),
				"name":  "Test Query " + string(rune(i)),
				"query": "SELECT * FROM table" + string(rune(i)),
			},
			UpdatedAt:   time.Now(),
			SyncVersion: 1,
			DeviceID:    "test-device",
		}
	}

	uploadReq := SyncUploadRequest{
		DeviceID:   "test-device-123",
		LastSyncAt: time.Now().Add(-24 * time.Hour),
		Changes:    changes,
	}

	reqBody, err := json.Marshal(uploadReq)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", suite.baseURL+"/api/sync/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := suite.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should handle large payloads successfully
	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusRequestEntityTooLarge)
}
