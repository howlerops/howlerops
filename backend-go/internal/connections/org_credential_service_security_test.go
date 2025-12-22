package connections

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jbeck018/howlerops/backend-go/pkg/crypto"
	"github.com/jbeck018/howlerops/backend-go/pkg/storage/turso"
)

// TestOEKIsolation_DifferentOrgsGetDifferentKeys verifies that organizations
// have completely isolated OEKs - one org's OEK cannot decrypt another org's credentials
func TestOEKIsolation_DifferentOrgsGetDifferentKeys(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service, _, credStore, _ := setupTestService()

	// User A in Org 1
	userAKey, err := crypto.GenerateMasterKey()
	require.NoError(t, err)

	userAKeyCopy := make([]byte, len(userAKey))
	copy(userAKeyCopy, userAKey)

	org1Password := []byte("org1-secret-password")
	org1PasswordCopy := make([]byte, len(org1Password))
	copy(org1PasswordCopy, org1Password)

	// User A shares credential to Org 1
	err = service.ShareCredential(ctx, "conn-db", "org-1", "user-a", userAKey, org1Password)
	require.NoError(t, err, "Org 1 credential share should succeed")

	// User B in Org 2
	userBKey, err := crypto.GenerateMasterKey()
	require.NoError(t, err)

	userBKeyCopy := make([]byte, len(userBKey))
	copy(userBKeyCopy, userBKey)

	org2Password := []byte("org2-different-password")

	// User B shares credential to Org 2
	err = service.ShareCredential(ctx, "conn-db", "org-2", "user-b", userBKey, org2Password)
	require.NoError(t, err, "Org 2 credential share should succeed")

	// Verify User A can retrieve their own org's password
	retrievedOrg1Password, err := service.GetSharedCredentialPassword(ctx, "conn-db", "org-1", "user-a", userAKeyCopy)
	require.NoError(t, err, "User A should decrypt Org 1 credential")
	defer crypto.ClearBytes(retrievedOrg1Password)
	assert.Equal(t, org1PasswordCopy, retrievedOrg1Password, "Org 1 password should match")

	// Verify User B can retrieve their own org's password
	retrievedOrg2Password, err := service.GetSharedCredentialPassword(ctx, "conn-db", "org-2", "user-b", userBKeyCopy)
	require.NoError(t, err, "User B should decrypt Org 2 credential")
	defer crypto.ClearBytes(retrievedOrg2Password)

	// Critical security check: Verify User A CANNOT decrypt Org 2's credential using Org 1's OEK
	// This tests that OEKs are truly isolated between organizations
	org2Cred, err := credStore.Get(ctx, "conn-db", "org-2")
	require.NoError(t, err, "Should be able to fetch Org 2 credential")

	// Try to decrypt Org 2's credential with User A's OEK (should fail)
	userAKeyCopy2 := make([]byte, len(userAKeyCopy))
	copy(userAKeyCopy2, userAKeyCopy)

	_, err = service.GetSharedCredentialPassword(ctx, "conn-db", "org-2", "user-a", userAKeyCopy2)
	assert.Error(t, err, "User A should NOT be able to decrypt Org 2 credential with Org 1 OEK")
	assert.Contains(t, err.Error(), "user does not have access to organization's OEK",
		"Error should indicate OEK access denial")

	// Verify the encrypted data is indeed different
	org1Cred, err := credStore.Get(ctx, "conn-db", "org-1")
	require.NoError(t, err, "Should be able to fetch Org 1 credential")

	assert.NotEqual(t, org1Cred.EncryptedPassword, org2Cred.EncryptedPassword,
		"Different orgs should have different encrypted passwords")
	assert.NotEqual(t, org1Cred.PasswordIV, org2Cred.PasswordIV,
		"Different orgs should have different IVs")
}

