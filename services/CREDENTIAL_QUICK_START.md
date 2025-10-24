# Credential Service - Quick Start Guide

## 30-Second Quick Start

```go
import "github.com/sql-studio/sql-studio/services"

// Create service
credService := services.NewCredentialService(logger)

// Store password
credService.StorePassword("my-connection", "password123")

// Get password
password, err := credService.GetPassword("my-connection")

// Delete password
credService.DeletePassword("my-connection")
```

## Platform Requirements

| Platform | Backend | Requirements | Notes |
|----------|---------|--------------|-------|
| macOS | Keychain Access | None (built-in) | May prompt for password |
| Windows | Credential Manager | None (built-in) | May prompt for permission |
| Linux | Secret Service API | `libsecret-1-0` | `sudo apt install libsecret-1-0` |

## Error Handling Cheat Sheet

```go
password, err := credService.GetPassword(id)
if err != nil {
    switch {
    case errors.Is(err, services.ErrNotFound):
        // Credential doesn't exist - prompt user

    case errors.Is(err, services.ErrPermissionDenied):
        // User denied access - show help message

    case errors.Is(err, services.ErrUnavailable):
        // Keychain not available - use fallback

    case errors.Is(err, services.ErrInvalidInput):
        // Programming error - fix the code
    }
}
```

## Common Patterns

### Store on Connection Create

```go
func CreateConnection(config ConnectionConfig) error {
    if config.Password != "" {
        credService.StorePassword(config.ID, config.Password)
        config.Password = "" // Don't store in config
    }
    return dbService.CreateConnection(config)
}
```

### Retrieve on Connection Use

```go
func Connect(connectionID string) error {
    password, err := credService.GetPassword(connectionID)
    if errors.Is(err, services.ErrNotFound) {
        return PromptForPassword(connectionID)
    }
    return ConnectToDatabase(connectionID, password)
}
```

### Clean Up on Delete

```go
func DeleteConnection(connectionID string) error {
    credService.DeletePassword(connectionID) // Always clean up
    return dbService.RemoveConnection(connectionID)
}
```

## Health Check Before Use

```go
if err := credService.HealthCheck(); err != nil {
    log.Println("Keychain unavailable, using manual entry")
    return useFallbackStorage()
}
// Keychain is available, use it
```

## Platform Info

```go
info := credService.GetPlatformInfo()
fmt.Printf("Using %s on %s\n", info["backend"], info["platform"])
```

## API Summary

| Method | Purpose | Thread-Safe |
|--------|---------|-------------|
| `StorePassword(id, pwd)` | Store credential | Yes |
| `GetPassword(id)` | Retrieve credential | Yes |
| `DeletePassword(id)` | Remove credential | Yes |
| `HasPassword(id)` | Check existence | Yes |
| `UpdatePassword(id, pwd)` | Update credential | Yes |
| `GetPlatformInfo()` | Platform details | Yes |
| `HealthCheck()` | Test availability | Yes |

## Testing

```bash
# Run tests
go test ./services/ -run TestCredentialService -v

# With coverage
go test ./services/ -run TestCredentialService -cover

# Run examples
go test ./services/ -run Example -v
```

## Troubleshooting

### Linux: "secret service is not available"

```bash
sudo apt-get install libsecret-1-0
```

### macOS: "access denied"

1. Open Keychain Access.app
2. Find "HowlerOps SQL Studio" entries
3. Right-click → Get Info → Access Control
4. Click "Allow all applications to access this item"

### Windows: "credential manager unavailable"

1. Open Services (Win+R → services.msc)
2. Find "Credential Manager" service
3. Start and set to Automatic

## Security Best Practices

✅ **DO:**
- Use credential service for all production passwords
- Check `HealthCheck()` before assuming availability
- Handle all error types appropriately
- Clean up credentials when connections are deleted

❌ **DON'T:**
- Store passwords in plaintext config files
- Log passwords or password lengths
- Assume keychain is always available
- Skip error handling

## Full Documentation

See `/Users/jacob_1/projects/sql-studio/services/CREDENTIAL_SERVICE.md` for:
- Complete API reference
- Detailed examples
- Platform-specific guides
- Integration patterns
- Security considerations

## Support

The credential service is production-ready and fully tested. For issues:

1. Check this quick start guide
2. Review full documentation
3. Check troubleshooting section
4. Run health check to verify availability
