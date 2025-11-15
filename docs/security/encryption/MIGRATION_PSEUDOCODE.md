# Password Migration Pseudocode

This document provides detailed pseudocode for all key migration functions to guide implementation.

---

## 1. Core Password Manager (Go)

### Dual-Read with Fallback

```go
func (pm *PasswordManager) GetPassword(
	ctx context.Context,
	userID string,
	connectionID string,
	masterKey []byte,
) (string, error) {

	// STEP 1: Try encrypted credentials first (new system)
	if masterKey is not empty {
		encryptedCred = fetch from encrypted_credentials table where user_id = userID AND connection_id = connectionID

		if encryptedCred exists {
			try {
				password = decrypt(encryptedCred, masterKey)
				log("Password retrieved from encrypted storage")
				return password
			} catch DecryptionError {
				log("Decryption failed, falling back to keychain")
			}
		}
	}

	// STEP 2: Fall back to keychain (legacy system)
	try {
		password = keychain.get(connectionID)
		log("Password retrieved from keychain")

		// STEP 3: Opportunistic migration (async)
		if masterKey is not empty {
			go migrate_in_background(userID, connectionID, password, masterKey)
		}

		return password
	} catch KeychainNotFoundError {
		return error("Password not found in encrypted storage or keychain")
	} catch KeychainError as err {
		return error("Failed to retrieve password: " + err)
	}
}
```

---

### Store Password (Dual-Write)

```go
func (pm *PasswordManager) StorePassword(
	ctx context.Context,
	userID string,
	connectionID string,
	password string,
	masterKey []byte,
) error {

	encrypted_success = false
	keychain_success = false

	// STEP 1: Try encrypted storage first
	if masterKey is not empty {
		try {
			encryptedCred = encrypt(password, masterKey)
			save to encrypted_credentials table
			mark_connection_as_migrated(connectionID)
			log("Password stored in encrypted DB")
			encrypted_success = true
		} catch EncryptionError as err {
			log("Encrypted storage failed: " + err)
		}
	}

	// STEP 2: Always store in keychain as backup (during transition)
	try {
		keychain.set(connectionID, password)
		log("Password stored in keychain")
		keychain_success = true
	} catch KeychainError as err {
		log("Keychain storage failed: " + err)
	}

	// STEP 3: Evaluate results
	if encrypted_success OR keychain_success {
		return success
	} else {
		return error("Failed to store in both encrypted DB and keychain")
	}
}
```

---

### Delete Password (Dual-Delete)

```go
func (pm *PasswordManager) DeletePassword(
	ctx context.Context,
	userID string,
	connectionID string,
) error {

	// Delete from both locations (ignore errors)
	try {
		delete from encrypted_credentials where user_id = userID AND connection_id = connectionID
	} catch {
		// Ignore - best effort
	}

	try {
		keychain.delete(connectionID)
	} catch {
		// Ignore - best effort
	}

	return success
}
```

---

### Opportunistic Migration (Background)

```go
func (pm *PasswordManager) opportunistic_migration(
	ctx context.Context,
	userID string,
	connectionID string,
	password string,
	masterKey []byte,
) {
	log("Starting opportunistic migration for connection " + connectionID)

	// STEP 1: Encrypt password
	try {
		encryptedCred = encrypt(password, masterKey)
	} catch EncryptionError as err {
		log("Migration failed: encryption error - " + err)
		log_migration(connectionID, "automatic", "failed", err.message)
		return
	}

	// STEP 2: Store in encrypted DB
	try {
		insert into encrypted_credentials (
			id, user_id, connection_id, encrypted_password, password_iv, password_auth_tag
		) values (
			generate_uuid(), userID, connectionID, encryptedCred.ciphertext, encryptedCred.iv, encryptedCred.authTag
		)
	} catch StorageError as err {
		log("Migration failed: storage error - " + err)
		log_migration(connectionID, "automatic", "failed", err.message)
		return
	}

	// STEP 3: Mark as migrated
	try {
		update connection_templates
		set password_migration_status = 'migrated',
		    password_migration_metadata = json_object(
		        'migratedAt', current_timestamp(),
		        'migratedFrom', 'keychain',
		        'migrationType', 'automatic'
		    )
		where id = connectionID
	} catch {
		log("Failed to update migration status (non-critical)")
	}

	// STEP 4: Log success
	log_migration(connectionID, "automatic", "success", "")
	log("Opportunistic migration completed for connection " + connectionID)
}
```