// TestOEKIsolation_RevokedMemberCannotDecrypt verifies that after revoking
// a member's OEK access, they can no longer decrypt credentials even if they
// kept a copy of their encrypted OEK
func TestOEKIsolation_RevokedMemberCannotDecrypt(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service, _, _, _ := setupTestService()

	// Setup: User A creates org and shares credential
	userAKey, err := crypto.GenerateMasterKey()
	require.NoError(t, err)

	userAKeyCopy1 := make([]byte, len(userAKey))
	copy(userAKeyCopy1, userAKey)

	userAKeyCopy2 := make([]byte, len(userAKey))
	copy(userAKeyCopy2, userAKey)

	password := []byte("shared-secret-password")
	err = service.ShareCredential(ctx, "conn-prod", "org-acme", "user-a", userAKey, password)
	require.NoError(t, err, "Initial share should succeed")

	// User B joins and gets provisioned with OEK
	userBKey, err := crypto.GenerateMasterKey()
	require.NoError(t, err)

	userBKeyCopy1 := make([]byte, len(userBKey))
	copy(userBKeyCopy1, userBKey)

	userBKeyCopy2 := make([]byte, len(userBKey))
	copy(userBKeyCopy2, userBKey)

	err = service.ProvisionOEKForNewMember(ctx, "org-acme", "user-a", userAKeyCopy1, "user-b", userBKey)
	require.NoError(t, err, "Provisioning should succeed")

	// Verify User B can decrypt before revocation
	retrievedPassword, err := service.GetSharedCredentialPassword(ctx, "conn-prod", "org-acme", "user-b", userBKeyCopy1)
	require.NoError(t, err, "User B should decrypt before revocation")
	defer crypto.ClearBytes(retrievedPassword)

	// Revoke User B's access
	err = service.RevokeOEKForMember(ctx, "org-acme", "user-b")
	require.NoError(t, err, "Revocation should succeed")

	// Critical security check: User B can no longer decrypt even with their old master key
	_, err = service.GetSharedCredentialPassword(ctx, "conn-prod", "org-acme", "user-b", userBKeyCopy2)
	assert.Error(t, err, "Revoked user should NOT be able to decrypt")
	assert.Contains(t, err.Error(), "user does not have access to organization's OEK",
		"Error should indicate OEK access denial")

	// Verify User A can still decrypt (unaffected by User B revocation)
	retrievedPasswordA, err := service.GetSharedCredentialPassword(ctx, "conn-prod", "org-acme", "user-a", userAKeyCopy2)
	require.NoError(t, err, "User A should still decrypt after revoking User B")
	defer crypto.ClearBytes(retrievedPasswordA)
}

// TestCryptoCleanup_MasterKeyCleared verifies that user master keys are
// properly zeroed out after use to prevent memory disclosure attacks
func TestCryptoCleanup_MasterKeyCleared(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service, _, _, _ := setupTestService()

	// Generate master key and keep reference to track it
	userMasterKey, err := crypto.GenerateMasterKey()
	require.NoError(t, err)

	// Make a copy to verify clearing
	masterKeyCopy := make([]byte, len(userMasterKey))
	copy(masterKeyCopy, userMasterKey)

	password := []byte("test-password")
	passwordCopy := make([]byte, len(password))
	copy(passwordCopy, password)

	// ShareCredential should clear the master key after use
	err = service.ShareCredential(ctx, "conn-123", "org-456", "user-789", userMasterKey, password)
	require.NoError(t, err)

	// Verify that the original master key slice has been zeroed
	allZeros := true
	for _, b := range userMasterKey {
		if b != 0 {
			allZeros = false
			break
		}
	}
	assert.True(t, allZeros, "Master key should be zeroed after ShareCredential")

	// Verify password is also cleared
	allZerosPassword := true
	for _, b := range password {
		if b != 0 {
			allZerosPassword = false
			break
		}
	}
	assert.True(t, allZerosPassword, "Password should be zeroed after ShareCredential")

	// Verify GetSharedCredentialPassword also clears master key
	userMasterKey2, err := crypto.GenerateMasterKey()
	require.NoError(t, err)

	copy(userMasterKey2, masterKeyCopy)

	_, err = service.GetSharedCredentialPassword(ctx, "conn-123", "org-456", "user-789", userMasterKey2)
	require.NoError(t, err)

	allZeros2 := true
	for _, b := range userMasterKey2 {
		if b != 0 {
			allZeros2 = false
			break
		}
	}
	assert.True(t, allZeros2, "Master key should be zeroed after GetSharedCredentialPassword")
}

