# SQL Studio Installation - Quick Reference

## One-Line Install

```bash
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh
```

## Common Use Cases

| Task | Command |
|------|---------|
| **Install latest** | `curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh \| sh` |
| **Install v2.0.0** | `curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh \| sh -s -- --version v2.0.0` |
| **Preview (dry-run)** | `curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh \| sh -s -- --dry-run` |
| **Verbose output** | `curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh \| sh -s -- --verbose` |
| **Custom directory** | `curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh \| sh -s -- --install-dir ~/bin` |
| **Force reinstall** | `curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh \| sh -s -- --force` |
| **Help** | `curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh \| sh -s -- --help` |

## After Installation

```bash
# Verify installation
sql-studio --version

# View help
sql-studio --help

# Run SQL Studio
sql-studio
```

## Supported Platforms

✅ macOS (Intel & Apple Silicon)
✅ Linux (x86_64, ARM64, ARM)
✅ Windows (via Git Bash or WSL)

## Manual Installation

```bash
# 1. Download binary for your platform
# Visit: https://github.com/sql-studio/sql-studio/releases

# 2. Extract
tar -xzf sql-studio-*.tar.gz

# 3. Install
mv sql-studio ~/.local/bin/
chmod +x ~/.local/bin/sql-studio

# 4. Verify
sql-studio --version
```

## Uninstall

```bash
# Remove binary
rm ~/.local/bin/sql-studio
# or
sudo rm /usr/local/bin/sql-studio
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| **Command not found** | Add `~/.local/bin` to PATH: `echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc && source ~/.bashrc` |
| **Permission denied** | Make executable: `chmod +x ~/.local/bin/sql-studio` |
| **Download fails** | Check internet connection, try with `--version v2.0.0` |
| **Platform unsupported** | Check supported platforms above, open an issue on GitHub |

## Environment Variables

```bash
# Custom install location
export INSTALL_DIR="$HOME/bin"
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh

# Specific version
export VERSION="v2.0.0"
curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh
```

## For Developers

```bash
# CI/CD example (GitHub Actions)
- run: curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --version v2.0.0
- run: echo "$HOME/.local/bin" >> $GITHUB_PATH

# Docker example
RUN curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh
ENV PATH="/root/.local/bin:${PATH}"
```

## Links

- **Documentation:** https://docs.sqlstudio.io
- **Releases:** https://github.com/sql-studio/sql-studio/releases
- **Issues:** https://github.com/sql-studio/sql-studio/issues
- **Full Installation Guide:** [docs/INSTALLATION.md](docs/INSTALLATION.md)