---

## 2. Bulk Migration (Go)

```go
func (pm *PasswordManager) MigrateBulkPasswords(
	ctx context.Context,
	userID string,
	masterKey []byte,
) (*MigrationResult, error) {

	result = new MigrationResult()

	// STEP 1: Find all unmigrated connections
	connections = query(`
		SELECT id, name FROM connection_templates
		WHERE user_id = ? AND password_migration_status = 'not_migrated'
	`, userID)

	result.total = len(connections)

	// STEP 2: Migrate each connection
	for each connection in connections {
		connectionID = connection.id

		// Try to get password from keychain
		try {
			password = keychain.get(connectionID)
		} catch KeychainNotFoundError {
			// No password stored - mark as no_password
			update connection_templates set password_migration_status = 'no_password' where id = connectionID
			result.skipped++
			continue
		} catch KeychainError as err {
			result.failed++
			result.errors[connectionID] = err.message
			log_migration(connectionID, "manual", "failed", err.message)
			continue
		}

		// Encrypt password
		try {
			encryptedCred = encrypt(password, masterKey)
		} catch EncryptionError as err {
			result.failed++
			result.errors[connectionID] = "Encryption failed: " + err.message
			log_migration(connectionID, "manual", "failed", err.message)
			continue
		}

		// Store in encrypted DB
		try {
			insert into encrypted_credentials (
				id, user_id, connection_id, encrypted_password, password_iv, password_auth_tag
			) values (
				generate_uuid(), userID, connectionID, encryptedCred.ciphertext, encryptedCred.iv, encryptedCred.authTag
			)
		} catch StorageError as err {
			result.failed++
			result.errors[connectionID] = "Storage failed: " + err.message
			log_migration(connectionID, "manual", "failed", err.message)
			continue
		}

		// Mark as migrated
		mark_connection_as_migrated(connectionID)
		log_migration(connectionID, "manual", "success", "")
		result.success++
	}

	return result
}
```

---

## 3. Migration on First Login (Frontend)

```typescript
// On successful login
async function handleLoginSuccess(response: LoginResponse) {
	// Store auth tokens
	authStore.setTokens(response.accessToken, response.refreshToken)
	authStore.setUser(response.user)

	// Cache master key in memory
	const masterKey = await importMasterKeyFromBase64(response.masterKey)
	authStore.setMasterKey(masterKey)

	// Check if user has unmigrated connections
	const migrationStatus = await api.getMigrationStatus()

	if (migrationStatus.unmigratedCount > 0) {
		// Show migration prompt
		showMigrationBanner({
			total: migrationStatus.totalConnections,
			unmigrated: migrationStatus.unmigratedCount,
			migrated: migrationStatus.migratedCount,
			onMigrate: async () => {
				await startBulkMigration()
			},
			onDismiss: () => {
				// Remind in 7 days
				localStorage.setItem('migration_reminder_dismissed', Date.now().toString())
			}
		})
	}
}
```

---

## 4. Migration Progress UI (React)

