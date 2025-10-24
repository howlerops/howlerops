package middleware

import (
	"testing"
	"time"
)

func TestNewInvitationRateLimiter(t *testing.T) {
	tests := []struct {
		name      string
		userLimit int
		orgLimit  int
		wantUser  int
		wantOrg   int
	}{
		{
			name:      "default limits",
			userLimit: 20,
			orgLimit:  5,
			wantUser:  20,
			wantOrg:   5,
		},
		{
			name:      "zero limits use defaults",
			userLimit: 0,
			orgLimit:  0,
			wantUser:  20,
			wantOrg:   5,
		},
		{
			name:      "custom limits",
			userLimit: 50,
			orgLimit:  10,
			wantUser:  50,
			wantOrg:   10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewInvitationRateLimiter(tt.userLimit, tt.orgLimit)
			if rl.userLimit != tt.wantUser {
				t.Errorf("user limit = %d, want %d", rl.userLimit, tt.wantUser)
			}
			if rl.orgLimit != tt.wantOrg {
				t.Errorf("org limit = %d, want %d", rl.orgLimit, tt.wantOrg)
			}
		})
	}
}

func TestCheckUserLimit(t *testing.T) {
	rl := NewInvitationRateLimiter(3, 5) // Low limit for testing
	userID := "user123"

	// First 3 requests should be allowed (burst)
	for i := 0; i < 3; i++ {
		if !rl.CheckUserLimit(userID) {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 4th request should be denied (exceeded burst)
	if rl.CheckUserLimit(userID) {
		t.Error("4th request should be denied")
	}

	// Different user should have separate limit
	otherUserID := "user456"
	if !rl.CheckUserLimit(otherUserID) {
		t.Error("different user should have separate limit")
	}
}

func TestCheckOrgLimit(t *testing.T) {
	rl := NewInvitationRateLimiter(20, 3) // Low org limit for testing
	orgID := "org123"

	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		if !rl.CheckOrgLimit(orgID) {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 4th request should be denied
	if rl.CheckOrgLimit(orgID) {
		t.Error("4th request should be denied")
	}

	// Different org should have separate limit
	otherOrgID := "org456"
	if !rl.CheckOrgLimit(otherOrgID) {
		t.Error("different org should have separate limit")
	}
}

func TestCheckBothLimits(t *testing.T) {
	rl := NewInvitationRateLimiter(2, 3)
	userID := "user123"
	orgID := "org123"

	tests := []struct {
		name         string
		setup        func()
		wantAllowed  bool
		wantReason   string
		reasonPrefix string
	}{
		{
			name: "both limits allow",
			setup: func() {
				rl.ResetUserLimit(userID)
				rl.ResetOrgLimit(orgID)
			},
			wantAllowed: true,
			wantReason:  "",
		},
		{
			name: "user limit exceeded",
			setup: func() {
				rl.ResetUserLimit(userID)
				rl.ResetOrgLimit(orgID)
				// Exhaust user limit
				for i := 0; i < 2; i++ {
					rl.CheckUserLimit(userID)
				}
			},
			wantAllowed:  false,
			reasonPrefix: "user rate limit",
		},
		{
			name: "org limit exceeded",
			setup: func() {
				rl.ResetUserLimit(userID)
				rl.ResetOrgLimit(orgID)
				// Exhaust org limit
				for i := 0; i < 3; i++ {
					rl.CheckOrgLimit(orgID)
				}
			},
			wantAllowed:  false,
			reasonPrefix: "organization rate limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			allowed, reason := rl.CheckBothLimits(userID, orgID)
			if allowed != tt.wantAllowed {
				t.Errorf("allowed = %v, want %v", allowed, tt.wantAllowed)
			}
			if tt.reasonPrefix != "" && reason[:len(tt.reasonPrefix)] != tt.reasonPrefix {
				t.Errorf("reason = %s, want prefix %s", reason, tt.reasonPrefix)
			}
		})
	}
}

func TestGetRemainingInvitations(t *testing.T) {
	rl := NewInvitationRateLimiter(5, 3)
	userID := "user123"
	orgID := "org123"

	// Initially, should have full quota
	userRemaining := rl.GetUserRemainingInvitations(userID)
	if userRemaining != 5 {
		t.Errorf("user remaining = %d, want 5", userRemaining)
	}

	orgRemaining := rl.GetOrgRemainingInvitations(orgID)
	if orgRemaining != 3 {
		t.Errorf("org remaining = %d, want 3", orgRemaining)
	}

	// After using one, should have one less
	rl.CheckUserLimit(userID)
	userRemaining = rl.GetUserRemainingInvitations(userID)
	if userRemaining != 4 {
		t.Errorf("user remaining after one request = %d, want 4", userRemaining)
	}
}

func TestResetLimits(t *testing.T) {
	rl := NewInvitationRateLimiter(2, 2)
	userID := "user123"
	orgID := "org123"

	// Exhaust limits
	rl.CheckUserLimit(userID)
	rl.CheckUserLimit(userID)
	rl.CheckOrgLimit(orgID)
	rl.CheckOrgLimit(orgID)

	// Both should be exhausted
	if rl.CheckUserLimit(userID) {
		t.Error("user limit should be exhausted")
	}
	if rl.CheckOrgLimit(orgID) {
		t.Error("org limit should be exhausted")
	}

	// Reset and try again
	rl.ResetUserLimit(userID)
	rl.ResetOrgLimit(orgID)

	if !rl.CheckUserLimit(userID) {
		t.Error("user limit should be reset")
	}
	if !rl.CheckOrgLimit(orgID) {
		t.Error("org limit should be reset")
	}
}

func TestGetRetryAfter(t *testing.T) {
	rl := NewInvitationRateLimiter(2, 2)
	userID := "user123"

	// Exhaust limit
	rl.CheckUserLimit(userID)
	rl.CheckUserLimit(userID)

	// Should have a retry-after time
	retryAfter := rl.GetUserRetryAfter(userID)
	if retryAfter <= 0 {
		t.Error("retry-after should be positive when limit is exceeded")
	}

	// For a new user, should be zero
	newUserRetryAfter := rl.GetUserRetryAfter("newuser")
	if newUserRetryAfter != 0 {
		t.Errorf("retry-after for new user = %v, want 0", newUserRetryAfter)
	}
}

func TestFormatRetryAfter(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "1 second",
			duration: 1 * time.Second,
			want:     "1",
		},
		{
			name:     "30 seconds",
			duration: 30 * time.Second,
			want:     "30",
		},
		{
			name:     "less than 1 second rounds to 1",
			duration: 500 * time.Millisecond,
			want:     "1",
		},
		{
			name:     "1 minute",
			duration: 60 * time.Second,
			want:     "60",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatRetryAfter(tt.duration)
			if got != tt.want {
				t.Errorf("FormatRetryAfter(%v) = %s, want %s", tt.duration, got, tt.want)
			}
		})
	}
}

