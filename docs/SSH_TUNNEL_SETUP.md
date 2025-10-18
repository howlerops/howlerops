# SSH Tunnel Setup Guide

This guide explains how to set up and use SSH tunnels for secure database connections through bastion hosts in HowlerOps.

## Overview

SSH tunneling (also known as SSH port forwarding) creates an encrypted connection between your computer and a database server through an intermediate bastion host. This is essential when:

- Your database is in a private network/VPC
- Direct database connections are blocked by firewall
- You need to access on-premise databases securely
- Security policies require all database traffic to be encrypted and audited

## Architecture

```
┌─────────────┐    SSH Tunnel     ┌──────────────┐    Private Network    ┌──────────────┐
│             │ ═══════════════> │              │ ───────────────────> │              │
│ HowlerOps   │   (Encrypted)     │ Bastion Host │                       │   Database   │
│             │ <═══════════════ │              │ <─────────────────── │   Server     │
└─────────────┘                   └──────────────┘                       └──────────────┘
  localhost                         Public IP                              Private IP
                                    (e.g., bastion.example.com)            (e.g., 10.0.1.50)
```

## Prerequisites

### 1. Bastion Host Access

You need:
- Hostname or IP address of the bastion host
- SSH credentials (username + password or private key)
- Network access to the bastion host (typically port 22)

### 2. SSH Key Pair (Recommended)

Generate an SSH key pair if you don't have one:

```bash
# Generate a new SSH key pair
ssh-keygen -t rsa -b 4096 -f ~/.ssh/bastion_key

# Set proper permissions
chmod 600 ~/.ssh/bastion_key
chmod 644 ~/.ssh/bastion_key.pub
```

### 3. Bastion Host Configuration

Copy your public key to the bastion host:

```bash
# Copy public key to bastion host
ssh-copy-id -i ~/.ssh/bastion_key.pub user@bastion.example.com

# Or manually append to authorized_keys
cat ~/.ssh/bastion_key.pub | ssh user@bastion.example.com 'cat >> ~/.ssh/authorized_keys'
```

### 4. Database Accessibility

Verify the database is accessible from the bastion host:

```bash
# SSH to bastion host
ssh user@bastion.example.com

# Test database connectivity
nc -zv database.internal 5432   # PostgreSQL
nc -zv database.internal 3306   # MySQL
nc -zv database.internal 27017  # MongoDB
```

## Configuration in HowlerOps

### Basic SSH Tunnel Setup

1. **Create a New Connection**
   - Click "Add Connection" in HowlerOps
   - Fill in database details (host, port, database, credentials)

2. **Enable SSH Tunnel**
   - Scroll to "SSH Tunnel Configuration"
   - Check "Use SSH Tunnel"

3. **Configure SSH Connection**
   - **SSH Host**: Enter bastion host address (e.g., `bastion.example.com`)
   - **SSH Port**: Default is 22 (change if your bastion uses a different port)
   - **SSH User**: Your username on the bastion host (e.g., `ubuntu`, `ec2-user`)

4. **Choose Authentication Method**

   **Option A: Password Authentication**
   - Select "Password" as auth method
   - Enter your SSH password
   - Note: Less secure, not recommended for production

   **Option B: Private Key Authentication** (Recommended)
   - Select "Private Key" as auth method
   - Choose one of two methods:
     - **File Path**: Enter path to your private key file (e.g., `~/.ssh/bastion_key`)
     - **Paste Key**: Click "Use Direct Key" and paste your private key content

5. **Configure Advanced Options** (Optional)
   - **Known Hosts Path**: Path to known_hosts file (e.g., `~/.ssh/known_hosts`)
   - **Strict Host Key Checking**: Enable to verify SSH host keys (recommended)
   - **Connection Timeout**: Seconds to wait for SSH connection (default: 30)
   - **Keep-Alive Interval**: Seconds between keep-alive packets (default: 30)

6. **Test the Connection**
   - Click "Test Connection"
   - Verify the test succeeds
   - Check the response time and server info

7. **Save the Connection**
   - Click "Add Connection" to save

## Common Setup Scenarios

### Scenario 1: AWS RDS via EC2 Bastion

**Architecture**:
- RDS database in private subnet
- EC2 bastion host in public subnet
- Security groups configured appropriately

