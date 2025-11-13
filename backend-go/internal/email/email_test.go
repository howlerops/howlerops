package email_test

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sql-studio/backend-go/internal/email"
)

// TestMockEmailService tests the mock email service implementation
func TestMockEmailService(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	mockSvc := email.NewMockEmailService(logger)

	t.Run("sends verification email", func(t *testing.T) {
		err := mockSvc.SendVerificationEmail("test@example.com", "token123", "http://example.com/verify")
		require.NoError(t, err)

		sent := mockSvc.GetSentEmails()
		assert.Len(t, sent, 1)
		assert.Equal(t, "test@example.com", sent[0].To)
		assert.Equal(t, "verification", sent[0].Type)
		assert.Equal(t, "token123", sent[0].Token)
	})

	t.Run("sends password reset email", func(t *testing.T) {
		mockSvc.ClearSentEmails()

		err := mockSvc.SendPasswordResetEmail("user@example.com", "resettoken", "http://example.com/reset")
		require.NoError(t, err)

		sent := mockSvc.GetSentEmails()
		assert.Len(t, sent, 1)
		assert.Equal(t, "user@example.com", sent[0].To)
		assert.Equal(t, "password_reset", sent[0].Type)
		assert.Equal(t, "resettoken", sent[0].Token)
	})

	t.Run("sends welcome email", func(t *testing.T) {
		mockSvc.ClearSentEmails()

		err := mockSvc.SendWelcomeEmail("newuser@example.com", "John Doe")
		require.NoError(t, err)

		sent := mockSvc.GetSentEmails()
		assert.Len(t, sent, 1)
		assert.Equal(t, "newuser@example.com", sent[0].To)
		assert.Equal(t, "welcome", sent[0].Type)
	})

	t.Run("sends organization invitation email", func(t *testing.T) {
		mockSvc.ClearSentEmails()

		err := mockSvc.SendOrganizationInvitationEmail(
			"invitee@example.com",
			"Acme Corp",
			"John Doe",
			"admin",
			"http://example.com/invite/token123",
		)
		require.NoError(t, err)

		sent := mockSvc.GetSentEmails()
		assert.Len(t, sent, 1)
		assert.Equal(t, "invitee@example.com", sent[0].To)
		assert.Equal(t, "organization_invitation", sent[0].Type)
		assert.Equal(t, "Acme Corp", sent[0].OrgName)
		assert.Equal(t, "John Doe", sent[0].InviterName)
		assert.Equal(t, "admin", sent[0].Role)
	})

	t.Run("sends organization welcome email", func(t *testing.T) {
		mockSvc.ClearSentEmails()

		err := mockSvc.SendOrganizationWelcomeEmail(
			"newmember@example.com",
			"Jane Smith",
			"Tech Startup",
			"member",
		)
		require.NoError(t, err)

		sent := mockSvc.GetSentEmails()
		assert.Len(t, sent, 1)
		assert.Equal(t, "newmember@example.com", sent[0].To)
		assert.Equal(t, "organization_welcome", sent[0].Type)
		assert.Equal(t, "Tech Startup", sent[0].OrgName)
		assert.Equal(t, "member", sent[0].Role)
	})

	t.Run("sends member removed email", func(t *testing.T) {
		mockSvc.ClearSentEmails()

		err := mockSvc.SendMemberRemovedEmail("removed@example.com", "Old Company")
		require.NoError(t, err)

		sent := mockSvc.GetSentEmails()
		assert.Len(t, sent, 1)
		assert.Equal(t, "removed@example.com", sent[0].To)
		assert.Equal(t, "member_removed", sent[0].Type)
		assert.Equal(t, "Old Company", sent[0].OrgName)
	})
}

// TestEmailTemplateGeneration tests that email templates can be generated without errors
func TestEmailTemplateGeneration(t *testing.T) {
	// Note: In a real implementation with Resend, you would mock the HTTP client
	// For now, we test template parsing by attempting to create the service

	t.Run("email service creation requires API key", func(t *testing.T) {
		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		_, err := email.NewResendEmailService("", "from@example.com", "Test Sender", logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "API key is required")
	})

	t.Run("email service creation requires from email", func(t *testing.T) {
		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		_, err := email.NewResendEmailService("test-api-key", "", "Test Sender", logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "from email is required")
	})

	t.Run("email service creation succeeds with valid params", func(t *testing.T) {
		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		svc, err := email.NewResendEmailService("test-api-key", "from@example.com", "Test Sender", logger)
		assert.NoError(t, err)
		assert.NotNil(t, svc)
	})
}

// TestEmailContentValidation tests that email content is properly formatted
func TestEmailContentValidation(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	mockSvc := email.NewMockEmailService(logger)

	t.Run("verification email contains correct elements", func(t *testing.T) {
		email := "test@example.com"
		token := "verification-token-123"
		url := "http://example.com/verify?token=" + token

		err := mockSvc.SendVerificationEmail(email, token, url)
		require.NoError(t, err)

		sent := mockSvc.GetSentEmails()
		assert.Equal(t, email, sent[0].To)
		assert.Equal(t, token, sent[0].Token)
		assert.Equal(t, url, sent[0].URL)
	})

	t.Run("invitation email contains organization details", func(t *testing.T) {
		mockSvc.ClearSentEmails()

		orgName := "My Company"
		inviterName := "Boss Person"
		role := "admin"

		err := mockSvc.SendOrganizationInvitationEmail(
			"invite@example.com",
			orgName,
			inviterName,
			role,
			"http://example.com/accept",
		)
		require.NoError(t, err)

		sent := mockSvc.GetSentEmails()
		require.Len(t, sent, 1)
		assert.Equal(t, orgName, sent[0].OrgName)
		assert.Equal(t, inviterName, sent[0].InviterName)
		assert.Equal(t, role, sent[0].Role)
	})
}

