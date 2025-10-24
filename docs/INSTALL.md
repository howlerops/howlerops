# Installation Guide

This guide covers all methods of installing SQL Studio Backend.

## Table of Contents

- [Quick Install (Recommended)](#quick-install-recommended)
- [Installation Methods](#installation-methods)
  - [Shell Script Installer](#shell-script-installer)
  - [Homebrew (macOS/Linux)](#homebrew-macoslinux)
  - [Direct Download](#direct-download)
  - [Build from Source](#build-from-source)
  - [Docker](#docker)
- [Platform-Specific Instructions](#platform-specific-instructions)
  - [macOS](#macos)
  - [Linux](#linux)
  - [Windows](#windows)
- [Configuration](#configuration)
- [Verification](#verification)
- [Upgrading](#upgrading)
- [Uninstallation](#uninstallation)
- [Troubleshooting](#troubleshooting)

## Quick Install (Recommended)

### macOS / Linux

```bash
# One-line installer (downloads and installs latest version)
curl -fsSL https://install.sqlstudio.io/install.sh | bash
```

This script will:
1. Detect your OS and architecture
2. Download the appropriate binary
3. Verify checksum
4. Install to `/usr/local/bin`
5. Set up configuration directory

### Windows

```powershell
# PowerShell installer (run as Administrator)
irm https://install.sqlstudio.io/install.ps1 | iex
```

## Installation Methods

### Shell Script Installer

**macOS and Linux**

```bash
# Download and run installer
curl -fsSL https://install.sqlstudio.io/install.sh -o install.sh
chmod +x install.sh
./install.sh

# Install specific version
./install.sh --version 2.1.0

# Install to custom location
./install.sh --prefix /opt/sqlstudio

# See all options
./install.sh --help
```

**Windows PowerShell**

```powershell
# Download installer
Invoke-WebRequest -Uri https://install.sqlstudio.io/install.ps1 -OutFile install.ps1

# Run installer
.\install.ps1

# Install specific version
.\install.ps1 -Version "2.1.0"

# Install to custom location
.\install.ps1 -InstallPath "C:\Program Files\SQLStudio"
```

### Homebrew (macOS/Linux)

**Coming Soon**

```bash
# Add tap
brew tap sqlstudio/tap

# Install
brew install sqlstudio

# Upgrade
brew upgrade sqlstudio

# Uninstall
brew uninstall sqlstudio
```

### Direct Download

Download pre-built binaries from [GitHub Releases](https://github.com/yourusername/sql-studio/releases):

1. **Choose your platform**:
   - macOS Intel: `sql-studio-2.1.0-darwin-amd64.tar.gz`
   - macOS Apple Silicon: `sql-studio-2.1.0-darwin-arm64.tar.gz`
   - Linux AMD64: `sql-studio-2.1.0-linux-amd64.tar.gz`
   - Linux ARM64: `sql-studio-2.1.0-linux-arm64.tar.gz`
   - Windows AMD64: `sql-studio-2.1.0-windows-amd64.zip`

2. **Download and extract**:

   ```bash
   # macOS/Linux example
   VERSION=2.1.0
   PLATFORM=darwin-arm64  # or darwin-amd64, linux-amd64, etc.

   curl -L -o sql-studio.tar.gz \
     "https://github.com/yourusername/sql-studio/releases/download/v${VERSION}/sql-studio-${VERSION}-${PLATFORM}.tar.gz"

   tar -xzf sql-studio.tar.gz
   ```

3. **Verify checksum** (recommended):

   ```bash
   # Download checksums file
   curl -L -o checksums.txt \
     "https://github.com/yourusername/sql-studio/releases/download/v${VERSION}/checksums.txt"

   # Verify (macOS/Linux)
   shasum -a 256 -c checksums.txt --ignore-missing

   # Or manually
   shasum -a 256 sql-studio-${PLATFORM}
   ```

4. **Install binary**:

   ```bash
   # macOS/Linux
   sudo mv sql-studio-${PLATFORM} /usr/local/bin/sql-studio
   sudo chmod +x /usr/local/bin/sql-studio

   # Verify installation
   sql-studio --version
   ```

### Build from Source

**Prerequisites**:
- Go 1.24 or later
- Git
- SQLite development headers
- GCC (for CGO)

**Steps**:

```bash
# Clone repository
git clone https://github.com/yourusername/sql-studio.git
cd sql-studio/backend-go

# Install dependencies
make deps

# Build for current platform
make release

# Or build for all platforms
make release-all VERSION=2.1.0

# Install
sudo cp build/sql-studio-backend /usr/local/bin/sql-studio
sudo chmod +x /usr/local/bin/sql-studio

# Verify
sql-studio --version
```

### Docker

**Using Docker Hub**:

```bash
# Pull latest image
docker pull sqlstudio/backend:latest

# Or specific version
docker pull sqlstudio/backend:2.1.0

# Run container
docker run -d \
  --name sql-studio \
  -p 8080:8080 \
  -p 9090:9090 \
  -e TURSO_URL="libsql://your-db.turso.io" \
  -e TURSO_AUTH_TOKEN="your-token" \
  -e JWT_SECRET="your-secret" \
  -v sql-studio-data:/data \
  sqlstudio/backend:latest
```

**Building from source**:

```bash
# Clone repository
git clone https://github.com/yourusername/sql-studio.git
cd sql-studio/backend-go

# Build Docker image
docker build -t sql-studio:local .

# Run
docker run -d \
  --name sql-studio \
  -p 8080:8080 \
  -p 9090:9090 \
  -e TURSO_URL="libsql://your-db.turso.io" \
  -e TURSO_AUTH_TOKEN="your-token" \
  sql-studio:local
```

## Platform-Specific Instructions

### macOS

**Intel Macs (x86_64)**:

```bash
VERSION=2.1.0
curl -L -o sql-studio.tar.gz \
  "https://github.com/yourusername/sql-studio/releases/download/v${VERSION}/sql-studio-${VERSION}-darwin-amd64.tar.gz"

tar -xzf sql-studio.tar.gz
sudo mv sql-studio-darwin-amd64 /usr/local/bin/sql-studio
sudo chmod +x /usr/local/bin/sql-studio

# First run may require security approval
sql-studio --version
```

**Apple Silicon (M1/M2/M3)**:

```bash
VERSION=2.1.0
curl -L -o sql-studio.tar.gz \
  "https://github.com/yourusername/sql-studio/releases/download/v${VERSION}/sql-studio-${VERSION}-darwin-arm64.tar.gz"

tar -xzf sql-studio.tar.gz
sudo mv sql-studio-darwin-arm64 /usr/local/bin/sql-studio
sudo chmod +x /usr/local/bin/sql-studio

# Allow in System Settings → Privacy & Security if needed
sql-studio --version
```

**macOS Security Note**:

If you see "cannot be opened because it is from an unidentified developer":

```bash
# Remove quarantine attribute
sudo xattr -d com.apple.quarantine /usr/local/bin/sql-studio

# Or approve in System Settings → Privacy & Security
```

### Linux

**Ubuntu/Debian (AMD64)**:

```bash
VERSION=2.1.0
curl -L -o sql-studio.tar.gz \
  "https://github.com/yourusername/sql-studio/releases/download/v${VERSION}/sql-studio-${VERSION}-linux-amd64.tar.gz"

tar -xzf sql-studio.tar.gz
sudo mv sql-studio-linux-amd64 /usr/local/bin/sql-studio
sudo chmod +x /usr/local/bin/sql-studio

sql-studio --version
```

**ARM64 (Raspberry Pi, ARM servers)**:

```bash
VERSION=2.1.0
curl -L -o sql-studio.tar.gz \
  "https://github.com/yourusername/sql-studio/releases/download/v${VERSION}/sql-studio-${VERSION}-linux-arm64.tar.gz"

tar -xzf sql-studio.tar.gz
sudo mv sql-studio-linux-arm64 /usr/local/bin/sql-studio
sudo chmod +x /usr/local/bin/sql-studio

sql-studio --version
```

**Systemd Service** (optional):

```bash
# Create service file
sudo tee /etc/systemd/system/sql-studio.service > /dev/null <<EOF
[Unit]
Description=SQL Studio Backend
After=network.target

[Service]
Type=simple
User=sqlstudio
Group=sqlstudio
ExecStart=/usr/local/bin/sql-studio
Restart=on-failure
RestartSec=5s
Environment="TURSO_URL=libsql://your-db.turso.io"
Environment="TURSO_AUTH_TOKEN=your-token"
Environment="JWT_SECRET=your-secret"

[Install]
WantedBy=multi-user.target
EOF

# Create user
sudo useradd -r -s /bin/false sqlstudio

# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable sql-studio
sudo systemctl start sql-studio

# Check status
sudo systemctl status sql-studio
```

### Windows

**Installation**:

```powershell
# Download
$VERSION = "2.1.0"
Invoke-WebRequest -Uri "https://github.com/yourusername/sql-studio/releases/download/v$VERSION/sql-studio-$VERSION-windows-amd64.zip" -OutFile sql-studio.zip

# Extract
Expand-Archive -Path sql-studio.zip -DestinationPath "C:\Program Files\SQLStudio"

# Add to PATH (as Administrator)
$env:Path += ";C:\Program Files\SQLStudio"
[System.Environment]::SetEnvironmentVariable("Path", $env:Path, [System.EnvironmentVariableTarget]::Machine)

# Verify
sql-studio-windows-amd64.exe --version
```

**Windows Service** (optional):

Using [NSSM](https://nssm.cc/):

```powershell
# Install NSSM
choco install nssm

# Install service
nssm install SQLStudio "C:\Program Files\SQLStudio\sql-studio-windows-amd64.exe"

# Configure environment variables
nssm set SQLStudio AppEnvironmentExtra TURSO_URL=libsql://your-db.turso.io
nssm set SQLStudio AppEnvironmentExtra TURSO_AUTH_TOKEN=your-token

# Start service
nssm start SQLStudio

# Check status
nssm status SQLStudio
```

## Configuration

### Environment Variables

Create a configuration file:

**Linux/macOS** (`~/.sql-studio/config.env`):

```bash
# Required
TURSO_URL=libsql://your-db.turso.io
TURSO_AUTH_TOKEN=your-token
JWT_SECRET=your-secret-min-32-chars

# Optional
HTTP_PORT=8080
GRPC_PORT=9090
LOG_LEVEL=info
ENVIRONMENT=production
```

**Windows** (`%USERPROFILE%\.sql-studio\config.env`):

```
TURSO_URL=libsql://your-db.turso.io
TURSO_AUTH_TOKEN=your-token
JWT_SECRET=your-secret-min-32-chars
```

### Configuration File

Alternatively, use YAML config (`~/.sql-studio/config.yaml`):

```yaml
server:
  http_port: 8080
  grpc_port: 9090

database:
  turso_url: "libsql://your-db.turso.io"
  turso_auth_token: "your-token"
  max_connections: 25

auth:
  jwt_secret: "your-secret"
  jwt_expiration: 24h

log:
  level: info
  format: json
```

## Verification

After installation, verify SQL Studio is working:

```bash
# Check version
sql-studio --version

# Expected output:
# SQL Studio Backend
# Version:    2.1.0
# Commit:     abc1234
# Build Date: 2025-10-23T10:30:00Z

# Start server (development mode)
sql-studio

# In another terminal, test health endpoint
curl http://localhost:8080/health

# Expected: {"status":"ok"}
```

## Upgrading

### Using Installer Script

```bash
# Upgrade to latest
curl -fsSL https://install.sqlstudio.io/install.sh | bash

# Upgrade to specific version
curl -fsSL https://install.sqlstudio.io/install.sh | bash -s -- --version 2.2.0
```

### Manual Upgrade

```bash
# Stop running instance
killall sql-studio  # or stop systemd service

# Download new version (same steps as installation)
VERSION=2.2.0
# ... download steps ...

# Replace binary
sudo mv sql-studio-${PLATFORM} /usr/local/bin/sql-studio

# Restart
sql-studio --version
```

### Docker Upgrade

```bash
# Pull latest
docker pull sqlstudio/backend:latest

# Stop and remove old container
docker stop sql-studio
docker rm sql-studio

# Start new container
docker run -d \
  --name sql-studio \
  -p 8080:8080 \
  -v sql-studio-data:/data \
  sqlstudio/backend:latest
```

## Uninstallation

### Remove Binary

```bash
# Remove installed binary
sudo rm /usr/local/bin/sql-studio

# Remove configuration (optional)
rm -rf ~/.sql-studio
```

### Remove Docker

```bash
# Stop and remove container
docker stop sql-studio
docker rm sql-studio

# Remove image
docker rmi sqlstudio/backend:latest

# Remove volume (optional - deletes data)
docker volume rm sql-studio-data
```

### Remove Systemd Service (Linux)

```bash
# Stop service
sudo systemctl stop sql-studio

# Disable service
sudo systemctl disable sql-studio

# Remove service file
sudo rm /etc/systemd/system/sql-studio.service

# Reload systemd
sudo systemctl daemon-reload
```

## Troubleshooting

### Binary Won't Run

**macOS**: "cannot be opened because it is from an unidentified developer"

```bash
sudo xattr -d com.apple.quarantine /usr/local/bin/sql-studio
```

**Linux**: Permission denied

```bash
sudo chmod +x /usr/local/bin/sql-studio
```

**Windows**: "The system cannot execute the specified program"

- Ensure you have Visual C++ Redistributable installed
- Run as Administrator

### Connection Issues

```bash
# Check if service is running
curl http://localhost:8080/health

# Check logs
LOG_LEVEL=debug sql-studio

# Check ports are not in use
lsof -i :8080  # macOS/Linux
netstat -ano | findstr :8080  # Windows
```

### Database Connection Fails

```bash
# Verify Turso credentials
echo $TURSO_URL
echo $TURSO_AUTH_TOKEN

# Test connection manually
sqlite3 "$(echo $TURSO_URL | sed 's|libsql://||')"
```

### Missing Dependencies (Linux)

```bash
# Check dependencies
ldd /usr/local/bin/sql-studio

# Install missing libraries (Ubuntu/Debian)
sudo apt-get install -y libsqlite3-0

# Install missing libraries (RHEL/CentOS)
sudo yum install -y sqlite-libs
```

### Port Already in Use

```bash
# Change ports via environment variables
HTTP_PORT=8081 GRPC_PORT=9091 sql-studio

# Or in config file
vim ~/.sql-studio/config.yaml
```

## System Requirements

- **OS**: macOS 10.15+, Linux (kernel 3.10+), Windows 10+
- **Architecture**: AMD64 (x86_64) or ARM64
- **RAM**: Minimum 256MB, recommended 512MB+
- **Disk**: 50MB for binary, additional for data
- **Network**: Internet connection for Turso (cloud mode)

## Additional Resources

- [Release Process](RELEASE.md)
- [Deployment Guide](backend-go/DEPLOYMENT.md)
- [Configuration Reference](backend-go/README.md)
- [API Documentation](backend-go/API_DOCUMENTATION.md)

## Getting Help

- Documentation: https://docs.sqlstudio.io
- GitHub Issues: https://github.com/yourusername/sql-studio/issues
- Discussions: https://github.com/yourusername/sql-studio/discussions
- Email: support@sqlstudio.io

## License

SQL Studio is released under the MIT License. See [LICENSE](LICENSE) for details.
