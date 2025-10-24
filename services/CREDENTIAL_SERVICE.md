# Credential Service Documentation

## Overview

The `CredentialService` provides secure storage of database passwords using the operating system's native keychain/credential manager. This ensures passwords are stored securely outside of application configuration files and are protected by OS-level security mechanisms.

## Supported Platforms

### macOS
- **Backend**: Keychain Access
- **How it works**: Credentials are stored in the macOS Keychain and can be viewed/managed via Keychain Access.app
- **User Experience**: May prompt for keychain password on first access
- **Security**: Protected by macOS keychain security with optional biometric unlock

### Windows
- **Backend**: Windows Credential Manager
- **How it works**: Credentials are stored in Windows Credential Manager
- **User Experience**: May prompt for access permission
- **Security**: Protected by Windows security with DPAPI encryption

### Linux
- **Backend**: Secret Service API (libsecret)
- **How it works**: Uses D-Bus Secret Service API (GNOME Keyring, KDE Wallet)
- **Requirements**: `libsecret-1-0` package must be installed
- **Installation**: `sudo apt-get install libsecret-1-0` (Ubuntu/Debian)
- **Security**: Protected by desktop environment keyring

## Usage

### Basic Example

```go
package main

import (
    "log"
    "github.com/sirupsen/logrus"
    "github.com/sql-studio/sql-studio/services"
)

func main() {
    // Create logger
    logger := logrus.New()
    logger.SetLevel(logrus.InfoLevel)

    // Create credential service
    credService := services.NewCredentialService(logger)

    // Store a password
    connectionID := "my-postgres-connection"
    password := "super-secret-password"

    if err := credService.StorePassword(connectionID, password); err != nil {
        log.Fatalf("Failed to store password: %v", err)
    }
    log.Println("Password stored successfully")

    // Retrieve the password
    retrievedPassword, err := credService.GetPassword(connectionID)
    if err != nil {
        log.Fatalf("Failed to get password: %v", err)
    }
    log.Printf("Retrieved password: %s\n", retrievedPassword)

    // Delete the password
    if err := credService.DeletePassword(connectionID); err != nil {
        log.Fatalf("Failed to delete password: %v", err)
    }
    log.Println("Password deleted successfully")
}
```

### Health Check

Check if the keychain is available and working:

```go
credService := services.NewCredentialService(logger)

if err := credService.HealthCheck(); err != nil {
    log.Printf("Keychain unavailable: %v", err)
    // Fall back to manual credential storage
} else {
    log.Println("Keychain is working properly")
}
```

### Platform Information

Get information about the current platform's keychain support:

```go
credService := services.NewCredentialService(logger)
info := credService.GetPlatformInfo()

fmt.Printf("Platform: %s\n", info["platform"])
fmt.Printf("Backend: %s\n", info["backend"])
fmt.Printf("Supported: %v\n", info["supported"])
fmt.Printf("Notes: %s\n", info["notes"])
```

## API Reference

### Types

#### Sentinel Errors

```go
var (
    // ErrNotFound is returned when a credential is not found in the keychain
    ErrNotFound = errors.New("credential not found")

    // ErrPermissionDenied is returned when access to the keychain is denied
    ErrPermissionDenied = errors.New("permission denied to access keychain")

    // ErrUnavailable is returned when the keychain service is not available
    ErrUnavailable = errors.New("keychain service unavailable")

    // ErrInvalidInput is returned when invalid parameters are provided
    ErrInvalidInput = errors.New("invalid input parameters")
)
```

### Methods

#### NewCredentialService

```go
func NewCredentialService(logger *logrus.Logger) *CredentialService
```

Creates a new credential service instance. If logger is nil, a default logger with WARN level is created.

#### StorePassword

```go
func (s *CredentialService) StorePassword(connectionID, password string) error
```

Securely stores a password for the given connection ID in the OS keychain.

**Parameters:**
- `connectionID`: Unique identifier for the database connection (used as keychain account)
- `password`: The password to store securely

