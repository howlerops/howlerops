package organization

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// BenchmarkPermissionCheck benchmarks the permission checking logic
func BenchmarkPermissionCheck(b *testing.B) {
	// Test different scenarios
	scenarios := []struct {
		name string
		role OrganizationRole
		perm Permission
	}{
		{"Owner-ViewOrg", RoleOwner, PermViewOrganization},
		{"Owner-DeleteOrg", RoleOwner, PermDeleteOrganization},
		{"Admin-InviteMembers", RoleAdmin, PermInviteMembers},
		{"Admin-DeleteOrg", RoleAdmin, PermDeleteOrganization}, // Should fail
		{"Member-ViewOrg", RoleMember, PermViewOrganization},
		{"Member-InviteMembers", RoleMember, PermInviteMembers}, // Should fail
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = HasPermission(scenario.role, scenario.perm)
			}
		})
	}
}

// BenchmarkPermissionCheckWithDB benchmarks permission checks including database queries
func BenchmarkPermissionCheckWithDB(b *testing.B) {
	// Setup service with mock repository
	repo := newMockRepository()
	logger := logrus.New()
	service := NewService(repo, logger)

	// Setup test data
	orgID := "test-org"
	ownerID := "owner-id"
	memberID := "member-id"

	// Create organization with members
	org := &Organization{
		ID:         orgID,
		Name:       "Test Org",
		OwnerID:    ownerID,
		MaxMembers: 100,
	}
	repo.organizations = append(repo.organizations, org)

	repo.members = append(repo.members,
		&OrganizationMember{
			OrganizationID: orgID,
			UserID:         ownerID,
			Role:           RoleOwner,
		},
		&OrganizationMember{
			OrganizationID: orgID,
			UserID:         memberID,
			Role:           RoleMember,
		},
	)

	ctx := context.Background()

	b.Run("GetOrganization-Owner", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = service.GetOrganization(ctx, orgID, ownerID)
		}
	})

	b.Run("GetOrganization-Member", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = service.GetOrganization(ctx, orgID, memberID)
		}
	})

	b.Run("GetOrganization-NonMember", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = service.GetOrganization(ctx, orgID, "non-member")
		}
	})
}

// BenchmarkConcurrentPermissionChecks benchmarks concurrent permission checks
func BenchmarkConcurrentPermissionChecks(b *testing.B) {
	// Setup service
	repo := newMockRepository()
	logger := logrus.New()
	service := NewService(repo, logger)

	// Create multiple organizations and members
	numOrgs := 10
	numMembersPerOrg := 50

	for orgIdx := 0; orgIdx < numOrgs; orgIdx++ {
		orgID := fmt.Sprintf("org-%d", orgIdx)
		ownerID := fmt.Sprintf("owner-%d", orgIdx)

		org := &Organization{
			ID:         orgID,
			Name:       fmt.Sprintf("Org %d", orgIdx),
			OwnerID:    ownerID,
			MaxMembers: 100,
		}
		repo.organizations = append(repo.organizations, org)

		// Add owner
		repo.members = append(repo.members, &OrganizationMember{
			OrganizationID: orgID,
			UserID:         ownerID,
			Role:           RoleOwner,
		})

		// Add members
		for memberIdx := 0; memberIdx < numMembersPerOrg; memberIdx++ {
			repo.members = append(repo.members, &OrganizationMember{
				OrganizationID: orgID,
				UserID:         fmt.Sprintf("member-%d-%d", orgIdx, memberIdx),
				Role:           RoleMember,
			})
		}
	}

	ctx := context.Background()

	// Benchmark different concurrency levels
	concurrencyLevels := []int{1, 10, 50, 100}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency-%d", concurrency), func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				wg.Add(concurrency)

				for j := 0; j < concurrency; j++ {
					go func(idx int) {
						defer wg.Done()

						// Random org and user
						orgID := fmt.Sprintf("org-%d", idx%numOrgs)
						userID := fmt.Sprintf("member-%d-%d", idx%numOrgs, idx%numMembersPerOrg)

						_, _ = service.GetMembers(ctx, orgID, userID)
					}(j)
				}

				wg.Wait()
			}
		})
	}
}

