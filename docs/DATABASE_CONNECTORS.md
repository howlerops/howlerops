# Database Connectors Guide

HowlerOps now supports 10 different database types with comprehensive connection management, including secure SSH tunnel and VPC networking options.

## Supported Database Types

### Relational Databases
- **PostgreSQL** - Full support for PostgreSQL 9.6+
- **MySQL** - MySQL 5.7+ and MySQL 8.0+
- **MariaDB** - MariaDB 10.3+
- **TiDB** - Distributed MySQL-compatible database with HTAP capabilities

### NoSQL Databases
- **MongoDB** - Document database with schema inference (4.0+)

### OLAP Databases
- **ClickHouse** - Columnar database for analytics

### Search Engines
- **Elasticsearch** - Distributed search and analytics engine (7.x, 8.x)
- **OpenSearch** - Open-source fork of Elasticsearch

### Embedded Databases
- **SQLite** - Serverless, file-based database

## Connection Configuration

### Basic Connection

All database connections require these basic fields:

- **Name**: A friendly name for your connection
- **Database Type**: Select from supported database types
- **Host**: Database server hostname or IP address
- **Port**: Database server port (auto-populated with defaults)
- **Database**: Database/schema name
- **Username**: Database username
- **Password**: Database password

### SSL/TLS Configuration

For secure connections, configure SSL mode:

- **disable**: No SSL encryption
- **prefer**: Use SSL if available (default for most databases)
- **require**: Always use SSL, fail if not available
- **verify-ca**: Verify the CA certificate
- **verify-full**: Verify CA and hostname

## SSH Tunnel Configuration

SSH tunnels allow you to securely connect to databases through a bastion host or jump server. This is essential for accessing databases in private networks.

### When to Use SSH Tunnels

- Database is in a private VPC/network
- Direct database access is restricted
- You need to connect through a bastion host
- Compliance requires all database connections to be tunneled

### Configuration Steps

1. **Enable SSH Tunnel**: Check the "Use SSH Tunnel" option in the connection form

2. **SSH Server Details**:
   - **SSH Host**: Bastion host hostname or IP
   - **SSH Port**: SSH port (default: 22)
   - **SSH User**: Username on the bastion host

3. **Authentication Method**:

   **Option A: Password Authentication**
   - Select "Password" as the auth method
   - Enter your SSH password

   **Option B: Private Key Authentication** (Recommended)
   - Select "Private Key" as the auth method
   - Provide your private key by either:
     - **File Path**: Path to your private key file (e.g., `~/.ssh/id_rsa`)
     - **Direct Key**: Paste the private key content directly

4. **Advanced Options** (Optional):
   - **Known Hosts Path**: Path to SSH known_hosts file for host verification
   - **Strict Host Key Checking**: Enable to enforce SSH host key verification
   - **Connection Timeout**: SSH connection timeout in seconds (default: 30)
   - **Keep-Alive Interval**: Send keep-alive packets every N seconds (default: 30)

### Example: PostgreSQL via SSH Tunnel

```yaml
Connection Name: Production Database
Database Type: PostgreSQL
Host: 10.0.1.50          # Private IP in VPC
Port: 5432
Database: myapp_production
Username: app_user
Password: ********

SSH Tunnel Configuration:
  Use SSH Tunnel: ✓
  SSH Host: bastion.example.com
  SSH Port: 22
  SSH User: ubuntu
  Auth Method: Private Key
  Private Key Path: ~/.ssh/production.pem
  Strict Host Key Checking: ✓
```

### SSH Tunnel Flow

```
Your Computer → SSH Tunnel → Bastion Host → Database Server
                (Encrypted)                  (Private Network)
```

The application automatically:
1. Establishes SSH connection to bastion host
2. Creates a local port forward
3. Connects to the database through the tunnel
4. Maintains the tunnel with keep-alive packets
5. Handles reconnection if the tunnel drops

## VPC Configuration

VPC configuration allows direct connections to databases within a Virtual Private Cloud using AWS PrivateLink, Azure Private Link, or GCP Private Service Connect.

### VPC Settings

- **VPC ID**: Your VPC identifier
- **Subnet ID**: Subnet where the database resides
- **Security Group IDs**: Comma-separated list of security groups
- **Private Link Service**: AWS PrivateLink service name
- **Endpoint Service Name**: Service endpoint for private connections

### Example: RDS Database with VPC