**Returns:**
- `nil` on success
- `ErrInvalidInput` if connectionID or password is empty
- `ErrPermissionDenied` if access to keychain is denied
- `ErrUnavailable` if keychain service is not available

**Thread-safe**: Yes (protected by mutex)

#### GetPassword

```go
func (s *CredentialService) GetPassword(connectionID string) (string, error)
```

Retrieves a password for the given connection ID from the OS keychain.

**Parameters:**
- `connectionID`: Unique identifier for the database connection

**Returns:**
- Password string and nil error on success
- Empty string and `ErrNotFound` if credential doesn't exist
- Empty string and `ErrPermissionDenied` if access is denied
- Empty string and `ErrUnavailable` if keychain service is not available

**Thread-safe**: Yes (protected by read mutex)

#### DeletePassword

```go
func (s *CredentialService) DeletePassword(connectionID string) error
```

Removes a password for the given connection ID from the OS keychain.

**Parameters:**
- `connectionID`: Unique identifier for the database connection

**Returns:**
- `nil` on success or if credential doesn't exist (idempotent)
- `ErrInvalidInput` if connectionID is empty
- `ErrPermissionDenied` if access to keychain is denied
- `ErrUnavailable` if keychain service is not available

**Thread-safe**: Yes (protected by mutex)

**Note**: This operation is idempotent - deleting a non-existent credential is not an error.

#### HasPassword

```go
func (s *CredentialService) HasPassword(connectionID string) bool
```

Checks if a password exists in the keychain for a given connection.

**Parameters:**
- `connectionID`: Unique identifier for the database connection

**Returns:**
- `true` if password exists
- `false` if password doesn't exist or connectionID is empty

#### UpdatePassword

```go
func (s *CredentialService) UpdatePassword(connectionID, newPassword string) error
```

Updates an existing password in the keychain. This is an alias for `StorePassword` as the keychain overwrites existing entries.

#### GetPlatformInfo

```go
func (s *CredentialService) GetPlatformInfo() map[string]interface{}
```

Returns information about the current platform's keychain support.

**Returns:**
Map with the following keys:
- `platform`: OS platform (darwin, windows, linux)
- `service`: Service name used in keychain
- `backend`: Name of the keychain backend
- `supported`: Boolean indicating if platform is supported
- `notes`: Additional notes about the platform
- `install_hint`: (Linux only) Installation command for required packages

#### HealthCheck

```go
func (s *CredentialService) HealthCheck() error
```

Performs a basic health check of the keychain service by attempting to store, retrieve, and delete a test credential.

**Returns:**
- `nil` if keychain is working properly
- Error describing what's wrong if keychain is unavailable

## Error Handling

The service provides comprehensive error handling with user-friendly messages:

```go
password, err := credService.GetPassword("my-connection")
if err != nil {
    switch {
    case errors.Is(err, services.ErrNotFound):
        // Credential doesn't exist - prompt user to enter password
        fmt.Println("No saved password found")

    case errors.Is(err, services.ErrPermissionDenied):
        // User denied keychain access - explain and fall back
        fmt.Println("Access to keychain denied. Please grant permission.")

    case errors.Is(err, services.ErrUnavailable):
        // Keychain not available - use alternative storage
        fmt.Println("Keychain not available on this system")

    case errors.Is(err, services.ErrInvalidInput):
        // Programming error - fix the code
        log.Fatalf("Invalid input: %v", err)

    default:
        // Unknown error
        log.Printf("Unexpected error: %v", err)
    }
}
```

## Security Considerations

### What Gets Logged

The service is designed with security in mind:

✅ **Logged:**
- Connection IDs
- Operation types (store, get, delete)
- Success/failure status
- Platform information
- Error types

❌ **Never Logged:**
- Passwords or password content
- Password lengths
- Password hashes

### Best Practices

1. **Never store passwords in plain text configuration files**
   - Always use the credential service for production deployments

2. **Provide fallback for unavailable keychains**
   - Check `HealthCheck()` before relying on keychain
   - Implement manual password entry as fallback