// BenchmarkRateLimiting benchmarks rate limiting overhead
func BenchmarkRateLimiting(b *testing.B) {
	// Setup service with rate limiter
	repo := newMockRepository()
	logger := logrus.New()
	service := NewService(repo, logger)

	rateLimiter := &benchmarkRateLimiter{
		limits: make(map[string]*limitInfo),
	}
	service.SetRateLimiter(rateLimiter)

	// Setup org
	orgID := "test-org"
	ownerID := "owner-id"
	repo.organizations = append(repo.organizations, &Organization{
		ID:         orgID,
		Name:       "Test Org",
		OwnerID:    ownerID,
		MaxMembers: 1000,
	})
	repo.members = append(repo.members, &OrganizationMember{
		OrganizationID: orgID,
		UserID:         ownerID,
		Role:           RoleOwner,
	})

	ctx := context.Background()

	b.Run("WithRateLimiting", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// This will check rate limits
			_, _ = service.CreateInvitation(ctx, orgID, ownerID, &CreateInvitationInput{
				Email: fmt.Sprintf("test%d@example.com", i),
				Role:  RoleMember,
			})
		}
	})

	// Disable rate limiting for comparison
	service.SetRateLimiter(nil)

	b.Run("WithoutRateLimiting", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = service.CreateInvitation(ctx, orgID, ownerID, &CreateInvitationInput{
				Email: fmt.Sprintf("test%d@example.com", i),
				Role:  RoleMember,
			})
		}
	})
}

// BenchmarkAuditLogging benchmarks audit log creation overhead
func BenchmarkAuditLogging(b *testing.B) {
	repo := newMockRepository()
	logger := logrus.New()
	service := NewService(repo, logger)

	ctx := context.Background()
	orgID := "test-org"
	userID := "test-user"

	b.Run("CreateAuditLog", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = service.CreateAuditLog(ctx, &AuditLog{
				OrganizationID: &orgID,
				UserID:         userID,
				Action:         "test.action",
				ResourceType:   "test",
				Details: map[string]interface{}{
					"iteration": i,
					"benchmark": true,
				},
			})
		}
	})

	b.Run("GetAuditLogs", func(b *testing.B) {
		// Pre-populate some audit logs
		for i := 0; i < 1000; i++ {
			repo.auditLogs = append(repo.auditLogs, &AuditLog{
				ID:             fmt.Sprintf("audit-%d", i),
				OrganizationID: &orgID,
				UserID:         userID,
				Action:         fmt.Sprintf("action.%d", i),
				CreatedAt:      time.Now(),
			})
		}

		// Add permission for user
		repo.members = append(repo.members, &OrganizationMember{
			OrganizationID: orgID,
			UserID:         userID,
			Role:           RoleOwner,
		})

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = service.GetAuditLogs(ctx, orgID, userID, 100, 0)
		}
	})
}

// BenchmarkTokenGeneration benchmarks secure token generation
func BenchmarkTokenGeneration(b *testing.B) {
	b.Run("GenerateSecureToken", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = generateSecureToken()
		}
	})
}

// BenchmarkInputValidation benchmarks input validation
func BenchmarkInputValidation(b *testing.B) {
	testCases := []struct {
		name  string
		input string
	}{
		{"ShortName", "Org"},
		{"NormalName", "My Organization"},
		{"LongName", "This Is A Very Long Organization Name That Tests Limits"},
		{"WithSpecialChars", "Org-Name_123"},
		{"InvalidChars", "Org@#$%^&*()"},
	}

	service := &Service{logger: logrus.New()}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = service.validateOrganizationName(tc.input)
			}
		})
	}
}

// BenchmarkEmailValidation benchmarks email validation
func BenchmarkEmailValidation(b *testing.B) {
	testEmails := []string{
		"user@example.com",
		"very.long.email.address@subdomain.example.com",
		"invalid-email",
		"user+tag@example.com",
		"user@example",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, email := range testEmails {
			_ = isValidEmail(email)
		}
	}
}

// BenchmarkMembershipLookup benchmarks member lookup performance
func BenchmarkMembershipLookup(b *testing.B) {
	repo := newMockRepository()

	// Create organizations with varying member counts
	memberCounts := []int{10, 100, 1000, 10000}

	for _, count := range memberCounts {
		b.Run(fmt.Sprintf("Members-%d", count), func(b *testing.B) {
			// Reset repo
			repo.members = make([]*OrganizationMember, 0, count)

			orgID := "test-org"
			// Add members
			for i := 0; i < count; i++ {
				repo.members = append(repo.members, &OrganizationMember{
					OrganizationID: orgID,
					UserID:         fmt.Sprintf("user-%d", i),
					Role:           RoleMember,
				})
			}

			ctx := context.Background()
			// Look up a member in the middle
			targetUserID := fmt.Sprintf("user-%d", count/2)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = repo.GetMember(ctx, orgID, targetUserID)
			}
		})
	}
}