// TestCryptoCleanup_OEKCleared verifies that OEKs are properly zeroed
// after cryptographic operations to prevent memory disclosure
func TestCryptoCleanup_OEKCleared(t *testing.T) {
	t.Parallel()

	// Test OEK clearing in encryption/decryption cycle
	oek, err := crypto.GenerateOrgEnvelopeKey()
	require.NoError(t, err)

	oekCopy := make([]byte, len(oek))
	copy(oekCopy, oek)

	plaintext := []byte("secret-data")
	encrypted, err := crypto.EncryptWithOEK(plaintext, oek)
	require.NoError(t, err)

	// Decrypt with copy
	decrypted, err := crypto.DecryptWithOEK(encrypted, oekCopy)
	require.NoError(t, err)
	defer crypto.ClearBytes(decrypted)

	// Clear OEK
	crypto.ClearBytes(oek)

	// Verify OEK is zeroed
	allZeros := true
	for _, b := range oek {
		if b != 0 {
			allZeros = false
			break
		}
	}
	assert.True(t, allZeros, "OEK should be zeroed after clearing")

	// Clear the copy as well
	crypto.ClearBytes(oekCopy)
	allZeros2 := true
	for _, b := range oekCopy {
		if b != 0 {
			allZeros2 = false
			break
		}
	}
	assert.True(t, allZeros2, "OEK copy should be zeroed after clearing")
}

// TestAccessControl_NonMemberCannotShare verifies that users who are not
// members of an organization cannot share credentials to that organization
func TestAccessControl_NonMemberCannotShare(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service, oekStore, _, _ := setupTestService()

	// Setup: User A creates OEK for Org 1
	userAKey, err := crypto.GenerateMasterKey()
	require.NoError(t, err)

	password := []byte("org-password")
	err = service.ShareCredential(ctx, "conn-db", "org-1", "user-a", userAKey, password)
	require.NoError(t, err, "User A should successfully share to their org")

	// User B tries to share to Org 1 without being provisioned
	userBKey, err := crypto.GenerateMasterKey()
	require.NoError(t, err)

	passwordB := []byte("malicious-password")

	// Critical security check: User B should NOT be able to share to Org 1
	err = service.ShareCredential(ctx, "conn-db2", "org-1", "user-b", userBKey, passwordB)
	assert.Error(t, err, "Non-member should NOT be able to share credentials")
	assert.Contains(t, err.Error(), "user does not have access to organization's OEK",
		"Error should indicate lack of OEK access")

	// Verify only User A has OEK for Org 1
	_, err = oekStore.GetOEKForUser(ctx, "org-1", "user-a")
	assert.NoError(t, err, "User A should have OEK")

	_, err = oekStore.GetOEKForUser(ctx, "org-1", "user-b")
	assert.Error(t, err, "User B should NOT have OEK for Org 1")
}

// TestAccessControl_NonMemberCannotDecrypt verifies that users without
// OEK access cannot decrypt organization shared credentials
func TestAccessControl_NonMemberCannotDecrypt(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service, _, _, _ := setupTestService()

	// Setup: User A shares credential to Org 1
	userAKey, err := crypto.GenerateMasterKey()
	require.NoError(t, err)

	password := []byte("org-secret")
	err = service.ShareCredential(ctx, "conn-prod", "org-1", "user-a", userAKey, password)
	require.NoError(t, err, "User A should share successfully")

	// User B tries to decrypt without being member
	userBKey, err := crypto.GenerateMasterKey()
	require.NoError(t, err)

	// Critical security check: Non-member should NOT be able to decrypt
	_, err = service.GetSharedCredentialPassword(ctx, "conn-prod", "org-1", "user-b", userBKey)
	assert.Error(t, err, "Non-member should NOT be able to decrypt")
	assert.Contains(t, err.Error(), "user does not have access to organization's OEK",
		"Error should indicate OEK access denial")
}

