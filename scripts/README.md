# Howlerops Scripts

This directory contains utility scripts for developing, testing, and installing Howlerops.

## Installation Script

### `install.sh` (in repository root)

Universal installation script for Howlerops that works across all platforms.

**Location:** `/install.sh` (also symlinked in this directory for testing)

**Usage:**
```bash
# Quick install
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh

# Install specific version
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --version v2.0.0

# Dry run
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --dry-run

# Verbose output
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --verbose

# Custom install directory
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --install-dir ~/.local/bin

# Force overwrite
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --force
```

**Features:**
- ✅ Cross-platform: macOS, Linux, Windows (Git Bash/WSL)
- ✅ Multi-architecture: x86_64, arm64, arm
- ✅ Automatic platform detection
- ✅ SHA256 checksum verification
- ✅ Smart installation directory selection
- ✅ PATH configuration assistance
- ✅ Colored, user-friendly output
- ✅ Dry-run mode for previewing
- ✅ Verbose mode for debugging
- ✅ POSIX-compliant (works with sh, bash, zsh)

**Supported Platforms:**

| OS | Architecture | Binary Name |
|----|--------------|-------------|
| macOS | Intel (x86_64) | `sql-studio-darwin-amd64.tar.gz` |
| macOS | Apple Silicon (arm64) | `sql-studio-darwin-arm64.tar.gz` |
| Linux | x86_64 | `sql-studio-linux-amd64.tar.gz` |
| Linux | arm64 | `sql-studio-linux-arm64.tar.gz` |
| Linux | arm | `sql-studio-linux-arm.tar.gz` |
| Windows | x86_64 | `sql-studio-windows-amd64.tar.gz` |
| Windows | arm64 | `sql-studio-windows-arm64.tar.gz` |

## Test Scripts

### `test-install.sh`

Comprehensive test suite for the installation script.

**Usage:**
```bash
./scripts/test-install.sh
```

**Tests:**
- Script exists and is executable
- Help flag works
- Valid shell syntax
- Dry-run mode functionality
- Verbose mode functionality
- Platform detection
- Dependency checking
- Install directory determination
- Custom install directory
- Version specification
- Error handling
- POSIX compliance
- Shellcheck static analysis (if available)

**Example Output:**
```
==========================================
  Howlerops Installer Test Suite
==========================================

[TEST] Script exists and is executable
[PASS] Script exists and is executable
[TEST] Help flag works
[PASS] Help flag works
...
==========================================
  Test Summary
==========================================
Tests run:    12
Tests passed: 12
Tests failed: 0

All tests passed!
```

## Development Scripts

### Running Tests

To test the installation script before committing:

```bash
# Run all tests
./scripts/test-install.sh

# Test locally with dry-run
./install.sh --dry-run --verbose

# Test with specific version
./install.sh --dry-run --version v1.0.0

# Test with custom directory
./install.sh --dry-run --install-dir /tmp/test-install
```

### Static Analysis

If you have shellcheck installed:

```bash
shellcheck install.sh
shellcheck scripts/test-install.sh
```

### Local Testing

To test the installation flow locally:

1. **Create a test release** (or use an existing one)
2. **Run with dry-run** to verify logic:
   ```bash
   ./install.sh --dry-run --verbose --version v1.0.0
   ```
3. **Test in a container** for isolation:
   ```bash
   # Ubuntu
   docker run -it --rm -v $(pwd):/workspace ubuntu:22.04 bash
   cd /workspace
   apt-get update && apt-get install -y curl
   ./install.sh --dry-run

   # Alpine (POSIX sh)
   docker run -it --rm -v $(pwd):/workspace alpine:latest sh
   cd /workspace
   apk add --no-cache curl bash
   ./install.sh --dry-run
   ```

## CI/CD Integration

The installation script is designed to work seamlessly in CI/CD pipelines.

### GitHub Actions

```yaml
- name: Install Howlerops
  run: |
    curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --version v2.0.0
    echo "$HOME/.local/bin" >> $GITHUB_PATH

- name: Verify installation
  run: sql-studio --version
```

### GitLab CI

```yaml
install_sql_studio:
  script:
    - curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --version v2.0.0
    - export PATH="$HOME/.local/bin:$PATH"
    - sql-studio --version
```

### Docker

```dockerfile
FROM ubuntu:22.04

RUN apt-get update && apt-get install -y curl ca-certificates && \
    curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

ENV PATH="/root/.local/bin:${PATH}"

CMD ["sql-studio"]
```

## Release Workflow

The installation script works in conjunction with the GitHub Actions release workflow:

1. **Tag Release:**
   ```bash
   git tag v2.0.0
   git push origin v2.0.0
   ```

2. **Workflow Builds:**
   - Cross-platform binaries (macOS, Linux, Windows)
   - Multi-architecture (amd64, arm64)
   - Archives: `sql-studio-{os}-{arch}.tar.gz`
   - Checksums: Individual `.sha256` files + combined `checksums.txt`

3. **Installer Downloads:**
   - Detects platform and architecture
   - Downloads appropriate archive from GitHub releases
   - Verifies checksum
   - Extracts and installs binary

## Troubleshooting

### Installation Script Issues

**Problem:** Script fails to detect platform
```bash
# Solution: Check platform manually
uname -s  # OS
uname -m  # Architecture

# Try with verbose mode
./install.sh --dry-run --verbose
```

**Problem:** Download fails
```bash
# Solution: Check connectivity and version
curl -I https://github.com/sql-studio/sql-studio/releases/latest

# Try specific version
./install.sh --version v2.0.0
```

**Problem:** Checksum verification fails
```bash
# Solution: Re-download or skip verification manually
# Download archive
curl -LO https://github.com/sql-studio/sql-studio/releases/download/v2.0.0/sql-studio-linux-amd64.tar.gz

# Extract
tar -xzf sql-studio-linux-amd64.tar.gz

# Install
sudo mv sql-studio /usr/local/bin/
sudo chmod +x /usr/local/bin/sql-studio
```

### Test Script Issues

**Problem:** Tests fail on macOS/Linux
```bash
# Solution: Ensure script is executable
chmod +x install.sh
chmod +x scripts/test-install.sh

# Run with verbose bash
bash -x scripts/test-install.sh
```

**Problem:** Shellcheck not found
```bash
# Install shellcheck
# macOS
brew install shellcheck

# Ubuntu/Debian
sudo apt-get install shellcheck

# Or skip shellcheck tests (they're optional)
```

## Contributing

When modifying the installation script:

1. **Maintain POSIX compliance** - use `sh`, not `bash`-specific features
2. **Test on multiple platforms** - macOS, Linux (various distros), Windows (WSL/Git Bash)
3. **Run test suite** - `./scripts/test-install.sh`
4. **Run shellcheck** - `shellcheck install.sh`
5. **Test dry-run mode** - `./install.sh --dry-run --verbose`
6. **Update documentation** - Keep this README and `docs/INSTALLATION.md` in sync

## Security

The installation script follows security best practices:

- ✅ Downloads only from official GitHub releases
- ✅ Verifies SHA256 checksums
- ✅ Uses HTTPS exclusively
- ✅ Minimal permissions (no sudo unless necessary)
- ✅ No code execution from downloaded content
- ✅ Transparent operations (dry-run and verbose modes)

## License

All scripts in this directory are part of Howlerops and released under the MIT License.
See [LICENSE](../LICENSE) for details.
