package organization

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// mockRepository is a mock implementation for testing
type mockRepository struct {
	mu            sync.RWMutex
	organizations []*Organization
	members       []*OrganizationMember
	invitations   []*OrganizationInvitation
	auditLogs     []*AuditLog
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		organizations: make([]*Organization, 0),
		members:       make([]*OrganizationMember, 0),
		invitations:   make([]*OrganizationInvitation, 0),
		auditLogs:     make([]*AuditLog, 0),
	}
}

// Organization methods
func (m *mockRepository) Create(ctx context.Context, org *Organization) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	org.ID = fmt.Sprintf("org-%d", len(m.organizations)+1)
	org.CreatedAt = time.Now()
	org.UpdatedAt = time.Now()
	m.organizations = append(m.organizations, org)

	// Add owner as first member
	m.members = append(m.members, &OrganizationMember{
		ID:             fmt.Sprintf("member-%d", len(m.members)+1),
		OrganizationID: org.ID,
		UserID:         org.OwnerID,
		Role:           RoleOwner,
		JoinedAt:       time.Now(),
	})

	return nil
}

func (m *mockRepository) GetByID(ctx context.Context, id string) (*Organization, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, org := range m.organizations {
		if org.ID == id {
			return org, nil
		}
	}
	return nil, fmt.Errorf("organization not found")
}

func (m *mockRepository) GetByUserID(ctx context.Context, userID string) ([]*Organization, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*Organization
	for _, member := range m.members {
		if member.UserID == userID {
			for _, org := range m.organizations {
				if org.ID == member.OrganizationID {
					result = append(result, org)
					break
				}
			}
		}
	}
	return result, nil
}

func (m *mockRepository) Update(ctx context.Context, org *Organization) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, existing := range m.organizations {
		if existing.ID == org.ID {
			org.UpdatedAt = time.Now()
			m.organizations[i] = org
			return nil
		}
	}
	return fmt.Errorf("organization not found")
}

func (m *mockRepository) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, org := range m.organizations {
		if org.ID == id {
			now := time.Now()
			m.organizations[i].DeletedAt = &now
			return nil
		}
	}
	return fmt.Errorf("organization not found")
}

// Member methods
func (m *mockRepository) GetMembers(ctx context.Context, orgID string) ([]*OrganizationMember, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*OrganizationMember
	for _, member := range m.members {
		if member.OrganizationID == orgID {
			// Clone to avoid mutations
			memberCopy := *member
			result = append(result, &memberCopy)
		}
	}
	return result, nil
}

func (m *mockRepository) GetMember(ctx context.Context, orgID, userID string) (*OrganizationMember, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, member := range m.members {
		if member.OrganizationID == orgID && member.UserID == userID {
			// Return a copy
			memberCopy := *member
			return &memberCopy, nil
		}
	}
	return nil, fmt.Errorf("member not found")
}

func (m *mockRepository) AddMember(ctx context.Context, member *OrganizationMember) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already a member
	for _, existing := range m.members {
		if existing.OrganizationID == member.OrganizationID && existing.UserID == member.UserID {
			return fmt.Errorf("user is already a member")
		}
	}

	member.ID = fmt.Sprintf("member-%d", len(m.members)+1)
	member.JoinedAt = time.Now()
	m.members = append(m.members, member)
	return nil
}

func (m *mockRepository) UpdateMemberRole(ctx context.Context, orgID, userID string, role OrganizationRole) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, member := range m.members {
		if member.OrganizationID == orgID && member.UserID == userID {
			m.members[i].Role = role
			return nil
		}
	}
	return fmt.Errorf("member not found")
}