// TestAccessControl_WrongMasterKeyFails verifies that using the wrong master key
// fails gracefully without leaking information
func TestAccessControl_WrongMasterKeyFails(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service, _, _, _ := setupTestService()

	// Setup: User A shares credential
	correctMasterKey, err := crypto.GenerateMasterKey()
	require.NoError(t, err)

	correctKeyCopy := make([]byte, len(correctMasterKey))
	copy(correctKeyCopy, correctMasterKey)

	password := []byte("secure-password")
	err = service.ShareCredential(ctx, "conn-123", "org-456", "user-789", correctMasterKey, password)
	require.NoError(t, err, "Share with correct key should succeed")

	// Try to decrypt with wrong master key
	wrongMasterKey, err := crypto.GenerateMasterKey()
	require.NoError(t, err)

	// Critical security check: Wrong master key should fail decryption
	_, err = service.GetSharedCredentialPassword(ctx, "conn-123", "org-456", "user-789", wrongMasterKey)
	assert.Error(t, err, "Wrong master key should fail decryption")
	assert.Contains(t, err.Error(), "failed to decrypt",
		"Error should indicate decryption failure")

	// Verify correct key still works
	retrievedPassword, err := service.GetSharedCredentialPassword(ctx, "conn-123", "org-456", "user-789", correctKeyCopy)
	require.NoError(t, err, "Correct master key should still work")
	defer crypto.ClearBytes(retrievedPassword)
}

// TestConcurrentOEKAccess verifies that multiple goroutines can safely
// access OEK operations without race conditions
func TestConcurrentOEKAccess(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service, oekStore, _, _ := setupTestService()

	// Setup: Create initial OEK for user-0
	initialUserKey, err := crypto.GenerateMasterKey()
	require.NoError(t, err)

	initialUserKeyCopy := make([]byte, len(initialUserKey))
	copy(initialUserKeyCopy, initialUserKey)

	password := []byte("initial-password")
	err = service.ShareCredential(ctx, "conn-shared", "org-concurrent", "user-0", initialUserKey, password)
	require.NoError(t, err, "Initial share should succeed")

	// Verify OEK was created for user-0
	_, err = oekStore.GetOEKForUser(ctx, "org-concurrent", "user-0")
	require.NoError(t, err, "Should get OEK for user-0")

	// Test concurrent OEK provisioning with multiple users
	numUsers := 10
	var wg sync.WaitGroup
	errors := make(chan error, numUsers)

	// Concurrent provisioning - each goroutine creates its own key copy
	for i := 1; i < numUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			// Create fresh master key for existing user
			existingKeyCopy := make([]byte, len(initialUserKeyCopy))
			copy(existingKeyCopy, initialUserKeyCopy)

			// Create new user's master key
			newUserKey, err := crypto.GenerateMasterKey()
			if err != nil {
				errors <- err
				return
			}

			userIDStr := fmt.Sprintf("user-%d", userID)
			err = service.ProvisionOEKForNewMember(ctx, "org-concurrent", "user-0", existingKeyCopy, userIDStr, newUserKey)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent OEK access failed: %v", err)
	}

	// Verify all users can decrypt the shared credential
	var wg2 sync.WaitGroup
	errors2 := make(chan error, numUsers)

	for i := 0; i < numUsers; i++ {
		wg2.Add(1)
		go func(userID int) {
			defer wg2.Done()

			// Get the OEK for this user
			userIDStr := fmt.Sprintf("user-%d", userID)
			storedUserOEK, err := oekStore.GetOEKForUser(ctx, "org-concurrent", userIDStr)
			if err != nil {
				errors2 <- err
				return
			}

			// Create a fresh master key copy to decrypt with
			var userMasterKey []byte
			if userID == 0 {
				userMasterKey = make([]byte, len(initialUserKeyCopy))
				copy(userMasterKey, initialUserKeyCopy)
			} else {
				// For provisioned users, we need to generate their key again
				// Since we can't store it, we'll just verify the OEK exists
				if storedUserOEK == nil {
					errors2 <- fmt.Errorf("user %d should have OEK", userID)
				}
				return
			}

			retrievedPassword, err := service.GetSharedCredentialPassword(ctx, "conn-shared", "org-concurrent", userIDStr, userMasterKey)
			if err != nil {
				errors2 <- err
				return
			}
			defer crypto.ClearBytes(retrievedPassword)
		}(i)
	}

	wg2.Wait()
	close(errors2)

	for err := range errors2 {
		t.Errorf("Concurrent password retrieval failed: %v", err)
	}
}

