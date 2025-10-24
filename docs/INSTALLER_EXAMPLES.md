# SQL Studio Installer - Usage Examples

This document provides real-world examples of using the SQL Studio installer in various scenarios.

## Basic Usage

### Standard Installation

```bash
# Install latest version
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh
```

**Expected Output:**
```
╔═══════════════════════════════════════╗
║    SQL Studio Universal Installer    ║
╚═══════════════════════════════════════╝

[INFO] Detected platform: darwin-arm64
[INFO] Checking dependencies...
[INFO] Latest version: v2.0.0
[INFO] Installation directory: ~/.local/bin
[INFO] Downloading SQL Studio v2.0.0...
[INFO] Verifying checksum...
[SUCCESS] Checksum verified
[INFO] Installing to ~/.local/bin...
[SUCCESS] Binary installed successfully

✓ SQL Studio has been installed successfully!

Location: ~/.local/bin/sql-studio
Version:  v2.0.0

Get started:
  sql-studio --help
  sql-studio version

Documentation: https://docs.sqlstudio.io
```

### Preview Before Installing

```bash
# Dry-run to see what would happen
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --dry-run
```

**Use Case:** You want to verify which binary will be downloaded and where it will be installed without actually making changes.

### Install Specific Version

```bash
# Install v1.5.0 instead of latest
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --version v1.5.0
```

**Use Case:** You need a specific version for compatibility or testing purposes.

### Verbose Installation

```bash
# See detailed output during installation
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --verbose
```

**Use Case:** Troubleshooting installation issues or understanding exactly what the script is doing.

## Advanced Usage

### Custom Installation Directory

```bash
# Install to ~/bin instead of ~/.local/bin
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --install-dir ~/bin
```

**Use Case:** You prefer to keep binaries in a custom location or have specific PATH requirements.

### Reinstall/Upgrade

```bash
# Force reinstall to upgrade existing installation
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --force
```

**Use Case:** Upgrading to a newer version or fixing a corrupted installation.

### Multiple Options Combined

```bash
# Install specific version to custom directory with verbose output
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- \
  --version v2.0.0 \
  --install-dir ~/tools/bin \
  --verbose \
  --force
```

**Use Case:** Complex installation scenarios requiring multiple customizations.

## Environment Variable Usage

### Using Environment Variables

```bash
# Set version and directory via environment variables
export VERSION="v2.0.0"
export INSTALL_DIR="$HOME/.local/bin"
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh
```

**Use Case:** Scripting installations or using configuration management tools.

### CI/CD Pipeline

```bash
# In a shell script
#!/bin/bash
set -e

# Configure installation
export VERSION="${SQL_STUDIO_VERSION:-latest}"
export INSTALL_DIR="${HOME}/.local/bin"

# Install
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh

# Verify
sql-studio --version
```

**Use Case:** Automated deployments in CI/CD pipelines.

## Platform-Specific Examples

### macOS (Apple Silicon)

```bash
# Install on Apple Silicon Mac
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh
# Downloads: sql-studio-darwin-arm64.tar.gz
```

### macOS (Intel)

```bash
# Install on Intel Mac
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh
# Downloads: sql-studio-darwin-amd64.tar.gz
```

### Linux (Ubuntu/Debian)

```bash
# Install on Ubuntu/Debian
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh
# Downloads: sql-studio-linux-amd64.tar.gz
```

### Linux (ARM64 Server)

```bash
# Install on ARM64 server (AWS Graviton, etc.)
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh
# Downloads: sql-studio-linux-arm64.tar.gz
```

### Windows (Git Bash)

```bash
# Install on Windows using Git Bash
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh
# Downloads: sql-studio-windows-amd64.tar.gz
```

### Raspberry Pi

```bash
# Install on Raspberry Pi (ARMv7)
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh
# Downloads: sql-studio-linux-arm.tar.gz
```

## Docker Examples

### Basic Dockerfile

```dockerfile
FROM ubuntu:22.04

# Install dependencies
RUN apt-get update && apt-get install -y curl ca-certificates

# Install SQL Studio
RUN curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh

# Add to PATH
ENV PATH="/root/.local/bin:${PATH}"

# Verify installation
RUN sql-studio --version

CMD ["sql-studio"]
```

### Multi-stage Build

```dockerfile
# Stage 1: Install SQL Studio
FROM ubuntu:22.04 AS installer

RUN apt-get update && apt-get install -y curl ca-certificates && \
    curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh

# Stage 2: Runtime
FROM ubuntu:22.04

# Copy only the binary
COPY --from=installer /root/.local/bin/sql-studio /usr/local/bin/sql-studio

# Verify
RUN sql-studio --version

CMD ["sql-studio"]
```

### Alpine Linux

```dockerfile
FROM alpine:latest

# Install dependencies
RUN apk add --no-cache curl bash ca-certificates

# Install SQL Studio
RUN curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh

# Add to PATH
ENV PATH="/root/.local/bin:${PATH}"

CMD ["sql-studio"]
```

## CI/CD Examples

### GitHub Actions

```yaml
name: Test with SQL Studio

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install SQL Studio
        run: |
          curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --version v2.0.0
          echo "$HOME/.local/bin" >> $GITHUB_PATH

      - name: Verify installation
        run: sql-studio --version

      - name: Run tests
        run: |
          # Your test commands using sql-studio
          sql-studio --help
```

