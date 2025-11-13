package email

import (
	"bytes"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestSendOrganizationInvitationEmail(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Suppress logs during tests

	mockSvc := NewMockEmailService(logger)

	tests := []struct {
		name        string
		email       string
		orgName     string
		inviterName string
		role        string
		wantError   bool
	}{
		{
			name:        "successful invitation email",
			email:       "newuser@example.com",
			orgName:     "Acme Corp",
			inviterName: "John Doe",
			role:        "member",
			wantError:   false,
		},
		{
			name:        "admin role invitation",
			email:       "admin@example.com",
			orgName:     "Tech Startup",
			inviterName: "Jane Smith",
			role:        "admin",
			wantError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc.ClearSentEmails()
			invitationURL := "https://sqlstudio.io/invitations/accept?token=abc123"

			err := mockSvc.SendOrganizationInvitationEmail(
				tt.email,
				tt.orgName,
				tt.inviterName,
				tt.role,
				invitationURL,
			)

			if (err != nil) != tt.wantError {
				t.Errorf("SendOrganizationInvitationEmail() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError {
				sent := mockSvc.GetSentEmails()
				if len(sent) != 1 {
					t.Errorf("expected 1 email sent, got %d", len(sent))
					return
				}

				email := sent[0]
				if email.To != tt.email {
					t.Errorf("email.To = %s, want %s", email.To, tt.email)
				}
				if email.Type != "organization_invitation" {
					t.Errorf("email.Type = %s, want organization_invitation", email.Type)
				}
				if email.OrgName != tt.orgName {
					t.Errorf("email.OrgName = %s, want %s", email.OrgName, tt.orgName)
				}
				if email.InviterName != tt.inviterName {
					t.Errorf("email.InviterName = %s, want %s", email.InviterName, tt.inviterName)
				}
				if email.Role != tt.role {
					t.Errorf("email.Role = %s, want %s", email.Role, tt.role)
				}
				if email.URL != invitationURL {
					t.Errorf("email.URL = %s, want %s", email.URL, invitationURL)
				}
			}
		})
	}
}

func TestSendOrganizationWelcomeEmail(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	mockSvc := NewMockEmailService(logger)

	tests := []struct {
		name      string
		email     string
		userName  string
		orgName   string
		role      string
		wantError bool
	}{
		{
			name:      "successful welcome email",
			email:     "user@example.com",
			userName:  "Alice Johnson",
			orgName:   "Dev Team",
			role:      "member",
			wantError: false,
		},
		{
			name:      "admin welcome email",
			email:     "admin@example.com",
			userName:  "Bob Smith",
			orgName:   "Engineering",
			role:      "admin",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc.ClearSentEmails()

			err := mockSvc.SendOrganizationWelcomeEmail(
				tt.email,
				tt.userName,
				tt.orgName,
				tt.role,
			)

			if (err != nil) != tt.wantError {
				t.Errorf("SendOrganizationWelcomeEmail() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError {
				sent := mockSvc.GetSentEmails()
				if len(sent) != 1 {
					t.Errorf("expected 1 email sent, got %d", len(sent))
					return
				}

				email := sent[0]
				if email.To != tt.email {
					t.Errorf("email.To = %s, want %s", email.To, tt.email)
				}
				if email.Type != "organization_welcome" {
					t.Errorf("email.Type = %s, want organization_welcome", email.Type)
				}
				if email.Name != tt.userName {
					t.Errorf("email.Name = %s, want %s", email.Name, tt.userName)
				}
				if email.OrgName != tt.orgName {
					t.Errorf("email.OrgName = %s, want %s", email.OrgName, tt.orgName)
				}
				if email.Role != tt.role {
					t.Errorf("email.Role = %s, want %s", email.Role, tt.role)
				}
			}
		})
	}
}

func TestSendMemberRemovedEmail(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	mockSvc := NewMockEmailService(logger)

	tests := []struct {
		name      string
		email     string
		orgName   string
		wantError bool
	}{
		{
			name:      "successful removal notification",
			email:     "removed@example.com",
			orgName:   "Old Team",
			wantError: false,
		},
		{
			name:      "removal from large org",
			email:     "user@example.com",
			orgName:   "Enterprise Corporation",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc.ClearSentEmails()

			err := mockSvc.SendMemberRemovedEmail(tt.email, tt.orgName)

			if (err != nil) != tt.wantError {
				t.Errorf("SendMemberRemovedEmail() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError {
				sent := mockSvc.GetSentEmails()
				if len(sent) != 1 {
					t.Errorf("expected 1 email sent, got %d", len(sent))
					return
				}

				email := sent[0]
				if email.To != tt.email {
					t.Errorf("email.To = %s, want %s", email.To, tt.email)
				}
				if email.Type != "member_removed" {
					t.Errorf("email.Type = %s, want member_removed", email.Type)
				}
				if email.OrgName != tt.orgName {
					t.Errorf("email.OrgName = %s, want %s", email.OrgName, tt.orgName)
				}
			}
		})
	}
}

func TestEmailTemplateRendering(t *testing.T) {
	// Test that templates can be loaded and parsed without errors
	templates, err := loadTemplates()
	if err != nil {
		t.Fatalf("Failed to load templates: %v", err)
	}

	if templates.organizationInvitation == nil {
		t.Error("organization invitation template is nil")
	}
	if templates.organizationWelcome == nil {
		t.Error("organization welcome template is nil")
	}
	if templates.memberRemoved == nil {
		t.Error("member removed template is nil")
	}

	// Test template execution with sample data
	data := TemplateData{
		Email:         "test@example.com",
		Name:          "Test User",
		OrgName:       "Test Org",
		InviterName:   "Inviter",
		Role:          "member",
		InvitationURL: "https://example.com/invite",
		Year:          2025,
	}

	// Test organization invitation template
	var buf bytes.Buffer
	if err := templates.organizationInvitation.Execute(&buf, data); err != nil {
		t.Errorf("Failed to execute organization invitation template: %v", err)
	}

	// Verify template contains expected content
	output := buf.String()
	if output == "" {
		t.Error("organization invitation template produced empty output")
	}
}

func TestMultipleEmails(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	mockSvc := NewMockEmailService(logger)

	// Send multiple types of emails
	_ = mockSvc.SendOrganizationInvitationEmail(
		"invite@example.com",
		"Team A",
		"Manager",
		"member",
		"https://example.com/invite1",
	) // Test setup - error not relevant

	_ = mockSvc.SendOrganizationWelcomeEmail(
		"welcome@example.com",
		"New User",
		"Team B",
		"admin",
	) // Test setup - error not relevant

	_ = mockSvc.SendMemberRemovedEmail("removed@example.com", "Team C") // Test setup - error not relevant

	sent := mockSvc.GetSentEmails()
	if len(sent) != 3 {
		t.Errorf("expected 3 emails sent, got %d", len(sent))
	}

	// Verify email types
	types := make(map[string]bool)
	for _, email := range sent {
		types[email.Type] = true
	}

	expectedTypes := []string{
		"organization_invitation",
		"organization_welcome",
		"member_removed",
	}

	for _, expectedType := range expectedTypes {
		if !types[expectedType] {
			t.Errorf("expected email type %s not found", expectedType)
		}
	}
}
