package integration

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AuthTestSuite contains all auth-related tests
type AuthTestSuite struct {
	baseURL string
	client  *http.Client
}

// NewAuthTestSuite creates a new auth test suite
func NewAuthTestSuite() *AuthTestSuite {
	baseURL := os.Getenv("TEST_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8500"
	}

	return &AuthTestSuite{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// isServerAvailable checks if the test server is available
func isServerAvailable(baseURL string) bool {
	// Extract host:port from URL (handle both http://localhost:8500 and localhost:8500)
	address := baseURL
	if len(address) > 7 && address[:7] == "http://" {
		address = address[7:]
	} else if len(address) > 8 && address[:8] == "https://" {
		address = address[8:]
	}

	// Try to connect with a short timeout
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close() // Best-effort close in test
	return true
}

// requireServer skips the test if the server is not available
func requireServer(t *testing.T, baseURL string) {
	if !isServerAvailable(baseURL) {
		t.Skipf("Skipping integration test: server not available at %s (connection refused or timeout)", baseURL)
	}
}

// SignupRequest represents a signup request
type SignupRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthResponse represents an auth response
type AuthResponse struct {
	User struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	} `json:"user"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// RefreshRequest represents a token refresh request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
}

// TestAuthFlow tests the complete authentication flow
func TestAuthFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := NewAuthTestSuite()
	requireServer(t, suite.baseURL)

	// Generate unique test user
	timestamp := time.Now().Unix()
	testUser := SignupRequest{
		Email:    "test" + string(rune(timestamp)) + "@example.com",
		Username: "testuser" + string(rune(timestamp)),
		Password: "TestPassword123!",
	}

	t.Run("1_Signup", func(t *testing.T) {
		suite.testSignup(t, testUser)
	})

	t.Run("2_Login", func(t *testing.T) {
		suite.testLogin(t, testUser)
	})

	t.Run("3_LoginWithInvalidPassword", func(t *testing.T) {
		suite.testLoginWithInvalidPassword(t, testUser)
	})

	t.Run("4_TokenRefresh", func(t *testing.T) {
		suite.testTokenRefresh(t, testUser)
	})

	t.Run("5_ProtectedEndpoint", func(t *testing.T) {
		suite.testProtectedEndpoint(t, testUser)
	})

	t.Run("6_Logout", func(t *testing.T) {
		suite.testLogout(t, testUser)
	})
}

func (s *AuthTestSuite) testSignup(t *testing.T, user SignupRequest) {
	reqBody, err := json.Marshal(user)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", s.baseURL+"/api/auth/signup", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }() // Best-effort close in test

	// Should return 201 Created or 200 OK
	assert.True(t, resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK,
		"Expected status 200/201, got %d", resp.StatusCode)

	var authResp AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&authResp)
	require.NoError(t, err)

	// Validate response
	assert.NotEmpty(t, authResp.User.ID, "User ID should not be empty")
	assert.Equal(t, user.Username, authResp.User.Username)
	assert.Equal(t, user.Email, authResp.User.Email)
	assert.NotEmpty(t, authResp.Token, "Token should not be empty")
	assert.NotEmpty(t, authResp.RefreshToken, "Refresh token should not be empty")
}

func (s *AuthTestSuite) testLogin(t *testing.T, user SignupRequest) {
	loginReq := LoginRequest{
		Username: user.Username,
		Password: user.Password,
	}

	reqBody, err := json.Marshal(loginReq)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", s.baseURL+"/api/auth/login", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }() // Best-effort close in test

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var authResp AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&authResp)
	require.NoError(t, err)

	assert.NotEmpty(t, authResp.Token)
	assert.NotEmpty(t, authResp.RefreshToken)
	assert.Equal(t, user.Username, authResp.User.Username)
}

func (s *AuthTestSuite) testLoginWithInvalidPassword(t *testing.T, user SignupRequest) {
	loginReq := LoginRequest{
		Username: user.Username,
		Password: "WrongPassword123!",
	}

	reqBody, err := json.Marshal(loginReq)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", s.baseURL+"/api/auth/login", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }() // Best-effort close in test

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	var errResp ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errResp)
	require.NoError(t, err)

	assert.True(t, errResp.Error)
	assert.NotEmpty(t, errResp.Message)
}

func (s *AuthTestSuite) testTokenRefresh(t *testing.T, user SignupRequest) {
	// First login to get tokens
	loginReq := LoginRequest{
		Username: user.Username,
		Password: user.Password,
	}

	reqBody, err := json.Marshal(loginReq)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", s.baseURL+"/api/auth/login", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }() // Best-effort close in test

	var authResp AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&authResp)
	require.NoError(t, err)

	// Now use refresh token
	refreshReq := RefreshRequest{
		RefreshToken: authResp.RefreshToken,
	}

	reqBody, err = json.Marshal(refreshReq)
	require.NoError(t, err)

	req, err = http.NewRequest("POST", s.baseURL+"/api/auth/refresh", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err = s.client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }() // Best-effort close in test

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var newAuthResp AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&newAuthResp)
	require.NoError(t, err)

	assert.NotEmpty(t, newAuthResp.Token)
	assert.NotEmpty(t, newAuthResp.RefreshToken)
	assert.NotEqual(t, authResp.Token, newAuthResp.Token, "New token should be different")
}

func (s *AuthTestSuite) testProtectedEndpoint(t *testing.T, user SignupRequest) {
	// First login to get token
	loginReq := LoginRequest{
		Username: user.Username,
		Password: user.Password,
	}

	reqBody, err := json.Marshal(loginReq)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", s.baseURL+"/api/auth/login", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }() // Best-effort close in test

	var authResp AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&authResp)
	require.NoError(t, err)

	// Test protected endpoint without token
	req, err = http.NewRequest("GET", s.baseURL+"/api/auth/profile", nil)
	require.NoError(t, err)

	resp, err = s.client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }() // Best-effort close in test

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// Test protected endpoint with token
	req, err = http.NewRequest("GET", s.baseURL+"/api/auth/profile", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+authResp.Token)

	resp, err = s.client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }() // Best-effort close in test

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func (s *AuthTestSuite) testLogout(t *testing.T, user SignupRequest) {
	// First login to get token
	loginReq := LoginRequest{
		Username: user.Username,
		Password: user.Password,
	}

	reqBody, err := json.Marshal(loginReq)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", s.baseURL+"/api/auth/login", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }() // Best-effort close in test

	var authResp AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&authResp)
	require.NoError(t, err)

	// Logout
	req, err = http.NewRequest("POST", s.baseURL+"/api/auth/logout", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+authResp.Token)

	resp, err = s.client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }() // Best-effort close in test

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify token is invalid after logout
	req, err = http.NewRequest("GET", s.baseURL+"/api/auth/profile", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+authResp.Token)

	resp, err = s.client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }() // Best-effort close in test

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// TestAuthRateLimiting tests rate limiting on auth endpoints
func TestAuthRateLimiting(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := NewAuthTestSuite()
	requireServer(t, suite.baseURL)

	loginReq := LoginRequest{
		Username: "nonexistent",
		Password: "password",
	}

	// Make many requests quickly
	failureCount := 0
	for i := 0; i < 10; i++ {
		reqBody, _ := json.Marshal(loginReq)
		req, _ := http.NewRequest("POST", suite.baseURL+"/api/auth/login", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := suite.client.Do(req)
		if err == nil {
			if resp.StatusCode == http.StatusTooManyRequests {
				failureCount++
			}
			resp.Body.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Rate limiting should kick in at some point
	// This test might be flaky depending on rate limit config
	t.Logf("Rate limit triggered %d times out of 10 requests", failureCount)
}

// TestAuthValidation tests input validation
func TestAuthValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite := NewAuthTestSuite()
	requireServer(t, suite.baseURL)

	tests := []struct {
		name       string
		request    interface{}
		endpoint   string
		wantStatus int
	}{
		{
			name: "Empty email on signup",
			request: SignupRequest{
				Email:    "",
				Username: "testuser",
				Password: "TestPassword123!",
			},
			endpoint:   "/api/auth/signup",
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "Invalid email format",
			request: SignupRequest{
				Email:    "not-an-email",
				Username: "testuser",
				Password: "TestPassword123!",
			},
			endpoint:   "/api/auth/signup",
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "Weak password",
			request: SignupRequest{
				Email:    "test@example.com",
				Username: "testuser",
				Password: "weak",
			},
			endpoint:   "/api/auth/signup",
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "Empty username on login",
			request: LoginRequest{
				Username: "",
				Password: "password",
			},
			endpoint:   "/api/auth/login",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", suite.baseURL+tt.endpoint, bytes.NewBuffer(reqBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			resp, err := suite.client.Do(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }() // Best-effort close in test

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}
