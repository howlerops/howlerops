package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// EmailService defines the interface for email operations
type EmailService interface {
	SendVerificationEmail(email, token, verificationURL string) error
	SendPasswordResetEmail(email, token, resetURL string) error
	SendWelcomeEmail(email, name string) error
	SendOrganizationInvitationEmail(email, orgName, inviterName, role, invitationURL string) error
	SendOrganizationWelcomeEmail(email, name, orgName, role string) error
	SendMemberRemovedEmail(email, orgName string) error
}

// ResendEmailService implements EmailService using Resend API
type ResendEmailService struct {
	apiKey     string
	fromEmail  string
	fromName   string
	baseURL    string
	httpClient *http.Client
	logger     *logrus.Logger
	templates  *emailTemplates
}

// ResendRequest represents a Resend API email request
type ResendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

// ResendResponse represents a Resend API response
type ResendResponse struct {
	ID    string `json:"id"`
	Error *struct {
		Message string `json:"message"`
		Name    string `json:"name"`
	} `json:"error,omitempty"`
}

// emailTemplates holds parsed email templates
type emailTemplates struct {
	verification           *template.Template
	passwordReset          *template.Template
	welcome                *template.Template
	organizationInvitation *template.Template
	organizationWelcome    *template.Template
	memberRemoved          *template.Template
}

// TemplateData holds data for email templates
type TemplateData struct {
	Email           string
	Name            string
	Token           string
	URL             string
	VerificationURL string
	ResetURL        string
	OrgName         string
	InviterName     string
	Role            string
	InvitationURL   string
	Year            int
}

// NewResendEmailService creates a new Resend email service
func NewResendEmailService(apiKey, fromEmail, fromName string, logger *logrus.Logger) (*ResendEmailService, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("resend API key is required")
	}
	if fromEmail == "" {
		return nil, fmt.Errorf("from email is required")
	}
	if fromName == "" {
		fromName = "Howlerops"
	}

	templates, err := loadTemplates()
	if err != nil {
		return nil, fmt.Errorf("failed to load email templates: %w", err)
	}

	return &ResendEmailService{
		apiKey:    apiKey,
		fromEmail: fromEmail,
		fromName:  fromName,
		baseURL:   "https://api.resend.com",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:    logger,
		templates: templates,
	}, nil
}

// loadTemplates loads and parses email templates
func loadTemplates() (*emailTemplates, error) {
	verification, err := template.New("verification").Parse(verificationTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse verification template: %w", err)
	}

	passwordReset, err := template.New("password_reset").Parse(passwordResetTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse password reset template: %w", err)
	}

	welcome, err := template.New("welcome").Parse(welcomeTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse welcome template: %w", err)
	}

	organizationInvitation, err := template.New("organization_invitation").Parse(organizationInvitationTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse organization invitation template: %w", err)
	}

	organizationWelcome, err := template.New("organization_welcome").Parse(organizationWelcomeTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse organization welcome template: %w", err)
	}

	memberRemoved, err := template.New("member_removed").Parse(memberRemovedTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse member removed template: %w", err)
	}

	return &emailTemplates{
		verification:           verification,
		passwordReset:          passwordReset,
		welcome:                welcome,
		organizationInvitation: organizationInvitation,
		organizationWelcome:    organizationWelcome,
		memberRemoved:          memberRemoved,
	}, nil
}

// SendVerificationEmail sends an email verification link
func (s *ResendEmailService) SendVerificationEmail(email, token, verificationURL string) error {
	data := TemplateData{
		Email:           email,
		Token:           token,
		VerificationURL: verificationURL,
		Year:            time.Now().Year(),
	}

	var body bytes.Buffer
	if err := s.templates.verification.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute verification template: %w", err)
	}

	req := ResendRequest{
		From:    fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail),
		To:      []string{email},
		Subject: "Verify your Howlerops account",
		HTML:    body.String(),
	}

	if err := s.sendEmail(req); err != nil {
		s.logger.WithError(err).WithField("email", email).Error("Failed to send verification email")
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	s.logger.WithField("email", email).Info("Verification email sent successfully")
	return nil
}

// SendPasswordResetEmail sends a password reset link
func (s *ResendEmailService) SendPasswordResetEmail(email, token, resetURL string) error {
	data := TemplateData{
		Email:    email,
		Token:    token,
		ResetURL: resetURL,
		Year:     time.Now().Year(),
	}

	var body bytes.Buffer
	if err := s.templates.passwordReset.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute password reset template: %w", err)
	}

	req := ResendRequest{
		From:    fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail),
		To:      []string{email},
		Subject: "Reset your Howlerops password",
		HTML:    body.String(),
	}

	if err := s.sendEmail(req); err != nil {
		s.logger.WithError(err).WithField("email", email).Error("Failed to send password reset email")
		return fmt.Errorf("failed to send password reset email: %w", err)
	}

	s.logger.WithField("email", email).Info("Password reset email sent successfully")
	return nil
}