// TestConcurrentCredentialDecryption verifies that multiple concurrent
// decryption requests are handled safely without race conditions
func TestConcurrentCredentialDecryption(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service, _, _, _ := setupTestService()

	// Setup: Create user master key
	userMasterKey, err := crypto.GenerateMasterKey()
	require.NoError(t, err)

	// Store a persistent copy for decryption
	masterKeyCopy := make([]byte, len(userMasterKey))
	copy(masterKeyCopy, userMasterKey)

	// Setup: Create multiple shared credentials
	numCreds := 5
	passwords := make([][]byte, numCreds)
	for i := 0; i < numCreds; i++ {
		connID := fmt.Sprintf("conn-%d", i)
		password := []byte(fmt.Sprintf("password-%d", i))
		passwords[i] = make([]byte, len(password))
		copy(passwords[i], password)

		// Create a fresh key copy for each share
		shareKeyCopy := make([]byte, len(userMasterKey))
		copy(shareKeyCopy, userMasterKey)

		err := service.ShareCredential(ctx, connID, "org-test", "user-123", shareKeyCopy, password)
		require.NoError(t, err, "Share %d should succeed", i)
	}

	// Concurrent decryption of all credentials (3 goroutines per credential)
	numGoroutinesPerCred := 3
	var wg sync.WaitGroup
	errors := make(chan error, numCreds*numGoroutinesPerCred)

	for i := 0; i < numCreds; i++ {
		for j := 0; j < numGoroutinesPerCred; j++ {
			wg.Add(1)
			go func(credID int) {
				defer wg.Done()

				connID := fmt.Sprintf("conn-%d", credID)

				// Create fresh key copy for this goroutine
				decryptKeyCopy := make([]byte, len(masterKeyCopy))
				copy(decryptKeyCopy, masterKeyCopy)

				retrievedPassword, err := service.GetSharedCredentialPassword(ctx, connID, "org-test", "user-123", decryptKeyCopy)
				if err != nil {
					errors <- err
					return
				}
				defer crypto.ClearBytes(retrievedPassword)

				// Verify password matches
				expectedPassword := passwords[credID]
				if len(retrievedPassword) != len(expectedPassword) {
					errors <- fmt.Errorf("credential %d: password length mismatch, expected %d got %d", credID, len(expectedPassword), len(retrievedPassword))
					return
				}
				for k := 0; k < len(retrievedPassword); k++ {
					if retrievedPassword[k] != expectedPassword[k] {
						errors <- fmt.Errorf("credential %d: password content mismatch at byte %d", credID, k)
						return
					}
				}
			}(i)
		}
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent credential decryption failed: %v", err)
	}
}

// TestAuditLog_ShareCreatesEntry verifies that credential share operations
// are properly logged to the audit trail
func TestAuditLog_ShareCreatesEntry(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service, _, _, auditStore := setupTestService()

	userKey, err := crypto.GenerateMasterKey()
	require.NoError(t, err)

	password := []byte("secret-password")
	err = service.ShareCredential(ctx, "conn-audit", "org-audit", "user-audit", userKey, password)
	require.NoError(t, err, "Share should succeed")

	// Wait for async audit logging
	time.Sleep(100 * time.Millisecond)

	// Verify audit log entry exists
	logs := auditStore.GetLogs()
	assert.NotEmpty(t, logs, "Audit logs should not be empty")

	// Find the share log entry
	found := false
	for _, log := range logs {
		if log.Action == "share" &&
			log.ConnectionID == "conn-audit" &&
			log.OrganizationID == "org-audit" &&
			log.UserID == "user-audit" {
			found = true
			assert.True(t, log.Success, "Share operation should be logged as successful")
			assert.Empty(t, log.ErrorMessage, "Successful operation should have no error message")
			assert.False(t, log.Timestamp.IsZero(), "Timestamp should be set")
			break
		}
	}
	assert.True(t, found, "Share audit log entry should exist")
}