```yaml
Connection Name: RDS PostgreSQL
Database Type: PostgreSQL
Host: postgres.vpc.internal
Port: 5432

VPC Configuration:
  Use VPC: ✓
  VPC ID: vpc-1234567890abcdef0
  Subnet ID: subnet-0bb1c79de3456789a
  Security Group IDs: sg-12345678, sg-87654321
  Private Link Service: com.amazonaws.vpce.us-east-1.vpce-svc-12345
```

## Database-Specific Configuration

### MongoDB

MongoDB connections support additional configuration:

- **Connection String**: Optional MongoDB URI format (mongodb://...)
- **Authentication Database**: Database used for authentication (default: "admin")
- **Username/Database**: Optional for MongoDB, use connection string instead

**Example Connection String**:
```
mongodb://username:password@host:27017/database?authSource=admin&replicaSet=rs0
```

**Schema Inference**: HowlerOps automatically samples the first 100 documents from each collection to infer the schema and field types.

### Elasticsearch / OpenSearch

- **Scheme**: HTTP or HTTPS
- **API Key**: Optional API key for authentication (alternative to username/password)
- **Database**: Optional index pattern (e.g., "logs-*")

**SQL API**: HowlerOps uses the Elasticsearch `_sql` API for query execution, allowing SQL queries against your indices.

### ClickHouse

- **Native Protocol**: Check to use native ClickHouse protocol (port 9000) instead of HTTP
- **Engine Metadata**: Table metadata includes ClickHouse-specific engine information (MergeTree, ReplicatedMergeTree, etc.)

**Features**:
- Streaming query results for large datasets
- Table engine metadata
- Database statistics

### TiDB

TiDB is MySQL-compatible with additional distributed database features:

- **Default Port**: 4000 (instead of MySQL's 3306)
- **TiDB Version Info**: Connection info includes TiDB-specific version
- **TiFlash Support**: Metadata shows TiFlash replica information
- **TiKV Regions**: Access to TiKV region statistics

**HTAP Capabilities**: TiDB supports both transactional (TiKV) and analytical (TiFlash) workloads.

## Connection Pooling

All database connections use connection pooling for optimal performance:

- **Max Connections**: Maximum number of open connections (default: 25)
- **Max Idle Connections**: Maximum idle connections to maintain (default: 5)
- **Idle Timeout**: Close idle connections after this duration (default: 5 minutes)
- **Connection Lifetime**: Maximum lifetime of a connection (default: 1 hour)

These can be configured per connection using the "Advanced Options" in the connection form.

## Testing Connections

Before saving a connection, use the "Test Connection" button to verify:

- Network connectivity
- Authentication credentials
- Database permissions
- SSL/TLS configuration
- SSH tunnel establishment (if configured)

The test returns:
- Connection status (success/failure)
- Response time in milliseconds
- Database version
- Server information

## Security Best Practices

### SSH Tunnels

1. **Use Private Key Authentication**: More secure than passwords
2. **Protect Private Keys**: Use `chmod 600 ~/.ssh/your_key.pem`
3. **Enable Strict Host Key Checking**: Prevents MITM attacks
4. **Use Known Hosts**: Maintain a known_hosts file
5. **Rotate Keys Regularly**: Update SSH keys periodically

### Database Credentials

1. **Least Privilege**: Grant only necessary permissions
2. **Read-Only Access**: Use read-only users for reporting/analytics
3. **Credential Rotation**: Rotate database passwords regularly
4. **Never Commit Credentials**: Keep credentials out of version control

### Network Security

1. **Use SSL/TLS**: Enable SSL for all production databases
2. **Verify Certificates**: Use `verify-ca` or `verify-full` SSL modes
3. **Private Networks**: Use VPC or SSH tunnels for databases in private networks
4. **Firewall Rules**: Restrict database access to known IP ranges
5. **Security Groups**: Configure VPC security groups properly

## Troubleshooting

### SSH Tunnel Connection Fails

**Problem**: Cannot establish SSH tunnel

**Solutions**:
- Verify SSH host is reachable: `ssh user@bastion-host`
- Check SSH port (default 22)
- Verify private key permissions: `chmod 600 key.pem`
- Ensure private key format is correct (PEM format)
- Check bastion host firewall rules
- Verify SSH user has access to bastion host

### Database Connection Times Out

**Problem**: Connection times out after SSH tunnel established

**Solutions**:
- Verify database host/port from bastion: `nc -zv db-host 5432`
- Check database firewall rules
- Verify security group allows traffic from bastion
- Ensure database is running and accepting connections
- Check database authentication credentials

### SSL/TLS Certificate Errors

**Problem**: SSL certificate verification fails

**Solutions**:
- Use `ssl_mode=require` instead of `verify-full` for self-signed certs
- Install CA certificate on your system
- Verify certificate hostname matches connection host
- Check certificate expiration date

### MongoDB Schema Not Loading

**Problem**: Collections show but no schema information

**Solutions**:
- Ensure database user has read permissions
- Check that collections contain documents
- Verify MongoDB version is 4.0+
- Collections with no documents won't show schema

### ClickHouse Connection Issues

**Problem**: Cannot connect to ClickHouse

**Solutions**:
- Try both HTTP (8123) and native protocol (9000) ports
- Verify ClickHouse user has correct permissions
- Check `users.xml` for allowed IP ranges
- Ensure database exists and user has access

## Performance Optimization

### Connection Pooling

Adjust pool settings based on workload:

**High-Concurrency Applications**:
```yaml
Max Connections: 50
Max Idle Connections: 10
Idle Timeout: 10 minutes
```

**Low-Traffic Applications**:
```yaml
Max Connections: 10
Max Idle Connections: 2
Idle Timeout: 2 minutes
```

### SSH Tunnel Keep-Alive

For long-running connections, configure keep-alive:

```yaml
Keep-Alive Interval: 30 seconds  # Prevents tunnel timeout
Connection Timeout: 60 seconds   # Allow time for tunnel establishment
```

### ClickHouse Optimization

- Use native protocol for better performance
- Enable compression for large result sets
- Configure appropriate buffer sizes

### MongoDB Optimization

- Use connection pooling with replica set connections
- Enable connection compression
- Set appropriate read preference (primary/secondary)

## Migration Guide

### Migrating from Direct Connections to SSH Tunnels

1. Test SSH tunnel connection separately first
2. Update connection configuration to use tunnel
3. Verify database host is reachable from bastion
4. Test the new connection before removing old one
5. Update any automation/scripts using the connection

### Moving to VPC Private Links

1. Set up VPC endpoint service
2. Create endpoint in your VPC
3. Update security groups to allow traffic
4. Configure VPC connection in HowlerOps
5. Test connectivity before switching

## Examples

### Production PostgreSQL with SSH Tunnel

```yaml
Name: Production PostgreSQL
Type: PostgreSQL
Host: 10.0.1.50
Port: 5432
Database: production
Username: app_readonly
SSL Mode: require

SSH Tunnel:
  Enabled: true
  Host: bastion.prod.example.com
  Port: 22
  User: ubuntu
  Auth: Private Key
  Key Path: ~/.ssh/prod-bastion.pem
  Strict Checking: true
```

### MongoDB Replica Set

```yaml
Name: MongoDB Cluster
Type: MongoDB
Connection String: mongodb://user:pass@mongo1:27017,mongo2:27017,mongo3:27017/myapp?replicaSet=rs0&authSource=admin
Auth Database: admin
```

### Elasticsearch with API Key

```yaml
Name: Elasticsearch Logs
Type: Elasticsearch
Scheme: HTTPS
Host: es.example.com
Port: 9200
Database: logs-*
API Key: VnVhQ2ZHY0JDZGJrU...
```

### ClickHouse Analytics

```yaml
Name: ClickHouse Analytics
Type: ClickHouse
Host: clickhouse.internal
Port: 9000
Database: analytics
Username: analyst
Native Protocol: true
```

### TiDB Production Cluster

```yaml
Name: TiDB Production
Type: TiDB
Host: tidb.prod.internal
Port: 4000
Database: app_production
Username: app_user
SSL Mode: require
```

## Additional Resources

- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [MongoDB Connection Guide](https://docs.mongodb.com/manual/reference/connection-string/)
- [Elasticsearch SQL](https://www.elastic.co/guide/en/elasticsearch/reference/current/sql-getting-started.html)
- [ClickHouse Documentation](https://clickhouse.com/docs/)
- [TiDB Documentation](https://docs.pingcap.com/tidb/stable)
- [SSH Tunneling Guide](https://www.ssh.com/academy/ssh/tunneling)
- [AWS PrivateLink](https://aws.amazon.com/privatelink/)

## Support

For issues or questions:
- Check the troubleshooting section above
- Review connection logs in the application
- Verify network connectivity and permissions
- Consult database-specific documentation