3. **Handle permission denials gracefully**
   - Inform users why keychain access is beneficial
   - Provide clear instructions for granting access

4. **Use meaningful connection IDs**
   - Connection IDs should be unique per database connection
   - Consider including host/port in the ID for clarity

5. **Clean up credentials when connections are deleted**
   - Call `DeletePassword()` when removing a connection

## Integration with SQL Studio

### Desktop Application (Wails)

```go
// In your connection manager
type ConnectionManager struct {
    credService *services.CredentialService
    dbService   *services.DatabaseService
}

func (cm *ConnectionManager) CreateConnection(config database.ConnectionConfig) error {
    // Store password in keychain if provided
    if config.Password != "" {
        if err := cm.credService.StorePassword(config.ID, config.Password); err != nil {
            log.Printf("Warning: Failed to store password in keychain: %v", err)
            // Continue anyway - user can manually enter password
        }
        // Don't store password in config
        config.Password = ""
    }

    return cm.dbService.CreateConnection(config)
}

func (cm *ConnectionManager) GetConnection(connectionID string) (database.Database, error) {
    // Try to retrieve password from keychain
    password, err := cm.credService.GetPassword(connectionID)
    if err != nil {
        if errors.Is(err, services.ErrNotFound) {
            // Prompt user for password
            return nil, fmt.Errorf("password required for connection")
        }
        return nil, fmt.Errorf("failed to retrieve password: %w", err)
    }

    // Use password to connect
    // ...
}
```

### Backend Server

For the backend server, the credential service can be used to store sensitive configuration like API keys and service passwords:

```go
// Store API keys securely
credService.StorePassword("openai-api-key", os.Getenv("OPENAI_API_KEY"))

// Retrieve when needed
apiKey, err := credService.GetPassword("openai-api-key")
if err != nil {
    log.Fatalf("Failed to retrieve API key: %v", err)
}
```

## Troubleshooting

### Linux: "secret service is not available"

**Problem**: Linux system doesn't have libsecret installed.

**Solution**:
```bash
# Ubuntu/Debian
sudo apt-get install libsecret-1-0

# Fedora/RHEL
sudo dnf install libsecret

# Arch Linux
sudo pacman -S libsecret
```

### macOS: "keychain access denied"

**Problem**: User denied access to keychain.

**Solution**:
1. Open "Keychain Access.app"
2. Find "HowlerOps SQL Studio" entries
3. Double-click entry → Access Control → Allow all applications
4. Or grant access when prompted

### Windows: "credential manager unavailable"

**Problem**: Windows Credential Manager service is disabled.

**Solution**:
1. Open Services (services.msc)
2. Find "Credential Manager" service
3. Start the service and set to Automatic

### CI/CD: Tests failing in CI

**Problem**: CI environments don't have keychain access.

**Solution**: Tests use mock keyring automatically. If integration tests fail, skip keychain tests in CI:

```go
func shouldSkipKeychainTests() bool {
    return os.Getenv("CI") == "true"
}

func TestCredentialService(t *testing.T) {
    if shouldSkipKeychainTests() {
        t.Skip("Skipping keychain test in CI")
    }
    // ... test code
}
```

## Performance

The credential service is designed for high performance:

- **Thread-safe**: Uses RWMutex for concurrent access
- **Fast operations**: OS keychains are optimized for quick access
- **No caching**: Always retrieves from OS keychain for maximum security
- **Minimal overhead**: Direct binding to OS APIs

Typical operation times:
- Store: < 10ms
- Get: < 5ms
- Delete: < 10ms

## Dependencies

- `github.com/zalando/go-keyring` - Cross-platform keychain access
- `github.com/sirupsen/logrus` - Structured logging

## Testing

Run tests:
```bash
go test -v ./services/ -run TestCredentialService
```

The tests use mock keyring to avoid requiring actual keychain access during testing.

## License

This credential service is part of SQL Studio and is licensed under the same terms as the main project.