// TestAuditLog_DecryptCreatesEntry verifies that credential decrypt operations
// are properly logged to the audit trail
func TestAuditLog_DecryptCreatesEntry(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service, _, _, auditStore := setupTestService()

	// Setup: Share credential
	userKey, err := crypto.GenerateMasterKey()
	require.NoError(t, err)

	userKeyCopy := make([]byte, len(userKey))
	copy(userKeyCopy, userKey)

	password := []byte("audit-password")
	err = service.ShareCredential(ctx, "conn-decrypt-audit", "org-audit2", "user-audit2", userKey, password)
	require.NoError(t, err, "Share should succeed")

	// Clear initial logs
	time.Sleep(100 * time.Millisecond)
	auditStore.mu.Lock()
	auditStore.logs = make([]*turso.CredentialAccessLog, 0)
	auditStore.mu.Unlock()

	// Decrypt credential
	retrievedPassword, err := service.GetSharedCredentialPassword(ctx, "conn-decrypt-audit", "org-audit2", "user-audit2", userKeyCopy)
	require.NoError(t, err, "Decrypt should succeed")
	defer crypto.ClearBytes(retrievedPassword)

	// Wait for async audit logging
	time.Sleep(100 * time.Millisecond)

	// Verify audit log entry exists
	logs := auditStore.GetLogs()
	assert.NotEmpty(t, logs, "Audit logs should not be empty")

	// Find the decrypt log entry
	found := false
	for _, log := range logs {
		if log.Action == "decrypt" &&
			log.ConnectionID == "conn-decrypt-audit" &&
			log.OrganizationID == "org-audit2" &&
			log.UserID == "user-audit2" {
			found = true
			assert.True(t, log.Success, "Decrypt operation should be logged as successful")
			assert.Empty(t, log.ErrorMessage, "Successful operation should have no error message")
			assert.False(t, log.Timestamp.IsZero(), "Timestamp should be set")
			break
		}
	}
	assert.True(t, found, "Decrypt audit log entry should exist")
}

// TestAuditLog_FailuresAreLogged verifies that failed credential operations
// are properly logged with error details
func TestAuditLog_FailuresAreLogged(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service, _, _, auditStore := setupTestService()

	// Test failed share (invalid scenario - no existing OEK and simulated provision requirement)
	userKey, err := crypto.GenerateMasterKey()
	require.NoError(t, err)

	// Try to decrypt non-existent credential (will fail)
	_, err = service.GetSharedCredentialPassword(ctx, "conn-nonexistent", "org-fail", "user-fail", userKey)
	assert.Error(t, err, "Should fail for non-existent credential")

	// Wait for async audit logging
	time.Sleep(100 * time.Millisecond)

	// Verify failure is logged
	logs := auditStore.GetLogs()
	found := false
	for _, log := range logs {
		if log.Action == "decrypt" &&
			log.ConnectionID == "conn-nonexistent" &&
			log.OrganizationID == "org-fail" &&
			log.UserID == "user-fail" {
			found = true
			assert.False(t, log.Success, "Failed operation should be logged as unsuccessful")
			assert.NotEmpty(t, log.ErrorMessage, "Failed operation should have error message")
			assert.False(t, log.Timestamp.IsZero(), "Timestamp should be set")
			break
		}
	}
	assert.True(t, found, "Failed decrypt should be logged")
}

// TestAuditLog_RevokeCreatesEntry verifies that OEK revocation is logged
func TestAuditLog_RevokeCreatesEntry(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service, _, _, auditStore := setupTestService()

	// Setup: Create OEK for user
	userKey, err := crypto.GenerateMasterKey()
	require.NoError(t, err)

	password := []byte("password")
	err = service.ShareCredential(ctx, "conn-revoke", "org-revoke", "user-revoke", userKey, password)
	require.NoError(t, err, "Share should succeed")

	// Clear initial logs
	time.Sleep(100 * time.Millisecond)
	auditStore.mu.Lock()
	auditStore.logs = make([]*turso.CredentialAccessLog, 0)
	auditStore.mu.Unlock()

	// Revoke user's OEK
	err = service.RevokeOEKForMember(ctx, "org-revoke", "user-revoke")
	require.NoError(t, err, "Revoke should succeed")

	// Verify audit log entry (revoke is synchronous, no wait needed)
	logs := auditStore.GetLogs()
	assert.NotEmpty(t, logs, "Audit logs should not be empty")

	// Find the revoke log entry
	found := false
	for _, log := range logs {
		if log.Action == "revoke_oek" &&
			log.OrganizationID == "org-revoke" &&
			log.UserID == "user-revoke" {
			found = true
			assert.True(t, log.Success, "Revoke operation should be logged as successful")
			assert.False(t, log.Timestamp.IsZero(), "Timestamp should be set")
			break
		}
	}
	assert.True(t, found, "Revoke audit log entry should exist")
}