```tsx
function MigrationProgressModal({ userID, masterKey }: Props) {
	const [progress, setProgress] = useState({
		total: 0,
		completed: 0,
		failed: 0,
		current: null as string | null,
		errors: {} as Record<string, string>
	})

	async function startMigration() {
		// Get list of unmigrated connections
		const connections = await api.getUnmigratedConnections(userID)
		setProgress(prev => ({ ...prev, total: connections.length }))

		// Migrate each connection
		for (const conn of connections) {
			setProgress(prev => ({ ...prev, current: conn.name }))

			try {
				// Read from keychain
				const password = await api.getPasswordFromKeychain(conn.id)

				// Encrypt with master key
				const encrypted = await encryptPassword(password, masterKey)

				// Store in encrypted DB
				await api.storeEncryptedPassword(userID, conn.id, encrypted)

				// Mark as migrated
				await api.markAsMigrated(conn.id)

				setProgress(prev => ({ ...prev, completed: prev.completed + 1 }))
			} catch (err) {
				setProgress(prev => ({
					...prev,
					failed: prev.failed + 1,
					errors: { ...prev.errors, [conn.id]: err.message }
				}))
			}
		}

		setProgress(prev => ({ ...prev, current: null }))
	}

	const percentComplete = progress.total > 0
		? Math.round((progress.completed / progress.total) * 100)
		: 0

	return (
		<Modal>
			<h2>Password Migration Progress</h2>

			<ProgressBar value={percentComplete} max={100} />
			<p>{percentComplete}% ({progress.completed} of {progress.total} connections)</p>

			{progress.current && (
				<p>Migrating: {progress.current}...</p>
			)}

			{progress.failed > 0 && (
				<Alert variant="warning">
					{progress.failed} connections failed to migrate. You can retry later.
				</Alert>
			)}

			{progress.completed === progress.total && (
				<Alert variant="success">
					Migration complete! Your passwords are now synced to the cloud.
				</Alert>
			)}

			<Button onClick={startMigration} disabled={progress.current !== null}>
				{progress.current ? 'Migrating...' : 'Start Migration'}
			</Button>
		</Modal>
	)
}
```

---

## 5. Connection Form (Save with Encryption)

```typescript
async function saveConnection(formData: ConnectionFormData) {
	const masterKey = authStore.getMasterKey()

	// Create connection metadata (without password)
	const connection = {
		name: formData.name,
		type: formData.type,
		host: formData.host,
		port: formData.port,
		database: formData.database,
		username: formData.username,
		// password intentionally omitted
	}

	// Save connection
	const savedConnection = await api.saveConnection(connection)

	// If password provided, encrypt and store separately
	if (formData.password && masterKey) {
		try {
			const encrypted = await encryptPassword(formData.password, masterKey)

			await api.storeEncryptedPassword(
				authStore.user.id,
				savedConnection.id,
				encrypted
			)

			console.log('Password encrypted and stored in cloud')
		} catch (err) {
			// Fall back to keychain
			console.warn('Encrypted storage failed, using keychain:', err)
			await api.storePasswordInKeychain(savedConnection.id, formData.password)
		}
	}

	return savedConnection
}
```

---

## 6. Connection Use (Retrieve and Decrypt)

```typescript
async function openConnection(connectionID: string) {
	const userID = authStore.user.id
	const masterKey = authStore.getMasterKey()

	// Get connection metadata
	const connection = await api.getConnection(connectionID)

	// Try to get password from encrypted DB first
	if (masterKey) {
		try {
			const encryptedCred = await api.getEncryptedPassword(userID, connectionID)

			if (encryptedCred) {
				const password = await decryptPassword(encryptedCred, masterKey)
				connection.password = password
				console.log('Password decrypted from cloud storage')
				return connection
			}
		} catch (err) {
			console.warn('Failed to decrypt password from cloud:', err)
		}
	}

	// Fall back to keychain
	try {
		const password = await api.getPasswordFromKeychain(connectionID)
		connection.password = password
		console.log('Password retrieved from keychain')

		// Opportunistic migration: If we got it from keychain and have master key, migrate it
		if (masterKey) {
			// Don't await - background migration
			migratePasswordInBackground(userID, connectionID, password, masterKey)
		}

		return connection
	} catch (err) {
		// No password found anywhere
		throw new Error('Password not found. Please re-enter your database password.')
	}
}

async function migratePasswordInBackground(
	userID: string,
	connectionID: string,
	password: string,
	masterKey: CryptoKey
) {
	try {
		const encrypted = await encryptPassword(password, masterKey)
		await api.storeEncryptedPassword(userID, connectionID, encrypted)
		await api.markAsMigrated(connectionID)
		console.log(`Background migration completed for connection ${connectionID}`)
	} catch (err) {
		console.error('Background migration failed:', err)
	}
}
```

---

## 7. API Endpoints (Backend)

### GET /api/connections/:id/password

