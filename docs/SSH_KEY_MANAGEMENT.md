# SSH Key Management Guide

## Overview

This guide explains how to manage SSH keys for secure database connections through bastion hosts in Howlerops. SSH keys provide stronger authentication than passwords and are essential for automated connections.

## SSH Key Types

### RSA Keys
- **Algorithm**: RSA
- **Key sizes**: 2048, 3072, 4096 bits (minimum 2048 recommended)
- **Format**: PEM or OpenSSH
- **Use case**: General purpose, widely supported

### DSA Keys
- **Algorithm**: DSA
- **Key size**: 1024 bits (deprecated, not recommended)
- **Format**: PEM
- **Use case**: Legacy systems only

### ECDSA Keys
- **Algorithm**: Elliptic Curve DSA
- **Key sizes**: 256, 384, 521 bits
- **Format**: PEM or OpenSSH
- **Use case**: Modern systems, smaller key sizes

### Ed25519 Keys
- **Algorithm**: Ed25519
- **Key size**: 256 bits
- **Format**: OpenSSH
- **Use case**: Modern systems, high performance

### OpenSSH Keys
- **Format**: OpenSSH private key format
- **Algorithm**: Any supported by OpenSSH
- **Use case**: Modern OpenSSH clients

## Generating SSH Keys

### RSA Key (Recommended)
```bash
# Generate 4096-bit RSA key
ssh-keygen -t rsa -b 4096 -f ~/.ssh/sql-studio-rsa -C "sql-studio@example.com"

# Generate with passphrase (recommended)
ssh-keygen -t rsa -b 4096 -f ~/.ssh/sql-studio-rsa -C "sql-studio@example.com" -N "your-passphrase"
```

### Ed25519 Key (Modern)
```bash
# Generate Ed25519 key
ssh-keygen -t ed25519 -f ~/.ssh/sql-studio-ed25519 -C "sql-studio@example.com"

# Generate with passphrase
ssh-keygen -t ed25519 -f ~/.ssh/sql-studio-ed25519 -C "sql-studio@example.com" -N "your-passphrase"
```

### ECDSA Key
```bash
# Generate 384-bit ECDSA key
ssh-keygen -t ecdsa -b 384 -f ~/.ssh/sql-studio-ecdsa -C "sql-studio@example.com"
```

## Key File Formats

### PEM Format (Traditional)
```
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA...
[base64 encoded key data]
...
-----END RSA PRIVATE KEY-----
```

### OpenSSH Format (Modern)
```
-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAA...
[base64 encoded key data]
...
-----END OPENSSH PRIVATE KEY-----
```

### PKCS#8 Format
```
-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQC...
[base64 encoded key data]
...
-----END PRIVATE KEY-----
```

## Key Management Best Practices

### 1. Key Naming Convention
Use descriptive names that indicate the purpose and environment:

```bash
# Good naming
~/.ssh/sql-studio-prod-rsa
~/.ssh/sql-studio-staging-ed25519
~/.ssh/aws-bastion-prod-rsa
~/.ssh/gcp-jumpbox-dev-ed25519

# Avoid generic names
~/.ssh/id_rsa
~/.ssh/key
~/.ssh/private_key
```

### 2. Key Permissions
Set proper file permissions for security:

```bash
# Private key (read-only for owner)
chmod 600 ~/.ssh/sql-studio-rsa

# Public key (readable by owner and group)
chmod 644 ~/.ssh/sql-studio-rsa.pub

# SSH directory (owner only)
chmod 700 ~/.ssh
```

### 3. Key Storage Organization
Organize keys by environment and purpose:

```
~/.ssh/
├── sql-studio/
│   ├── prod-rsa
│   ├── prod-rsa.pub
│   ├── staging-ed25519
│   └── staging-ed25519.pub
├── aws/
│   ├── bastion-prod-rsa
│   └── bastion-prod-rsa.pub
└── gcp/
    ├── jumpbox-dev-ed25519
    └── jumpbox-dev-ed25519.pub
```

### 4. Passphrase Protection
Always use passphrases for private keys:

```bash
# Add passphrase to existing key
ssh-keygen -p -f ~/.ssh/sql-studio-rsa

# Remove passphrase (not recommended)
ssh-keygen -p -f ~/.ssh/sql-studio-rsa -N ""
```

## Using Keys in Howlerops

### 1. Upload Key File
1. Open connection manager
2. Select "Use SSH Tunnel"
3. Choose "Private Key" authentication
4. Click "Upload File" tab
5. Select your private key file
6. Enter passphrase if key is encrypted

### 2. Paste Key Content
1. Open connection manager
2. Select "Use SSH Tunnel"
3. Choose "Private Key" authentication
4. Click "Paste Key" tab
5. Paste the private key content
6. Enter passphrase if key is encrypted

### 3. Key Validation
Howlerops validates keys before storing:
- **Format validation**: Ensures proper PEM/OpenSSH format
- **Key type detection**: Identifies RSA, DSA, EC, Ed25519
- **Encryption check**: Detects if key is passphrase-protected
- **Fingerprint generation**: Creates unique identifier

## Key Rotation

### 1. Generate New Key
```bash
# Generate new key with different name
ssh-keygen -t ed25519 -f ~/.ssh/sql-studio-new-ed25519 -C "sql-studio@example.com"
```

### 2. Deploy Public Key
```bash
# Copy public key to bastion host
ssh-copy-id -i ~/.ssh/sql-studio-new-ed25519.pub user@bastion.example.com

# Or manually append to authorized_keys
cat ~/.ssh/sql-studio-new-ed25519.pub | ssh user@bastion.example.com 'cat >> ~/.ssh/authorized_keys'
```