func TestGetStats(t *testing.T) {
	rl := NewInvitationRateLimiter(20, 5)

	// Initially should have no limiters
	stats := rl.GetStats()
	if stats["user_limiters_count"] != 0 {
		t.Errorf("initial user limiters count = %v, want 0", stats["user_limiters_count"])
	}
	if stats["org_limiters_count"] != 0 {
		t.Errorf("initial org limiters count = %v, want 0", stats["org_limiters_count"])
	}

	// After checking limits, should have limiters
	rl.CheckUserLimit("user1")
	rl.CheckUserLimit("user2")
	rl.CheckOrgLimit("org1")

	stats = rl.GetStats()
	if stats["user_limiters_count"] != 2 {
		t.Errorf("user limiters count after checks = %v, want 2", stats["user_limiters_count"])
	}
	if stats["org_limiters_count"] != 1 {
		t.Errorf("org limiters count after checks = %v, want 1", stats["org_limiters_count"])
	}
	if stats["user_limit"] != 20 {
		t.Errorf("user limit = %v, want 20", stats["user_limit"])
	}
	if stats["org_limit"] != 5 {
		t.Errorf("org limit = %v, want 5", stats["org_limit"])
	}
}

func TestConcurrentAccess(t *testing.T) {
	rl := NewInvitationRateLimiter(100, 100)
	userID := "user123"
	orgID := "org123"

	// Test concurrent access to ensure no race conditions
	done := make(chan bool, 20)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				rl.CheckUserLimit(userID)
				rl.GetUserRemainingInvitations(userID)
			}
			done <- true
		}()
		go func() {
			for j := 0; j < 10; j++ {
				rl.CheckOrgLimit(orgID)
				rl.GetOrgRemainingInvitations(orgID)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 20; i++ {
		<-done
	}

	// If we get here without deadlock or panic, the test passes
	stats := rl.GetStats()
	if stats["user_limiters_count"] == nil || stats["org_limiters_count"] == nil {
		t.Error("stats should be accessible after concurrent access")
	}
}