// TestEmailErrorHandling tests error scenarios
func TestEmailErrorHandling(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	t.Run("mock service never fails", func(t *testing.T) {
		mockSvc := email.NewMockEmailService(logger)

		// Mock service should never return errors
		err := mockSvc.SendVerificationEmail("", "", "")
		assert.NoError(t, err, "Mock service should not fail even with empty params")
	})

	t.Run("sent emails can be cleared", func(t *testing.T) {
		mockSvc := email.NewMockEmailService(logger)

		_ = mockSvc.SendVerificationEmail("test@example.com", "token", "url") // Best-effort mock in test
		assert.Len(t, mockSvc.GetSentEmails(), 1)

		mockSvc.ClearSentEmails()
		assert.Len(t, mockSvc.GetSentEmails(), 0)
	})

	t.Run("multiple emails accumulate", func(t *testing.T) {
		mockSvc := email.NewMockEmailService(logger)

		_ = mockSvc.SendVerificationEmail("user1@example.com", "token1", "url1") // Best-effort mock in test
		_ = mockSvc.SendPasswordResetEmail("user2@example.com", "token2", "url2") // Best-effort mock in test
		_ = mockSvc.SendWelcomeEmail("user3@example.com", "User 3") // Best-effort mock in test

		sent := mockSvc.GetSentEmails()
		assert.Len(t, sent, 3)
		assert.Equal(t, "verification", sent[0].Type)
		assert.Equal(t, "password_reset", sent[1].Type)
		assert.Equal(t, "welcome", sent[2].Type)
	})
}

// TestEmailServiceInterface tests that both services implement the same interface
func TestEmailServiceInterface(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Both services should implement the EmailService interface
	var _ email.EmailService = email.NewMockEmailService(logger)

	// ResendEmailService also implements it (but we can't test without real credentials)
	// This is compile-time checked
}

// TestEmailCaseSensitivity tests email handling with different cases
func TestEmailCaseSensitivity(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	mockSvc := email.NewMockEmailService(logger)

	testCases := []string{
		"Test@Example.com",
		"TEST@EXAMPLE.COM",
		"test@example.com",
	}

	for _, email := range testCases {
		mockSvc.ClearSentEmails()
		err := mockSvc.SendVerificationEmail(email, "token", "url")
		require.NoError(t, err)

		sent := mockSvc.GetSentEmails()
		assert.Equal(t, email, sent[0].To, "Email should preserve original case")
	}
}

// TestEmailNameFallback tests that empty names fall back to defaults
func TestEmailNameFallback(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	mockSvc := email.NewMockEmailService(logger)

	t.Run("empty name uses fallback", func(t *testing.T) {
		// The welcome email should handle empty names gracefully
		// Note: The actual fallback is in the template/service implementation
		err := mockSvc.SendWelcomeEmail("test@example.com", "")
		assert.NoError(t, err, "Should handle empty name gracefully")
	})
}

// Benchmark email service operations
func BenchmarkMockEmailService(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	mockSvc := email.NewMockEmailService(logger)

	b.Run("SendVerificationEmail", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = mockSvc.SendVerificationEmail("bench@example.com", "token", "url") // Best-effort mock in test
		}
	})

	b.Run("SendOrganizationInvitationEmail", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = mockSvc.SendOrganizationInvitationEmail( // Best-effort mock in test
				"bench@example.com",
				"Company",
				"Inviter",
				"member",
				"url",
			)
		}
	})
}

// TestRealWorldScenario tests a complete email flow
func TestRealWorldScenario_CompleteInvitationFlow(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	mockSvc := email.NewMockEmailService(logger)

	// Scenario: Complete organization invitation workflow
	orgName := "Tech Startup Inc"
	memberEmail := "newdev@example.com"
	inviterName := "John Doe"

	// Step 1: Send invitation
	err := mockSvc.SendOrganizationInvitationEmail(
		memberEmail,
		orgName,
		inviterName,
		"developer",
		"https://app.sqlstudio.io/invitations/accept?token=abc123",
	)
	require.NoError(t, err)

	sent := mockSvc.GetSentEmails()
	require.Len(t, sent, 1)
	assert.Equal(t, "organization_invitation", sent[0].Type)

	// Step 2: After acceptance, send welcome email
	err = mockSvc.SendOrganizationWelcomeEmail(
		memberEmail,
		"New Developer",
		orgName,
		"developer",
	)
	require.NoError(t, err)

	sent = mockSvc.GetSentEmails()
	require.Len(t, sent, 2)
	assert.Equal(t, "organization_welcome", sent[1].Type)

	// Step 3: If removed later, send removal notification
	err = mockSvc.SendMemberRemovedEmail(memberEmail, orgName)
	require.NoError(t, err)

	sent = mockSvc.GetSentEmails()
	require.Len(t, sent, 3)
	assert.Equal(t, "member_removed", sent[2].Type)

	// Verify all emails went to the same recipient
	for _, email := range sent {
		assert.Equal(t, memberEmail, email.To)
	}
}