// SendWelcomeEmail sends a welcome email to new users
func (s *ResendEmailService) SendWelcomeEmail(email, name string) error {
	if name == "" {
		name = "there"
	}

	data := TemplateData{
		Email: email,
		Name:  name,
		Year:  time.Now().Year(),
	}

	var body bytes.Buffer
	if err := s.templates.welcome.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute welcome template: %w", err)
	}

	req := ResendRequest{
		From:    fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail),
		To:      []string{email},
		Subject: "Welcome to Howlerops!",
		HTML:    body.String(),
	}

	if err := s.sendEmail(req); err != nil {
		s.logger.WithError(err).WithField("email", email).Error("Failed to send welcome email")
		return fmt.Errorf("failed to send welcome email: %w", err)
	}

	s.logger.WithField("email", email).Info("Welcome email sent successfully")
	return nil
}

// SendOrganizationInvitationEmail sends an organization invitation email
func (s *ResendEmailService) SendOrganizationInvitationEmail(email, orgName, inviterName, role, invitationURL string) error {
	data := TemplateData{
		Email:         email,
		OrgName:       orgName,
		InviterName:   inviterName,
		Role:          role,
		InvitationURL: invitationURL,
		Year:          time.Now().Year(),
	}

	var body bytes.Buffer
	if err := s.templates.organizationInvitation.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute organization invitation template: %w", err)
	}

	req := ResendRequest{
		From:    fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail),
		To:      []string{email},
		Subject: fmt.Sprintf("You're invited to join %s on Howlerops", orgName),
		HTML:    body.String(),
	}

	if err := s.sendEmail(req); err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"email":    email,
			"org_name": orgName,
		}).Error("Failed to send organization invitation email")
		return fmt.Errorf("failed to send organization invitation email: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"email":    email,
		"org_name": orgName,
	}).Info("Organization invitation email sent successfully")
	return nil
}

// SendOrganizationWelcomeEmail sends a welcome email for organization membership
func (s *ResendEmailService) SendOrganizationWelcomeEmail(email, name, orgName, role string) error {
	if name == "" {
		name = "there"
	}

	data := TemplateData{
		Email:   email,
		Name:    name,
		OrgName: orgName,
		Role:    role,
		Year:    time.Now().Year(),
	}

	var body bytes.Buffer
	if err := s.templates.organizationWelcome.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute organization welcome template: %w", err)
	}

	req := ResendRequest{
		From:    fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail),
		To:      []string{email},
		Subject: fmt.Sprintf("Welcome to %s!", orgName),
		HTML:    body.String(),
	}

	if err := s.sendEmail(req); err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"email":    email,
			"org_name": orgName,
		}).Error("Failed to send organization welcome email")
		return fmt.Errorf("failed to send organization welcome email: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"email":    email,
		"org_name": orgName,
	}).Info("Organization welcome email sent successfully")
	return nil
}

// SendMemberRemovedEmail sends a notification when a member is removed from an organization
func (s *ResendEmailService) SendMemberRemovedEmail(email, orgName string) error {
	data := TemplateData{
		Email:   email,
		OrgName: orgName,
		Year:    time.Now().Year(),
	}

	var body bytes.Buffer
	if err := s.templates.memberRemoved.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute member removed template: %w", err)
	}

	req := ResendRequest{
		From:    fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail),
		To:      []string{email},
		Subject: fmt.Sprintf("You've been removed from %s", orgName),
		HTML:    body.String(),
	}

	if err := s.sendEmail(req); err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"email":    email,
			"org_name": orgName,
		}).Error("Failed to send member removed email")
		return fmt.Errorf("failed to send member removed email: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"email":    email,
		"org_name": orgName,
	}).Info("Member removed email sent successfully")
	return nil
}

// sendEmail sends an email using the Resend API
func (s *ResendEmailService) sendEmail(req ResendRequest) error {
	// Prepare request body
	jsonBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", s.baseURL+"/emails", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			s.logger.WithError(err).Error("Failed to close response body")
		}
	}()

	// Parse response
	var resendResp ResendResponse
	if err := json.NewDecoder(resp.Body).Decode(&resendResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		if resendResp.Error != nil {
			return fmt.Errorf("resend API error: %s (%s)", resendResp.Error.Message, resendResp.Error.Name)
		}
		return fmt.Errorf("resend API returned status %d", resp.StatusCode)
	}

	return nil
}