**Configuration**:
```yaml
Database Configuration:
  Type: PostgreSQL
  Host: mydb.abc123.us-east-1.rds.amazonaws.com
  Port: 5432
  Database: production
  Username: dbuser
  Password: ********
  SSL Mode: require

SSH Tunnel Configuration:
  Use SSH Tunnel: ✓
  SSH Host: ec2-1-2-3-4.compute-1.amazonaws.com
  SSH Port: 22
  SSH User: ec2-user
  Auth Method: Private Key
  Private Key Path: ~/.ssh/aws-bastion.pem
```

**AWS Security Group Requirements**:
```
Bastion Host Security Group:
  Inbound: SSH (22) from your IP
  Outbound: PostgreSQL (5432) to RDS security group

RDS Security Group:
  Inbound: PostgreSQL (5432) from bastion security group
```

### Scenario 2: Azure Database via Jump Box

**Configuration**:
```yaml
Database Configuration:
  Type: MySQL
  Host: myserver.mysql.database.azure.com
  Port: 3306
  Database: myapp
  Username: admin@myserver
  Password: ********
  SSL Mode: require

SSH Tunnel Configuration:
  Use SSH Tunnel: ✓
  SSH Host: jumpbox.eastus.cloudapp.azure.com
  SSH Port: 22
  SSH User: azureuser
  Auth Method: Private Key
  Private Key Path: ~/.ssh/azure-jumpbox.pem
```

### Scenario 3: GCP Cloud SQL via Compute Instance

**Configuration**:
```yaml
Database Configuration:
  Type: PostgreSQL
  Host: 10.128.0.3
  Port: 5432
  Database: production
  Username: postgres
  Password: ********

SSH Tunnel Configuration:
  Use SSH Tunnel: ✓
  SSH Host: bastion.us-central1-a.my-project
  SSH Port: 22
  SSH User: user_gmail_com
  Auth Method: Private Key
  Private Key Path: ~/.ssh/google_compute_engine
```

### Scenario 4: On-Premise Database via Corporate VPN Bastion

**Configuration**:
```yaml
Database Configuration:
  Type: TiDB
  Host: 192.168.100.50
  Port: 4000
  Database: analytics
  Username: analyst
  Password: ********

SSH Tunnel Configuration:
  Use SSH Tunnel: ✓
  SSH Host: vpn-gateway.company.com
  SSH Port: 2222  # Custom SSH port
  SSH User: jdoe
  Auth Method: Private Key
  Private Key Path: ~/.ssh/company-vpn.pem
  Strict Host Key Checking: ✓
  Known Hosts Path: ~/.ssh/known_hosts
```

### Scenario 5: MongoDB Replica Set via Bastion

**Configuration**:
```yaml
Database Configuration:
  Type: MongoDB
  Connection String: mongodb://10.0.1.50:27017,10.0.1.51:27017,10.0.1.52:27017/myapp?replicaSet=rs0
  Auth Database: admin
  Username: appuser
  Password: ********

SSH Tunnel Configuration:
  Use SSH Tunnel: ✓
  SSH Host: bastion.internal.company.com
  SSH Port: 22
  SSH User: ubuntu
  Auth Method: Private Key
  Private Key Path: ~/.ssh/mongo-bastion.pem
  Keep-Alive Interval: 60  # Longer interval for replica set
```

## Private Key Formats

HowlerOps supports standard SSH private key formats:

### PEM Format (RSA)
```
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA...
...
-----END RSA PRIVATE KEY-----
```

### OpenSSH Format
```
-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAA...
...
-----END OPENSSH PRIVATE KEY-----
```

### Converting Keys

If you have a different key format:

```bash
# Convert PuTTY PPK to OpenSSH format
puttygen key.ppk -O private-openssh -o key.pem

# Convert PKCS#8 to PEM format
openssl pkcs8 -in key.pk8 -out key.pem
```

## Security Best Practices

### 1. Key Management

```bash
# Proper key permissions (required)
chmod 600 ~/.ssh/private_key.pem

# Keep keys organized
mkdir -p ~/.ssh/keys
mv *.pem ~/.ssh/keys/
chmod 700 ~/.ssh/keys
```

### 2. SSH Config File

Create `~/.ssh/config` for easier management:

```
Host prod-bastion
    HostName bastion.example.com
    User ubuntu
    Port 22
    IdentityFile ~/.ssh/prod-bastion.pem
    StrictHostKeyChecking yes
    UserKnownHostsFile ~/.ssh/known_hosts

Host staging-bastion
    HostName staging.bastion.example.com
    User ubuntu
    IdentityFile ~/.ssh/staging-bastion.pem
    StrictHostKeyChecking yes
```

Then use the alias in HowlerOps: `prod-bastion`