```go
func GetConnectionPassword(w http.ResponseWriter, r *http.Request) {
	connectionID = path_params["id"]
	userID = get_user_from_auth_token(r)
	masterKey = get_master_key_from_session(r) // May be empty

	password, err = passwordManager.GetPassword(ctx, userID, connectionID, masterKey)
	if err != nil {
		return error_response(404, "Password not found")
	}

	// NEVER send plaintext password over API
	// Instead, return an encrypted version with a session key
	sessionKey = generate_session_key()
	encryptedPassword = encrypt_with_session_key(password, sessionKey)

	return json_response({
		"encryptedPassword": encryptedPassword,
		"sessionKey": sessionKey, // Frontend decrypts immediately
	})
}
```

### POST /api/connections/:id/password

```go
func StoreConnectionPassword(w http.ResponseWriter, r *http.Request) {
	connectionID = path_params["id"]
	userID = get_user_from_auth_token(r)
	masterKey = get_master_key_from_session(r)

	body = parse_json(r.Body)
	password = body["password"] // Should be encrypted in transit (HTTPS)

	err = passwordManager.StorePassword(ctx, userID, connectionID, password, masterKey)
	if err != nil {
		return error_response(500, "Failed to store password")
	}

	return json_response({ "success": true })
}
```

### POST /api/passwords/migrate

```go
func MigrateBulkPasswords(w http.ResponseWriter, r *http.Request) {
	userID = get_user_from_auth_token(r)
	masterKey = get_master_key_from_session(r)

	if masterKey is empty {
		return error_response(401, "Master key required for migration")
	}

	result, err = passwordManager.MigrateBulkPasswords(ctx, userID, masterKey)
	if err != nil {
		return error_response(500, "Migration failed")
	}

	return json_response(result)
}
```

### GET /api/passwords/migration-status

```go
func GetMigrationStatus(w http.ResponseWriter, r *http.Request) {
	userID = get_user_from_auth_token(r)

	status = query(`
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN password_migration_status = 'migrated' THEN 1 ELSE 0 END) as migrated,
			SUM(CASE WHEN password_migration_status = 'not_migrated' THEN 1 ELSE 0 END) as unmigrated,
			SUM(CASE WHEN password_migration_status = 'no_password' THEN 1 ELSE 0 END) as no_password
		FROM connection_templates
		WHERE user_id = ? AND deleted_at IS NULL
	`, userID)

	return json_response({
		"totalConnections": status.total,
		"migratedCount": status.migrated,
		"unmigratedCount": status.unmigrated,
		"noPasswordCount": status.no_password,
		"completionPercentage": (status.migrated / status.total) * 100
	})
}
```

---

## 8. Error Handling Patterns

### Handle Missing Master Key

```typescript
try {
	const password = await getPasswordFromEncryptedDB(connectionID)
} catch (MasterKeyNotAvailableError) {
	// Prompt user to re-login
	showReauthPrompt({
		message: "Please log in again to access encrypted passwords",
		onSuccess: async (masterKey) => {
			const password = await decryptPassword(encryptedCred, masterKey)
			return password
		}
	})
}
```

### Handle Decryption Failure

```typescript
try {
	const password = await decryptPassword(encryptedCred, masterKey)
} catch (DecryptionError) {
	// Try keychain as fallback
	try {
		const password = await getPasswordFromKeychain(connectionID)

		// Log issue for investigation
		logError('Decryption failed, fell back to keychain', {
			connectionID,
			error: err.message
		})

		return password
	} catch (KeychainError) {
		// Both failed - ask user to re-enter
		showPasswordPrompt({
			connection: connectionName,
			onSubmit: async (password) => {
				// Re-encrypt and store
				await savePassword(connectionID, password)
			}
		})
	}
}
```

### Handle Network Timeout

```go
func (pm *PasswordManager) storeWithRetry(
	ctx context.Context,
	userID, connectionID string,
	encrypted *crypto.EncryptedPasswordData,
) error {
	maxRetries = 3
	backoff = 1 * time.Second

	for attempt = 1; attempt <= maxRetries; attempt++ {
		err = pm.credentialStore.StoreCredential(ctx, userID, connectionID, encrypted)

		if err == nil {
			return nil // Success
		}

		if is_network_timeout(err) && attempt < maxRetries {
			log("Storage attempt " + attempt + " failed, retrying...")
			time.Sleep(backoff)
			backoff *= 2 // Exponential backoff
		} else {
			return err
		}
	}

	return errors.New("Failed to store credential after " + maxRetries + " attempts")
}
```