func (m *mockRepository) RemoveMember(ctx context.Context, orgID, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, member := range m.members {
		if member.OrganizationID == orgID && member.UserID == userID {
			m.members = append(m.members[:i], m.members[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("member not found")
}

func (m *mockRepository) GetMemberCount(ctx context.Context, orgID string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, member := range m.members {
		if member.OrganizationID == orgID {
			count++
		}
	}
	return count, nil
}

// Invitation methods
func (m *mockRepository) CreateInvitation(ctx context.Context, inv *OrganizationInvitation) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for duplicate
	for _, existing := range m.invitations {
		if existing.OrganizationID == inv.OrganizationID &&
			existing.Email == inv.Email &&
			existing.AcceptedAt == nil &&
			!existing.IsExpired() {
			return fmt.Errorf("UNIQUE constraint failed: invitation already exists")
		}
	}

	inv.ID = fmt.Sprintf("inv-%d", len(m.invitations)+1)
	inv.CreatedAt = time.Now()
	m.invitations = append(m.invitations, inv)
	return nil
}

func (m *mockRepository) GetInvitation(ctx context.Context, id string) (*OrganizationInvitation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, inv := range m.invitations {
		if inv.ID == id {
			return inv, nil
		}
	}
	return nil, fmt.Errorf("invitation not found")
}

func (m *mockRepository) GetInvitationByToken(ctx context.Context, token string) (*OrganizationInvitation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, inv := range m.invitations {
		if inv.Token == token {
			// Include organization data
			for _, org := range m.organizations {
				if org.ID == inv.OrganizationID {
					inv.Organization = org
					break
				}
			}
			return inv, nil
		}
	}
	return nil, fmt.Errorf("invitation not found")
}

func (m *mockRepository) GetInvitationsByOrg(ctx context.Context, orgID string) ([]*OrganizationInvitation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*OrganizationInvitation
	for _, inv := range m.invitations {
		if inv.OrganizationID == orgID {
			result = append(result, inv)
		}
	}
	return result, nil
}

func (m *mockRepository) GetInvitationsByEmail(ctx context.Context, email string) ([]*OrganizationInvitation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*OrganizationInvitation
	for _, inv := range m.invitations {
		if inv.Email == email && inv.AcceptedAt == nil && !inv.IsExpired() {
			result = append(result, inv)
		}
	}
	return result, nil
}

func (m *mockRepository) UpdateInvitation(ctx context.Context, inv *OrganizationInvitation) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, existing := range m.invitations {
		if existing.ID == inv.ID {
			m.invitations[i] = inv
			return nil
		}
	}
	return fmt.Errorf("invitation not found")
}

func (m *mockRepository) DeleteInvitation(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, inv := range m.invitations {
		if inv.ID == id {
			m.invitations = append(m.invitations[:i], m.invitations[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("invitation not found")
}

// Audit log methods
func (m *mockRepository) CreateAuditLog(ctx context.Context, log *AuditLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.ID = fmt.Sprintf("audit-%d", len(m.auditLogs)+1)
	log.CreatedAt = time.Now()
	m.auditLogs = append(m.auditLogs, log)
	return nil
}

func (m *mockRepository) GetAuditLogs(ctx context.Context, orgID string, limit, offset int) ([]*AuditLog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*AuditLog
	for _, log := range m.auditLogs {
		if log.OrganizationID != nil && *log.OrganizationID == orgID {
			result = append(result, log)
		}
	}

	// Apply pagination
	start := offset
	if start >= len(result) {
		return []*AuditLog{}, nil
	}

	end := start + limit
	if end > len(result) {
		end = len(result)
	}

	return result[start:end], nil
}

// mockRateLimiter is a mock rate limiter for testing
type mockRateLimiter struct {
	mu         sync.Mutex
	counts     map[string]int
	limitAfter int
}

func (m *mockRateLimiter) CheckBothLimits(userID, orgID string) (bool, string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Initialize map if nil
	if m.counts == nil {
		m.counts = make(map[string]int)
	}

	key := fmt.Sprintf("%s:%s", userID, orgID)
	m.counts[key]++

	if m.counts[key] > m.limitAfter {
		return false, fmt.Sprintf("Rate limit exceeded: max %d invitations per hour", m.limitAfter)
	}

	return true, ""
}

// newTestLogger creates a test logger
func newTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	return logger
}