### 3. Known Hosts Verification

Enable strict host key checking to prevent MITM attacks:

```bash
# Add bastion host to known_hosts
ssh-keyscan bastion.example.com >> ~/.ssh/known_hosts

# Or connect once manually
ssh user@bastion.example.com
# Type 'yes' when prompted to add to known_hosts
```

In HowlerOps:
- Enable "Strict Host Key Checking"
- Set "Known Hosts Path" to `~/.ssh/known_hosts`

### 4. Least Privilege

On the bastion host, limit SSH key permissions:

```bash
# In ~/.ssh/authorized_keys, add restrictions
command="echo 'Only port forwarding allowed'",no-pty,no-X11-forwarding,permitopen="database.internal:5432" ssh-rsa AAAAB3...
```

This allows only port forwarding to specific destinations.

### 5. Audit and Monitoring

- Enable SSH logging on bastion hosts
- Monitor SSH connection attempts
- Rotate SSH keys regularly
- Use session recording tools for compliance

### 6. Multi-Factor Authentication

For enhanced security, configure MFA on bastion hosts:

```bash
# Install Google Authenticator PAM module
sudo apt-get install libpam-google-authenticator

# Configure SSH to use it
# Edit /etc/pam.d/sshd
auth required pam_google_authenticator.so

# Edit /etc/ssh/sshd_config
ChallengeResponseAuthentication yes
```

## Troubleshooting

### Connection Refused

**Symptoms**: "Connection refused" or "No route to host"

**Solutions**:
1. Verify SSH port is correct (default 22)
2. Check bastion host is accessible: `ping bastion.example.com`
3. Test SSH manually: `ssh user@bastion.example.com`
4. Check firewall rules allow SSH from your IP
5. Verify SSH daemon is running on bastion

### Permission Denied (publickey)

**Symptoms**: "Permission denied (publickey)" error

**Solutions**:
1. Verify private key file path is correct
2. Check key file permissions: `chmod 600 key.pem`
3. Ensure public key is in bastion's `~/.ssh/authorized_keys`
4. Test SSH connection manually: `ssh -i key.pem user@bastion`
5. Check SSH logs on bastion: `sudo tail -f /var/log/auth.log`

### Host Key Verification Failed

**Symptoms**: "Host key verification failed"

**Solutions**:
1. Add bastion to known_hosts: `ssh-keyscan bastion.example.com >> ~/.ssh/known_hosts`
2. Remove old key if bastion was rebuilt: `ssh-keygen -R bastion.example.com`
3. Disable strict checking temporarily to add host
4. Verify you're connecting to the correct host

### Database Connection Timeout After Tunnel Established

**Symptoms**: SSH tunnel connects but database connection times out

**Solutions**:
1. Verify database is reachable from bastion:
   ```bash
   ssh user@bastion.example.com
   nc -zv database.internal 5432
   ```
2. Check database firewall rules allow traffic from bastion
3. Verify database host/port are correct in configuration
4. Check security groups (cloud environments)
5. Ensure database is running and accepting connections

### Tunnel Keeps Dropping

**Symptoms**: Connection works initially but drops after some time

**Solutions**:
1. Increase keep-alive interval (e.g., 30 seconds)
2. Check bastion SSH config: `ClientAliveInterval` and `ClientAliveCountMax`
3. Adjust connection timeout settings
4. Verify network stability between your machine and bastion
5. Check bastion resource utilization (CPU, memory, network)

### Private Key Format Issues

**Symptoms**: "Invalid key format" or "Could not parse private key"

**Solutions**:
1. Verify key is in PEM or OpenSSH format
2. Check key file is complete (no truncation)
3. Ensure no extra whitespace or characters
4. Convert key format if necessary:
   ```bash
   ssh-keygen -p -f oldkey -m pem -N "" -o newkey
   ```

## Performance Optimization

### Keep-Alive Settings

For long-running queries or idle connections:

```yaml
Keep-Alive Interval: 30 seconds  # Send keep-alive every 30s
Connection Timeout: 60 seconds   # Wait up to 60s for connection
```

### Connection Pooling

Adjust based on SSH tunnel overhead:

```yaml
Max Connections: 10       # Lower for SSH tunnels (vs 25 for direct)
Idle Timeout: 5 minutes  # Close idle connections sooner
```

### Compression

For high-latency connections, enable SSH compression:

Add to `~/.ssh/config`:
```
Host bastion.example.com
    Compression yes
    CompressionLevel 6
```

## Testing Your Setup