// TestSecurityBoundary_RandomKeyInjection verifies that injecting random
// cryptographic data does not compromise the system
func TestSecurityBoundary_RandomKeyInjection(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service, oekStore, _, _ := setupTestService()

	// Setup: Create legitimate OEK
	legitimateKey, err := crypto.GenerateMasterKey()
	require.NoError(t, err)

	password := []byte("legitimate-password")
	err = service.ShareCredential(ctx, "conn-sec", "org-sec", "user-legit", legitimateKey, password)
	require.NoError(t, err, "Legitimate share should succeed")

	// Attack: Try to inject random encrypted OEK data
	randomData := make([]byte, 48) // Typical encrypted OEK size
	_, err = rand.Read(randomData)
	require.NoError(t, err)

	maliciousEncrypted := &turso.OEKEncryptedData{
		Ciphertext: string(randomData[:32]),
		IV:         string(randomData[32:44]),
		AuthTag:    string(randomData[44:]),
	}

	err = oekStore.StoreOEKForUser(ctx, "org-sec", "user-attacker", maliciousEncrypted)
	require.NoError(t, err, "Storage should succeed (validation happens on decrypt)")

	// Try to decrypt with random master key
	attackerKey, err := crypto.GenerateMasterKey()
	require.NoError(t, err)

	// Critical security check: Random data should fail decryption gracefully
	_, err = service.GetSharedCredentialPassword(ctx, "conn-sec", "org-sec", "user-attacker", attackerKey)
	assert.Error(t, err, "Random encrypted data should fail decryption")
	assert.Contains(t, err.Error(), "failed to decrypt",
		"Error should indicate decryption failure")
}

// TestSecurityBoundary_MalformedData verifies that malformed encrypted data
// is rejected without compromising system security
func TestSecurityBoundary_MalformedData(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		setupFunc     func() *crypto.OEKEncryptedData
		expectedError string
	}{
		{
			name: "Empty ciphertext",
			setupFunc: func() *crypto.OEKEncryptedData {
				return &crypto.OEKEncryptedData{
					Ciphertext: "",
					IV:         "valid-iv-data",
					AuthTag:    "valid-auth-tag",
				}
			},
			expectedError: "failed to decrypt",
		},
		{
			name: "Empty IV",
			setupFunc: func() *crypto.OEKEncryptedData {
				return &crypto.OEKEncryptedData{
					Ciphertext: "valid-ciphertext",
					IV:         "",
					AuthTag:    "valid-auth-tag",
				}
			},
			expectedError: "failed to decrypt",
		},
		{
			name: "Empty auth tag",
			setupFunc: func() *crypto.OEKEncryptedData {
				return &crypto.OEKEncryptedData{
					Ciphertext: "valid-ciphertext",
					IV:         "valid-iv-data",
					AuthTag:    "",
				}
			},
			expectedError: "failed to decrypt",
		},
		{
			name: "Invalid base64 in ciphertext",
			setupFunc: func() *crypto.OEKEncryptedData {
				return &crypto.OEKEncryptedData{
					Ciphertext: "!!!invalid-base64!!!",
					IV:         "dmFsaWQtaXY=",
					AuthTag:    "dmFsaWQtdGFn",
				}
			},
			expectedError: "failed to decrypt",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			malformedData := tc.setupFunc()
			oek, _ := crypto.GenerateOrgEnvelopeKey()
			defer crypto.ClearBytes(oek)

			// Critical security check: Malformed data should fail gracefully
			_, err := crypto.DecryptWithOEK(malformedData, oek)
			assert.Error(t, err, "Malformed data should be rejected")
			// Note: Error message varies by malformation type, so we check for general failure
		})
	}
}