### 3. Test New Key
```bash
# Test SSH connection with new key
ssh -i ~/.ssh/sql-studio-new-ed25519 user@bastion.example.com
```

### 4. Update Howlerops
1. Update connection with new key
2. Test connection
3. Remove old key from bastion host
4. Delete old key files

## Troubleshooting

### Common Issues

#### 1. "Permission denied (publickey)"
**Causes:**
- Wrong private key file
- Incorrect file permissions
- Key not in bastion's authorized_keys
- Wrong username

**Solutions:**
```bash
# Check file permissions
ls -la ~/.ssh/sql-studio-rsa

# Test SSH connection manually
ssh -i ~/.ssh/sql-studio-rsa -v user@bastion.example.com

# Check authorized_keys on bastion
ssh user@bastion.example.com 'cat ~/.ssh/authorized_keys'
```

#### 2. "Could not parse private key"
**Causes:**
- Corrupted key file
- Wrong key format
- Missing BEGIN/END markers

**Solutions:**
```bash
# Validate key format
openssl rsa -in ~/.ssh/sql-studio-rsa -check -noout

# Convert key format if needed
ssh-keygen -p -f ~/.ssh/sql-studio-rsa -m pem
```

#### 3. "Host key verification failed"
**Causes:**
- Bastion host key changed
- Missing host in known_hosts

**Solutions:**
```bash
# Remove old host key
ssh-keygen -R bastion.example.com

# Add new host key
ssh-keyscan bastion.example.com >> ~/.ssh/known_hosts

# Or connect once to add host
ssh user@bastion.example.com
```

#### 4. "Key is encrypted but no passphrase provided"
**Causes:**
- Key has passphrase but none entered
- Wrong passphrase

**Solutions:**
- Enter correct passphrase in Howlerops
- Or remove passphrase: `ssh-keygen -p -f keyfile -N ""`

### Debugging Commands

#### Test SSH Connection
```bash
# Verbose SSH connection
ssh -i ~/.ssh/sql-studio-rsa -v user@bastion.example.com

# Test with specific port
ssh -i ~/.ssh/sql-studio-rsa -p 2222 -v user@bastion.example.com
```

#### Validate Key
```bash
# Check RSA key
openssl rsa -in ~/.ssh/sql-studio-rsa -check -noout

# Check Ed25519 key
ssh-keygen -l -f ~/.ssh/sql-studio-ed25519

# Get key fingerprint
ssh-keygen -l -f ~/.ssh/sql-studio-rsa.pub
```

#### Check Key Format
```bash
# Identify key type
file ~/.ssh/sql-studio-rsa

# View key header
head -1 ~/.ssh/sql-studio-rsa
```

## Security Considerations

### 1. Key Storage
- **Never share private keys**: Private keys should never be shared or transmitted
- **Secure storage**: Store keys in encrypted directories or use key management systems
- **Backup strategy**: Keep encrypted backups of important keys

### 2. Key Lifecycle
- **Regular rotation**: Rotate keys periodically (every 6-12 months)
- **Revocation**: Immediately revoke compromised keys
- **Audit trail**: Track key usage and access

### 3. Access Control
- **Principle of least privilege**: Use separate keys for different environments
- **Key restrictions**: Use SSH key restrictions when possible
- **Monitoring**: Monitor SSH access logs

### 4. Compliance
- **Key management policies**: Follow organizational key management policies
- **Audit requirements**: Ensure keys meet audit requirements
- **Retention policies**: Follow key retention and destruction policies

## Advanced Configuration

### SSH Config File
Create `~/.ssh/config` for easier management:

```
Host sql-studio-prod
    HostName bastion.example.com
    User ubuntu
    Port 22
    IdentityFile ~/.ssh/sql-studio-prod-rsa
    StrictHostKeyChecking yes
    UserKnownHostsFile ~/.ssh/known_hosts

Host sql-studio-staging
    HostName staging.bastion.example.com
    User ubuntu
    IdentityFile ~/.ssh/sql-studio-staging-ed25519
    StrictHostKeyChecking yes
```

### Key Restrictions
Add restrictions to public keys in `authorized_keys`:

```
# Restrict to specific commands
command="echo 'Only port forwarding allowed'",no-pty,no-X11-forwarding,permitopen="database.internal:5432" ssh-rsa AAAAB3...

# Restrict to specific source IPs
from="192.168.1.0/24" ssh-rsa AAAAB3...

# Time-based restrictions
expiry-time="20241231" ssh-rsa AAAAB3...
```

### Agent Forwarding
Use SSH agent for key management:

```bash
# Add key to agent
ssh-add ~/.ssh/sql-studio-rsa

# List loaded keys
ssh-add -l

# Remove key from agent
ssh-add -d ~/.ssh/sql-studio-rsa
```

## Integration with Howlerops

### Key Import Process
1. **File upload**: Select private key file
2. **Validation**: Client-side validation of key format
3. **Encryption**: Key encrypted with user's master key
4. **Storage**: Encrypted key stored in connection_secrets table
5. **Usage**: Key decrypted only when establishing SSH tunnel

### Security Features
- **Client-side validation**: Keys validated before transmission
- **Encrypted storage**: Keys never stored in plaintext
- **Memory clearing**: Decrypted keys cleared after use
- **Audit logging**: All key operations logged

### Error Handling
- **Clear error messages**: Helpful error messages for common issues
- **Validation feedback**: Real-time validation of key format
- **Recovery options**: Options to fix common key issues

## Conclusion

Proper SSH key management is essential for secure database connections. By following the best practices outlined in this guide, you can ensure that your SSH keys are secure, well-organized, and properly integrated with Howlerops. Regular key rotation, proper permissions, and secure storage are key to maintaining a secure environment.