### 1. Test SSH Connection

```bash
# Test basic SSH connectivity
ssh user@bastion.example.com echo "SSH connection works"

# Test with specific key
ssh -i ~/.ssh/key.pem user@bastion.example.com echo "SSH connection works"
```

### 2. Test Manual Port Forwarding

```bash
# Create manual SSH tunnel
ssh -L 5432:database.internal:5432 user@bastion.example.com

# In another terminal, test database connection
psql -h localhost -p 5432 -U dbuser -d production
```

### 3. Test Database Connectivity from Bastion

```bash
# SSH to bastion
ssh user@bastion.example.com

# Test PostgreSQL
nc -zv database.internal 5432

# Test MySQL
nc -zv database.internal 3306

# Test MongoDB
nc -zv database.internal 27017

# Full connection test
psql -h database.internal -p 5432 -U dbuser -d production -c "SELECT 1"
```

### 4. Test in HowlerOps

1. Configure the connection as described
2. Click "Test Connection"
3. Verify:
   - Status shows "Connected"
   - Response time is reasonable (< 1000ms typically)
   - Database version is displayed
   - Server info is populated

## Advanced Configuration

### Multiple Hops (Jump Hosts)

For environments requiring multiple jump hosts:

```bash
# In ~/.ssh/config
Host final-destination
    HostName database.internal
    User dbuser
    ProxyJump jump1.example.com,jump2.example.com
```

### Dynamic Port Forwarding

For more flexible tunneling:

```bash
# Create SOCKS proxy
ssh -D 1080 -N user@bastion.example.com

# Configure application to use SOCKS proxy
# HowlerOps doesn't currently support SOCKS proxies
```

### SSH Agent Forwarding

To use SSH agent for key management:

```bash
# Add key to SSH agent
ssh-add ~/.ssh/key.pem

# Enable agent forwarding in ~/.ssh/config
Host bastion.example.com
    ForwardAgent yes
```

## Monitoring and Maintenance

### Connection Health

Monitor SSH tunnel health in HowlerOps:
- Check connection status indicator
- Review response times in query history
- Monitor for connection drops or timeouts

### Log Analysis

Check SSH logs on bastion host:

```bash
# View SSH authentication logs
sudo tail -f /var/log/auth.log | grep sshd

# View connection statistics
sudo systemctl status sshd

# Check active SSH connections
sudo netstat -tnpa | grep :22
```

### Bastion Host Maintenance

Regular maintenance tasks:

1. **Update SSH Server**:
   ```bash
   sudo apt-get update
   sudo apt-get upgrade openssh-server
   ```

2. **Review Authorized Keys**:
   ```bash
   cat ~/.ssh/authorized_keys
   # Remove any old or unauthorized keys
   ```

3. **Monitor Resources**:
   ```bash
   # Check CPU and memory
   top

   # Check network connections
   netstat -an | grep ESTABLISHED | wc -l
   ```

4. **Rotate Logs**:
   ```bash
   # Configure logrotate for auth.log
   sudo nano /etc/logrotate.d/rsyslog
   ```

## Migration Checklist

When moving from direct connections to SSH tunnels:

- [ ] Test SSH access to bastion host
- [ ] Verify database accessibility from bastion
- [ ] Configure security groups/firewall rules
- [ ] Set up SSH key-based authentication
- [ ] Add bastion to known_hosts
- [ ] Test manual SSH tunnel creation
- [ ] Configure SSH tunnel in HowlerOps
- [ ] Test connection in HowlerOps
- [ ] Update documentation for team
- [ ] Train team members on new connection method
- [ ] Monitor for issues during transition period
- [ ] Remove old direct connection firewall rules

## Additional Resources

- [SSH Tunneling Explained](https://www.ssh.com/academy/ssh/tunneling)
- [AWS Bastion Host Best Practices](https://aws.amazon.com/blogs/security/how-to-record-ssh-sessions-established-through-a-bastion-host/)
- [Azure Jump Box Setup](https://docs.microsoft.com/en-us/azure/architecture/reference-architectures/n-tier/multi-region-sql-server)
- [GCP IAP TCP Forwarding](https://cloud.google.com/iap/docs/tcp-forwarding-overview)
- [SSH Key Management Best Practices](https://www.ssh.com/academy/ssh/key-management)

## Support

If you encounter issues:
1. Review this troubleshooting section
2. Check SSH and application logs
3. Test SSH connection manually
4. Verify network connectivity and permissions
5. Contact your infrastructure team for bastion access issues
