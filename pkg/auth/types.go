package auth

import "time"

// OAuthUser represents authenticated OAuth user data
type OAuthUser struct {
	ID          string    `json:"id"`
	Login       string    `json:"login"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	AvatarURL   string    `json:"avatarUrl,omitempty"`
	Provider    string    `json:"provider"` // "github" or "google"
	AccessToken string    `json:"-"`        // Never exposed to frontend
	ExpiresAt   time.Time `json:"expiresAt,omitempty"`
}

// PendingToken represents a pending OAuth2 token exchange
type PendingToken struct {
	State         string
	CodeVerifier  string // PKCE verifier
	CodeChallenge string // PKCE challenge
	Provider      string
	CreatedAt     time.Time
	ExpiresAt     time.Time
}

// GitHubUser represents GitHub user data from API
type GitHubUser struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

// GoogleUser represents Google user data from API
type GoogleUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
}