### GitLab CI

```yaml
test_sql_studio:
  image: ubuntu:22.04
  before_script:
    - apt-get update -qq
    - apt-get install -y -qq curl
    - curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --version v2.0.0
    - export PATH="$HOME/.local/bin:$PATH"
  script:
    - sql-studio --version
    # Your test commands
```

### CircleCI

```yaml
version: 2.1

jobs:
  test:
    docker:
      - image: cimg/base:stable
    steps:
      - checkout
      - run:
          name: Install SQL Studio
          command: |
            curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --version v2.0.0
            echo 'export PATH="$HOME/.local/bin:$PATH"' >> $BASH_ENV
      - run:
          name: Verify installation
          command: sql-studio --version
```

### Jenkins Pipeline

```groovy
pipeline {
    agent any

    stages {
        stage('Install SQL Studio') {
            steps {
                sh '''
                    curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --version v2.0.0
                    export PATH="$HOME/.local/bin:$PATH"
                    sql-studio --version
                '''
            }
        }

        stage('Test') {
            steps {
                sh '''
                    export PATH="$HOME/.local/bin:$PATH"
                    # Your test commands
                    sql-studio --help
                '''
            }
        }
    }
}
```

## Troubleshooting Examples

### Installation Behind Proxy

```bash
# Set proxy environment variables
export https_proxy=http://proxy.company.com:8080
export http_proxy=http://proxy.company.com:8080

# Then install
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh
```

### Installation Without curl

```bash
# Download script first
wget https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh

# Make executable
chmod +x install.sh

# Run locally
./install.sh --version v2.0.0
```

### Installation to System Directory (requires sudo)

```bash
# Install to /usr/local/bin (requires sudo)
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sudo sh -s -- --install-dir /usr/local/bin
```

### Verify Checksum Manually

```bash
# Download binary and checksums
wget https://github.com/sql-studio/sql-studio/releases/download/v2.0.0/sql-studio-linux-amd64.tar.gz
wget https://github.com/sql-studio/sql-studio/releases/download/v2.0.0/checksums.txt

# Verify (Linux)
sha256sum -c checksums.txt

# Verify (macOS)
shasum -a 256 -c checksums.txt

# Extract if verification passes
tar -xzf sql-studio-linux-amd64.tar.gz
sudo mv sql-studio /usr/local/bin/
```

### Debug Installation Issues

```bash
# Run with verbose output and dry-run
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --dry-run --verbose 2>&1 | tee install.log

# Review the log
cat install.log
```

## Uninstallation Examples

### Remove Binary

```bash
# Remove from ~/.local/bin
rm ~/.local/bin/sql-studio

# Or from /usr/local/bin
sudo rm /usr/local/bin/sql-studio
```

### Complete Cleanup

```bash
# Remove binary
rm ~/.local/bin/sql-studio

# Remove configuration (if any)
rm -rf ~/.config/sql-studio

# Remove cache (if any)
rm -rf ~/.cache/sql-studio
```

## Integration Examples

### Shell Script Integration

```bash
#!/bin/bash
# setup-dev-environment.sh

set -e

echo "Setting up development environment..."

# Install SQL Studio
if ! command -v sql-studio >/dev/null 2>&1; then
    echo "Installing SQL Studio..."
    curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh
    export PATH="$HOME/.local/bin:$PATH"
else
    echo "SQL Studio already installed"
fi

# Verify installation
sql-studio --version

echo "Development environment ready!"
```

### Makefile Integration

```makefile
.PHONY: install-tools
install-tools:
	@echo "Installing development tools..."
	@command -v sql-studio >/dev/null 2>&1 || \
		curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh
	@sql-studio --version
```

### Ansible Playbook

```yaml
- name: Install SQL Studio
  hosts: all
  tasks:
    - name: Download and run installer
      shell: |
        curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --version v2.0.0
      args:
        creates: "{{ ansible_env.HOME }}/.local/bin/sql-studio"

    - name: Verify installation
      command: sql-studio --version
      register: version_output

    - name: Display version
      debug:
        var: version_output.stdout
```

## Best Practices

1. **Always specify version in production:**
   ```bash
   curl -fsSL ... | sh -s -- --version v2.0.0
   ```

2. **Use dry-run first for new environments:**
   ```bash
   curl -fsSL ... | sh -s -- --dry-run
   ```

3. **Add to PATH in shell profile:**
   ```bash
   echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
   ```

4. **Verify installation:**
   ```bash
   sql-studio --version
   ```

5. **Use environment variables in scripts:**
   ```bash
   export VERSION="v2.0.0"
   export INSTALL_DIR="$HOME/.local/bin"
   ```

## Getting Help

If you encounter issues:

1. **Run with verbose output:**
   ```bash
   curl -fsSL ... | sh -s -- --verbose
   ```

2. **Check the help:**
   ```bash
   curl -fsSL ... | sh -s -- --help
   ```

3. **Open an issue:** https://github.com/sql-studio/sql-studio/issues

4. **Check documentation:** [docs/INSTALLATION.md](docs/INSTALLATION.md)