// BenchmarkInvitationProcessing benchmarks invitation acceptance flow
func BenchmarkInvitationProcessing(b *testing.B) {
	repo := newMockRepository()
	logger := logrus.New()
	service := NewService(repo, logger)

	// Setup organization
	orgID := "test-org"
	org := &Organization{
		ID:         orgID,
		Name:       "Test Org",
		OwnerID:    "owner-id",
		MaxMembers: 10000,
	}
	repo.organizations = append(repo.organizations, org)

	ctx := context.Background()

	b.Run("CreateInvitation", func(b *testing.B) {
		// Add owner
		repo.members = append(repo.members, &OrganizationMember{
			OrganizationID: orgID,
			UserID:         "owner-id",
			Role:           RoleOwner,
		})

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = service.CreateInvitation(ctx, orgID, "owner-id", &CreateInvitationInput{
				Email: fmt.Sprintf("user%d@example.com", i),
				Role:  RoleMember,
			})
		}
	})

	b.Run("AcceptInvitation", func(b *testing.B) {
		// Pre-create invitations
		for i := 0; i < b.N; i++ {
			inv := &OrganizationInvitation{
				ID:             fmt.Sprintf("inv-%d", i),
				OrganizationID: orgID,
				Email:          fmt.Sprintf("user%d@example.com", i),
				Role:           RoleMember,
				Token:          fmt.Sprintf("token-%d", i),
				ExpiresAt:      time.Now().Add(24 * time.Hour),
				Organization:   org,
			}
			repo.invitations = append(repo.invitations, inv)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			token := fmt.Sprintf("token-%d", i)
			userID := fmt.Sprintf("user-%d", i)
			_, _ = service.AcceptInvitation(ctx, token, userID)
		}
	})
}

// Helper rate limiter for benchmarking
type benchmarkRateLimiter struct {
	mu     sync.RWMutex
	limits map[string]*limitInfo
}

type limitInfo struct {
	count     int
	resetTime time.Time
}

func (r *benchmarkRateLimiter) CheckBothLimits(userID, orgID string) (bool, string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	key := fmt.Sprintf("%s:%s", userID, orgID)

	if info, exists := r.limits[key]; exists {
		if now.After(info.resetTime) {
			// Reset the counter
			info.count = 1
			info.resetTime = now.Add(time.Hour)
		} else {
			info.count++
			if info.count > 20 {
				return false, "rate limit exceeded"
			}
		}
	} else {
		r.limits[key] = &limitInfo{
			count:     1,
			resetTime: now.Add(time.Hour),
		}
	}

	return true, ""
}

// Benchmark results documentation
/*
Expected Performance Targets:

1. Permission Check (without DB): < 100ns
   - Simple map lookup should be very fast
   - No allocations expected

2. Permission Check (with DB): < 10ms
   - Includes mock database lookup
   - Should scale linearly with member count

3. Concurrent Permission Checks: Linear scaling
   - 10 concurrent: ~10x single operation
   - 100 concurrent: ~100x single operation
   - No lock contention expected

4. Rate Limiting Overhead: < 1ms
   - Map lookup and counter increment
   - Minimal overhead on operations

5. Audit Log Creation: < 5ms
   - Simple append operation
   - Should not block main operation

6. Token Generation: < 10ms
   - Crypto operations are expensive
   - Acceptable for security-critical operations

7. Input Validation: < 1μs
   - Regex matching should be fast
   - Compile regex once if possible

8. Member Lookup:
   - 10 members: < 100μs
   - 100 members: < 1ms
   - 1000 members: < 10ms
   - 10000 members: < 100ms

To run benchmarks:
  go test -bench=. -benchmem -benchtime=10s ./internal/organization

To profile specific benchmark:
  go test -bench=BenchmarkPermissionCheck -cpuprofile=cpu.prof ./internal/organization
  go tool pprof cpu.prof

To check for memory allocations:
  go test -bench=. -benchmem | grep "0 allocs"
*/
