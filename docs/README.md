# SQLitREST Documentation

## Overview

SQLitREST is a lightweight, high-performance REST API server for SQLite databases, designed as a 100% PostgREST-compatible alternative. Built in pure Go with modern architecture, it provides instant REST APIs for any SQLite database with zero configuration.

## Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/hazyhaar/sqlitReST.git
cd sqlitReST/sqlitrest

# Build the binary
go build -o sqlitrest ./cmd/sqlitrest

# Or run directly
go run ./cmd/sqlitrest serve
```

### Basic Usage

```bash
# Start the server
./sqlitrest serve

# Generate JWT tokens
./sqlitrest generate-token

# Make API requests
curl -H "Authorization: Bearer <token>" http://localhost:34334/users
```

## Features

### ✅ 100% PostgREST Compatible
- Same filter syntax: `id=eq.1`, `name=like.john*`
- Same media types: JSON, CSV, single object, EXPLAIN plan
- Same authentication flow with JWT
- Same error codes and response format

### 🚀 Advanced Features
- **Resource Embedding**: `select=*,posts(title,content)`
- **Logical Operators**: `and=(id.eq.1,name.eq.test)`
- **Row Level Security**: Dynamic policies per table
- **Schema Caching**: 5-minute TTL for performance
- **OpenAPI 3.0**: Automatic specification generation
- **RPC Functions**: Custom SQL functions via `/rpc/*`

### 🏗️ Architecture
- **Pure Go**: No CGO dependencies
- **Modern SQLite**: Uses modernc.org/sqlite driver
- **Multi-Database**: Support for multiple SQLite files
- **Configuration**: TOML-based configuration
- **Production Ready**: Comprehensive error handling

## API Reference

### CRUD Operations

```bash
# GET all records
GET /users

# GET with filters
GET /users?id=eq.1&age=gt.18

# POST new record
POST /users
{"name": "John", "email": "john@example.com"}

# PATCH existing record
PATCH /users?id=eq.1
{"name": "John Updated"}

# DELETE record
DELETE /users?id=eq.1
```

### Advanced Filtering

```bash
# Logical operators
GET /users?and=(id.eq.1,age.gt.18)
GET /users?or=(name.eq.john*,role.eq.admin)

# Complex filters
GET /users?name=ilike.john*&age=gte.18&role=eq.user

# Range filters
GET /users?age=gte.18&age=lte.65
```

### Resource Embedding

```bash
# Include related tables
GET /users?select=*,posts(title,created_at)

# Nested relations
GET /posts?select=*,users(name,email)

# Specific columns
GET /users?select=id,name,email
```

### Media Types

```bash
# JSON (default)
curl -H "Accept: application/json" /users

# CSV export
curl -H "Accept: text/csv" /users

# Single object
curl -H "Accept: application/vnd.pgrst.object" /users?id=eq.1

# EXPLAIN plan
curl -H "Accept: application/vnd.pgrst.plan" /users
```

### Authentication

```bash
# Generate tokens
./sqlitrest generate-token

# Use tokens
curl -H "Authorization: Bearer <token>" /users

# Admin token (full access)
curl -H "Authorization: Bearer <admin-token>" /users

# User token (restricted access)
curl -H "Authorization: Bearer <user-token>" /users?id=eq.2
```

## Configuration

### sqlitrest.toml

```toml
[server]
host = "localhost"
port = 34334

[[databases]]
name = "main"
path = "./data/main.db"
mode = "readwrite"

[auth.jwt]
enabled = true
algorithm = "HS256"
secret = "your-secret-key"
issuer = "sqlitrest"
audience = ["sqlitrest-api"]
```

### Environment Variables

```bash
# Server configuration
export SQLITREST_HOST=localhost
export SQLITREST_PORT=34334

# JWT configuration
export SQLITREST_JWT_SECRET=your-secret-key
export SQLITREST_JWT_ENABLED=true
```

## Row Level Security

### Creating Policies

```sql
-- Users can see their own records
INSERT INTO _policies (name, table_name, action, expression, description)
VALUES ('users_select_own', 'users', 'SELECT', 
        'id = current_user_id() OR current_role() = ''admin''', 
        'Users can see their own profile, admins can see all');

-- Only admins can delete
INSERT INTO _policies (name, table_name, action, expression, description)
VALUES ('users_delete_admin_only', 'users', 'DELETE', 
        'current_role() = ''admin''', 
        'Only admins can delete users');
```

### Policy Functions

- `current_user_id()` - Current authenticated user ID
- `current_role()` - Current user role
- `current_tenant_id()` - Current tenant ID (if applicable)

## RPC Functions

### Creating Functions

```sql
-- Simple function
CREATE FUNCTION hello(name TEXT) RETURNS TEXT AS '
    SELECT ''Hello, '' || name || ''!''
';

-- Complex function
CREATE FUNCTION user_stats() RETURNS TABLE(
    total_users INTEGER,
    active_users INTEGER,
    admin_count INTEGER
) AS '
    SELECT 
        COUNT(*) as total_users,
        COUNT(CASE WHEN role = ''user'' THEN 1 END) as active_users,
        COUNT(CASE WHEN role = ''admin'' THEN 1 END) as admin_count
    FROM users
';
```

### Using RPC

```bash
# Call function
GET /rpc/hello?name=World

# Call with POST
POST /rpc/user_stats

# List functions
GET /rpc/
```

## Debug Endpoints

```bash
# Database information
GET /_debug/databases

# Authentication context
GET /_debug/auth

# Active policies
GET /_debug/policies

# Schema information
GET /_debug/schema?table=users

# Cache statistics
GET /_debug/cache
```

## Performance

### Schema Caching

- **TTL**: 5 minutes by default
- **Automatic refresh**: On cache miss
- **Manual invalidation**: Via debug endpoints
- **Statistics**: Cache hit/miss ratios

### Connection Pooling

- **Reader connections**: Multiple for read operations
- **Writer connections**: Single for write operations
- **Automatic management**: Built-in connection lifecycle

## Deployment

### Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o sqlitrest ./cmd/sqlitrest

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/sqlitrest .
COPY --from=builder /app/sqlitrest.toml .
EXPOSE 34334
CMD ["./sqlitrest", "serve"]
```

### Systemd

```ini
[Unit]
Description=SQLitREST API Server
After=network.target

[Service]
Type=simple
User=sqlitrest
WorkingDirectory=/opt/sqlitrest
ExecStart=/opt/sqlitrest/sqlitrest serve
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### Production Tips

1. **Use WAL mode**: Better concurrency
2. **Enable foreign keys**: Data integrity
3. **Configure JWT**: Strong secret keys
4. **Set up policies**: Row-level security
5. **Monitor cache**: Performance optimization
6. **Use HTTPS**: Production security

## Examples

### Complete API Session

```bash
# 1. Start server
./sqlitrest serve &

# 2. Generate admin token
ADMIN_TOKEN=$(./sqlitrest generate-token | grep "Admin Token" | cut -d' ' -f3)

# 3. Create user table
curl -X POST -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '[
    {"id": 1, "name": "Admin", "email": "admin@example.com", "role": "admin"},
    {"id": 2, "name": "User", "email": "user@example.com", "role": "user"}
  ]' \
  http://localhost:34334/users

# 4. Generate user token
USER_TOKEN=$(./sqlitrest generate-token | grep "User Token" | cut -d' ' -f3)

# 5. Test Row Level Security
curl -H "Authorization: Bearer $USER_TOKEN" \
  http://localhost:34334/users
# Returns only user's own record

# 6. Test resource embedding
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  "http://localhost:34334/users?select=*,posts(title)"

# 7. Export CSV
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Accept: text/csv" \
  http://localhost:34334/users > users.csv
```

## Troubleshooting

### Common Issues

1. **"Connection refused"** - Server not running
2. **"Authentication failed"** - Invalid JWT token
3. **"Permission denied"** - Row Level Security policy
4. **"Database error"** - SQL syntax or constraint issue

### Debug Mode

```bash
# Enable verbose logging
./sqlitrest serve --verbose

# Check configuration
./sqlitrest config

# Test database connection
./sqlitrest test-db
```

### Health Check

```bash
# Server health
curl http://localhost:34334/health

# Database status
curl http://localhost:34334/_debug/databases

# Cache status
curl http://localhost:34334/_debug/cache
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - see LICENSE file for details.

## Support

- **Issues**: https://github.com/hazyhaar/sqlitReST/issues
- **Documentation**: https://github.com/hazyhaar/sqlitReST/docs
- **Examples**: https://github.com/hazyhaar/sqlitReST/examples

---

**SQLitREST** - Instant REST APIs for SQLite databases 🚀