---

## 9. Testing Helpers

### Mock Password Manager (for tests)

```go
type MockPasswordManager struct {
	keychainPasswords map[string]string
	encryptedPasswords map[string]*crypto.EncryptedPasswordData
}

func (m *MockPasswordManager) GetPassword(
	ctx context.Context,
	userID, connectionID string,
	masterKey []byte,
) (string, error) {
	// Check encrypted first
	if encrypted, exists := m.encryptedPasswords[connectionID]; exists {
		return crypto.DecryptPasswordWithKey(encrypted, masterKey)
	}

	// Fall back to keychain
	if password, exists := m.keychainPasswords[connectionID]; exists {
		return password, nil
	}

	return "", ErrNotFound
}

func (m *MockPasswordManager) SetupTestData() {
	// Connection 1: Only in keychain (needs migration)
	m.keychainPasswords["conn-1"] = "password123"

	// Connection 2: Only in encrypted DB (already migrated)
	m.encryptedPasswords["conn-2"] = encrypt("password456", testMasterKey)

	// Connection 3: In both (migration in progress)
	m.keychainPasswords["conn-3"] = "password789"
	m.encryptedPasswords["conn-3"] = encrypt("password789", testMasterKey)
}
```

### Integration Test

```go
func TestEndToEndMigration(t *testing.T) {
	// Setup
	ctx = context.Background()
	db = setupTestDatabase()
	pm = NewPasswordManager(db)
	userID = "test-user-1"
	masterKey = generateTestMasterKey()

	// 1. Create connection with password in keychain only
	connectionID = "test-conn-1"
	keychain.Set(connectionID, "original-password")

	// 2. Get password (should trigger opportunistic migration)
	password, err = pm.GetPassword(ctx, userID, connectionID, masterKey)
	assert.NoError(t, err)
	assert.Equal(t, "original-password", password)

	// 3. Wait for background migration
	time.Sleep(100 * time.Millisecond)

	// 4. Verify password now in encrypted DB
	encrypted, err = db.GetEncryptedPassword(userID, connectionID)
	assert.NoError(t, err)
	assert.NotNil(t, encrypted)

	// 5. Verify can decrypt
	decrypted, err = crypto.DecryptPasswordWithKey(encrypted, masterKey)
	assert.NoError(t, err)
	assert.Equal(t, "original-password", decrypted)

	// 6. Verify migration status updated
	status, err = db.GetMigrationStatus(connectionID)
	assert.NoError(t, err)
	assert.Equal(t, "migrated", status)

	// 7. Delete keychain entry
	keychain.Delete(connectionID)

	// 8. Verify still works from encrypted DB
	password, err = pm.GetPassword(ctx, userID, connectionID, masterKey)
	assert.NoError(t, err)
	assert.Equal(t, "original-password", password)
}
```

---

## 10. Monitoring & Logging

### Structured Logging

```go
// Success case
log.WithFields(logrus.Fields{
	"user_id": userID,
	"connection_id": connectionID,
	"source": "encrypted_db",
	"migration_status": "migrated",
}).Info("Password retrieved successfully")

// Fallback case
log.WithFields(logrus.Fields{
	"user_id": userID,
	"connection_id": connectionID,
	"source": "keychain",
	"migration_status": "not_migrated",
	"action": "opportunistic_migration_triggered",
}).Info("Password retrieved from keychain, migration started")

// Error case
log.WithFields(logrus.Fields{
	"user_id": userID,
	"connection_id": connectionID,
	"error": err.Error(),
	"migration_attempt": "automatic",
}).Error("Password migration failed")
```

### Metrics Collection

```go
// Counter: Migration attempts
migrationAttempts.WithLabelValues("automatic", "success").Inc()
migrationAttempts.WithLabelValues("automatic", "failed").Inc()
migrationAttempts.WithLabelValues("manual", "success").Inc()

// Gauge: Migration completion rate
migrationCompletionRate.Set(calculateCompletionRate())

// Histogram: Migration duration
timer := prometheus.NewTimer(migrationDuration)
defer timer.ObserveDuration()
```

---

This pseudocode provides a complete implementation blueprint for the password migration system.
