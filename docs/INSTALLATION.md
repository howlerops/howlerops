# SQL Studio Installation Guide

SQL Studio provides a universal installation script that works across macOS, Linux, and Windows (Git Bash/WSL).

## Quick Install

The fastest way to install SQL Studio is using our installation script:

```bash
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh
```

This will:
- Detect your OS and architecture
- Download the appropriate binary
- Verify checksums for security
- Install to `~/.local/bin` (or `/usr/local/bin` if needed)
- Update your PATH if necessary

## Installation Options

### Install a Specific Version

```bash
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --version v2.0.0
```

### Install to Custom Directory

```bash
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --install-dir /usr/local/bin
```

### Dry Run (Preview Without Installing)

```bash
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --dry-run
```

### Verbose Output

```bash
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --verbose
```

### Force Reinstall

```bash
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --force
```

## Supported Platforms

| Operating System | Architectures |
|-----------------|---------------|
| macOS (Darwin)  | x86_64 (Intel), arm64 (Apple Silicon) |
| Linux           | x86_64, arm64, arm |
| Windows         | x86_64, arm64 (via Git Bash/WSL) |

## Environment Variables

You can also configure the installation using environment variables:

```bash
# Set custom installation directory
export INSTALL_DIR="$HOME/bin"
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh

# Install specific version
export VERSION="v2.0.0"
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh
```

## Manual Installation

If you prefer to install manually:

1. **Download the binary for your platform:**

   Visit [GitHub Releases](https://github.com/sql-studio/sql-studio/releases) and download the appropriate archive:
   - macOS Intel: `sql-studio-darwin-amd64.tar.gz`
   - macOS Apple Silicon: `sql-studio-darwin-arm64.tar.gz`
   - Linux x86_64: `sql-studio-linux-amd64.tar.gz`
   - Linux ARM64: `sql-studio-linux-arm64.tar.gz`

2. **Verify the checksum (recommended):**

   ```bash
   # Download checksums.txt
   wget https://github.com/sql-studio/sql-studio/releases/download/v2.0.0/checksums.txt

   # Verify (macOS)
   shasum -a 256 -c checksums.txt

   # Verify (Linux)
   sha256sum -c checksums.txt
   ```

3. **Extract the archive:**

   ```bash
   tar -xzf sql-studio-*.tar.gz
   ```

4. **Move to a directory in your PATH:**

   ```bash
   mv sql-studio ~/.local/bin/
   # or
   sudo mv sql-studio /usr/local/bin/
   ```

5. **Make it executable:**

   ```bash
   chmod +x ~/.local/bin/sql-studio
   ```

## Updating PATH

If the installation directory is not in your PATH, add it to your shell profile:

### Bash

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

### Zsh

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

### Fish

```bash
set -Ua fish_user_paths $HOME/.local/bin
```

## Verifying Installation

After installation, verify it works:

```bash
# Check version
sql-studio --version

# View help
sql-studio --help

# Run SQL Studio
sql-studio
```

## Uninstalling

To remove SQL Studio:

```bash
# Remove the binary
rm ~/.local/bin/sql-studio
# or
sudo rm /usr/local/bin/sql-studio
```

## Troubleshooting

### Permission Denied

If you get a "permission denied" error:

```bash
# Make the binary executable
chmod +x ~/.local/bin/sql-studio
```

### Command Not Found

If you get "command not found":

1. Check if the binary is in your PATH:
   ```bash
   which sql-studio
   ```

2. If not found, add the installation directory to PATH (see above)

### Download Failures

If downloads fail:

- Check your internet connection
- Verify the repository has releases
- Try specifying a version explicitly: `--version v2.0.0`
- Use verbose mode to see detailed error messages: `--verbose`

### Platform Not Supported

If you see "Unsupported platform":

- Check the [supported platforms](#supported-platforms) list
- File an issue on [GitHub](https://github.com/sql-studio/sql-studio/issues) requesting support for your platform

### Checksum Verification Failed

If checksum verification fails:

- Try downloading again (may be a corrupted download)
- Verify you're downloading from the official GitHub releases
- Report the issue if it persists

## Advanced Usage

### Behind a Proxy

If you're behind a corporate proxy:

```bash
# For curl
export https_proxy=http://proxy.example.com:8080
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh

# For wget
export https_proxy=http://proxy.example.com:8080
wget -qO- https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh
```

### Air-Gapped Installation

For systems without internet access:

1. Download the binary and checksums on a connected machine
2. Transfer files to the air-gapped system
3. Follow the [manual installation](#manual-installation) steps

### CI/CD Integration

Example GitHub Actions workflow:

```yaml
- name: Install SQL Studio
  run: |
    curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --version v2.0.0
    echo "$HOME/.local/bin" >> $GITHUB_PATH
```

Example GitLab CI:

```yaml
install_sql_studio:
  script:
    - curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --version v2.0.0
    - export PATH="$HOME/.local/bin:$PATH"
```

## Docker Installation

For Docker users, use the official image:

```dockerfile
FROM sql-studio/sql-studio:latest
# or specific version
FROM sql-studio/sql-studio:v2.0.0
```

Or install in your own image:

```dockerfile
FROM ubuntu:22.04

RUN apt-get update && apt-get install -y curl ca-certificates && \
    curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

ENV PATH="/root/.local/bin:${PATH}"
```

## Security

The installation script:

- Downloads binaries only from official GitHub releases
- Verifies SHA256 checksums before installation
- Uses HTTPS for all downloads
- Requires minimal permissions (no sudo unless necessary)

For maximum security:

1. Review the installation script before running
2. Always verify checksums manually
3. Pin to specific versions in production
4. Use GPG signatures when available (coming soon)

## Getting Help

- **Documentation:** https://docs.sqlstudio.io
- **Issues:** https://github.com/sql-studio/sql-studio/issues
- **Discussions:** https://github.com/sql-studio/sql-studio/discussions
- **Discord:** https://discord.gg/sqlstudio (if applicable)

## License

SQL Studio is released under the MIT License. See [LICENSE](../LICENSE) for details.