// MockEmailService is a mock implementation for testing
type MockEmailService struct {
	logger     *logrus.Logger
	sentEmails []SentEmail
}

// SentEmail tracks sent emails in mock service
type SentEmail struct {
	To          string
	Type        string
	Token       string
	URL         string
	OrgName     string
	InviterName string
	Role        string
	Name        string
	SentAt      time.Time
}

// NewMockEmailService creates a new mock email service
func NewMockEmailService(logger *logrus.Logger) *MockEmailService {
	return &MockEmailService{
		logger:     logger,
		sentEmails: make([]SentEmail, 0),
	}
}

// SendVerificationEmail mock implementation
func (m *MockEmailService) SendVerificationEmail(email, token, verificationURL string) error {
	m.sentEmails = append(m.sentEmails, SentEmail{
		To:     email,
		Type:   "verification",
		Token:  token,
		URL:    verificationURL,
		SentAt: time.Now(),
	})
	m.logger.WithFields(logrus.Fields{
		"email": email,
		"type":  "verification",
		"url":   verificationURL,
	}).Info("Mock: Verification email sent")
	return nil
}

// SendPasswordResetEmail mock implementation
func (m *MockEmailService) SendPasswordResetEmail(email, token, resetURL string) error {
	m.sentEmails = append(m.sentEmails, SentEmail{
		To:     email,
		Type:   "password_reset",
		Token:  token,
		URL:    resetURL,
		SentAt: time.Now(),
	})
	m.logger.WithFields(logrus.Fields{
		"email": email,
		"type":  "password_reset",
		"url":   resetURL,
	}).Info("Mock: Password reset email sent")
	return nil
}

// SendWelcomeEmail mock implementation
func (m *MockEmailService) SendWelcomeEmail(email, name string) error {
	m.sentEmails = append(m.sentEmails, SentEmail{
		To:     email,
		Type:   "welcome",
		SentAt: time.Now(),
	})
	m.logger.WithFields(logrus.Fields{
		"email": email,
		"name":  name,
		"type":  "welcome",
	}).Info("Mock: Welcome email sent")
	return nil
}

// GetSentEmails returns all sent emails (for testing)
func (m *MockEmailService) GetSentEmails() []SentEmail {
	return m.sentEmails
}

// SendOrganizationInvitationEmail mock implementation
func (m *MockEmailService) SendOrganizationInvitationEmail(email, orgName, inviterName, role, invitationURL string) error {
	m.sentEmails = append(m.sentEmails, SentEmail{
		To:          email,
		Type:        "organization_invitation",
		OrgName:     orgName,
		InviterName: inviterName,
		Role:        role,
		URL:         invitationURL,
		SentAt:      time.Now(),
	})
	m.logger.WithFields(logrus.Fields{
		"email":        email,
		"type":         "organization_invitation",
		"org_name":     orgName,
		"inviter_name": inviterName,
		"role":         role,
	}).Info("Mock: Organization invitation email sent")
	return nil
}

// SendOrganizationWelcomeEmail mock implementation
func (m *MockEmailService) SendOrganizationWelcomeEmail(email, name, orgName, role string) error {
	m.sentEmails = append(m.sentEmails, SentEmail{
		To:      email,
		Type:    "organization_welcome",
		Name:    name,
		OrgName: orgName,
		Role:    role,
		SentAt:  time.Now(),
	})
	m.logger.WithFields(logrus.Fields{
		"email":    email,
		"name":     name,
		"type":     "organization_welcome",
		"org_name": orgName,
		"role":     role,
	}).Info("Mock: Organization welcome email sent")
	return nil
}

// SendMemberRemovedEmail mock implementation
func (m *MockEmailService) SendMemberRemovedEmail(email, orgName string) error {
	m.sentEmails = append(m.sentEmails, SentEmail{
		To:      email,
		Type:    "member_removed",
		OrgName: orgName,
		SentAt:  time.Now(),
	})
	m.logger.WithFields(logrus.Fields{
		"email":    email,
		"type":     "member_removed",
		"org_name": orgName,
	}).Info("Mock: Member removed email sent")
	return nil
}

// ClearSentEmails clears the sent emails list (for testing)
func (m *MockEmailService) ClearSentEmails() {
	m.sentEmails = make([]SentEmail, 0)
}
