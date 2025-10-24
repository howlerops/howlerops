package turso

import (
	"github.com/sql-studio/backend-go/internal/auth"
)

// This file verifies at compile time that our stores implement the correct interfaces
// If this file compiles, all interfaces are correctly implemented

// Verify TursoUserStore implements auth.UserStore
var _ auth.UserStore = (*TursoUserStore)(nil)

// Verify TursoSessionStore implements auth.SessionStore
var _ auth.SessionStore = (*TursoSessionStore)(nil)

// Verify TursoLoginAttemptStore implements auth.LoginAttemptStore
var _ auth.LoginAttemptStore = (*TursoLoginAttemptStore)(nil)

// Compile-time interface verification complete ✅
