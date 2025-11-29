# API Reference

This document provides a comprehensive reference for all SQLitREST API endpoints and features.

## Base URL

```
http://localhost:34334
```

## Authentication

SQLitREST uses JWT tokens for authentication. Include the token in the `Authorization` header:

```bash
curl -H "Authorization: Bearer <your-jwt-token>" http://localhost:34334/users
```

### Generate Tokens

```bash
./sqlitrest generate-token
```

This generates:
- **Admin Token**: Full access to all resources
- **User Token**: Restricted access based on Row Level Security policies

## Endpoints

### Database Operations

#### GET /{table}
Retrieve records from a table.

```bash
# Get all records
GET /users

# Get with filters
GET /users?id=eq.1&age=gt.18

# Get specific columns
GET /users?select=id,name,email

# Get with ordering
GET /users?order=id.desc,age.asc

# Get with pagination
GET /users?limit=10&offset=20
```

#### POST /{table}
Create new records in a table.

```bash
# Create single record
POST /users
Content-Type: application/json
{"name": "John", "email": "john@example.com", "age": 25}

# Create multiple records
POST /users
Content-Type: application/json
[
  {"name": "John", "email": "john@example.com"},
  {"name": "Jane", "email": "jane@example.com"}
]
```

#### PATCH /{table}
Update existing records.

```bash
# Update with filter
PATCH /users?id=eq.1
Content-Type: application/json
{"name": "John Updated", "age": 26}

# Update multiple fields
PATCH /users?id=eq.1
Content-Type: application/json
{"name": "John", "email": "newemail@example.com", "age": 30}
```

#### DELETE /{table}
Delete records from a table.

```bash
# Delete with filter
DELETE /users?id=eq.1

# Delete multiple records
DELETE /users?age=lt.18
```

## Filtering

### Basic Filters

PostgREST-style filtering with operators:

| Operator | Description | Example |
|----------|-------------|---------|
| `eq` | Equal | `id=eq.1` |
| `neq` | Not equal | `id=neq.1` |
| `gt` | Greater than | `age=gt.18` |
| `gte` | Greater than or equal | `age=gte.18` |
| `lt` | Less than | `age=lt.65` |
| `lte` | Less than or equal | `age=lte.65` |
| `like` | LIKE pattern | `name=like.john*` |
| `ilike` | ILIKE pattern (case-insensitive) | `name=ilike.john*` |
| `in` | IN list | `id=in.1,2,3` |
| `is` | IS NULL/NOT NULL | `deleted_at=is.null` |
| `not` | NOT operator | `name=not.john` |

### Logical Operators

Combine multiple conditions with AND/OR:

```bash
# AND conditions (all must be true)
GET /users?and=(id.eq.1,age.gt.18,name.like.john*)

# OR conditions (any must be true)
GET /users?or=(id.eq.1,id.eq.2,name.eq.admin)

# Mixed with basic filters
GET /users?age=gt.18&or=(name.eq.john*,role.eq.admin)
```

### Complex Examples

```bash
# Users older than 18, named John OR admins
GET /users?and=(age.gt.18)&or=(name.ilike.john*,role.eq.admin)

# Active users in specific departments
GET /users?and=(status.eq.active,department.in.sales,marketing,engineering)

# Recent posts by active users
GET /posts?and=(created_at=gte.2024-01-01,author_id.in.(SELECT id FROM users WHERE status = 'active'))
```

## Resource Embedding

Include related tables using foreign key relationships:

### Basic Embedding

```bash
# Include all columns from related table
GET /users?select=*,posts

# Include specific columns
GET /users?select=*,posts(title,created_at)

# Multiple relations
GET /users?select=*,posts(title),profile(bio)
```

### Nested Relations

```bash
# Deep nesting
GET /posts?select=*,users(name,email),comments(*,users(name))
```

### Column Selection

```bash
# Specific columns only
GET /users?select=id,name,email

# Exclude columns (PostgREST style)
GET /users?select=-password,-secret_key

# Mixed with relations
GET /users?select=id,name,posts(title,created_at)
```

## Media Types

Control response format with `Accept` header:

### JSON (Default)

```bash
curl -H "Accept: application/json" http://localhost:34334/users
```

### CSV Export

```bash
curl -H "Accept: text/csv" http://localhost:34334/users > users.csv
```

### Single Object

Returns first result as object (not array):

```bash
curl -H "Accept: application/vnd.pgrst.object" http://localhost:34334/users?id=eq.1
```

### EXPLAIN Plan

Returns query execution plan:

```bash
curl -H "Accept: application/vnd.pgrst.plan" http://localhost:34334/users
```

## RPC Functions

Execute custom SQL functions:

### List Functions

```bash
GET /rpc/
```

### Call Function

```bash
# GET request with parameters
GET /rpc/hello?name=World

# POST request with JSON body
POST /rpc/user_stats
Content-Type: application/json
{}

# Function with parameters
POST /rpc/create_user
Content-Type: application/json
{"name": "John", "email": "john@example.com"}
```

### Create Functions

```sql
-- Simple function
CREATE FUNCTION hello(name TEXT) RETURNS TEXT AS '
    SELECT ''Hello, '' || name || ''!''
';

-- Function returning table
CREATE FUNCTION user_stats() RETURNS TABLE(
    total_users INTEGER,
    active_users INTEGER
) AS '
    SELECT 
        COUNT(*) as total_users,
        COUNT(CASE WHEN status = ''active'' THEN 1 END) as active_users
    FROM users
';
```

## Row Level Security

### Policy Functions

Use these functions in policy expressions:

- `current_user_id()` - Current authenticated user ID
- `current_role()` - Current user role (admin/user/anonymous)
- `current_tenant_id()` - Current tenant ID (if applicable)

### Creating Policies

```sql
-- Users can see their own records
INSERT INTO _policies (name, table_name, action, expression, description)
VALUES ('users_select_own', 'users', 'SELECT', 
        'id = current_user_id() OR current_role() = ''admin''', 
        'Users can see their own profile, admins can see all');

-- Users can update their own records
INSERT INTO _policies (name, table_name, action, expression, description)
VALUES ('users_update_own', 'users', 'UPDATE', 
        'id = current_user_id() OR current_role() = ''admin''', 
        'Users can update their own profile, admins can update all');

-- Only admins can delete
INSERT INTO _policies (name, table_name, action, expression, description)
VALUES ('users_delete_admin_only', 'users', 'DELETE', 
        'current_role() = ''admin''', 
        'Only admins can delete users');

-- Public posts visible to all, own posts to authors
INSERT INTO _policies (name, table_name, action, expression, description)
VALUES ('posts_select_public_or_own', 'posts', 'SELECT', 
        'is_public = TRUE OR author_id = current_user_id() OR current_role() = ''admin''', 
        'Public posts visible to all, own posts to authors, all to admins');
```

## Debug Endpoints

### Database Information

```bash
GET /_debug/databases
```

Response:
```json
{
  "databases": [
    {
      "name": "main",
      "path": "./data/main.db",
      "mode": "readwrite",
      "size": "1.2MB"
    }
  ]
}
```

### Authentication Context

```bash
GET /_debug/auth
```

Response:
```json
{
  "authenticated": true,
  "user_id": "1",
  "role": "admin",
  "tenant_id": null,
  "permissions": ["read:all", "write:all"]
}
```

### Active Policies

```bash
GET /_debug/policies
```

Response:
```json
{
  "policies": [
    {
      "name": "users_select_own",
      "table": "users",
      "action": "SELECT",
      "expression": "id = current_user_id() OR current_role() = 'admin'",
      "description": "Users can see their own profile, admins can see all"
    }
  ]
}
```

### Schema Information

```bash
GET /_debug/schema?table=users
```

Response:
```json
{
  "table": "users",
  "columns": [
    {
      "name": "id",
      "type": "INTEGER",
      "not_null": true,
      "primary_key": true
    },
    {
      "name": "name",
      "type": "TEXT",
      "not_null": true,
      "primary_key": false
    }
  ],
  "foreign_keys": [],
  "indexes": [
    {
      "name": "idx_users_email",
      "columns": ["email"],
      "unique": true
    }
  ]
}
```

### Cache Statistics

```bash
GET /_debug/cache
```

Response:
```json
{
  "cached_tables": 5,
  "tables": ["users", "posts", "comments", "profiles", "categories"],
  "oldest_cache": "2024-01-15T10:30:00Z",
  "newest_cache": "2024-01-15T10:35:00Z",
  "oldest_age_seconds": 300,
  "newest_age_seconds": 120
}
```

## OpenAPI Specification

### Get OpenAPI JSON

```bash
GET /swagger.json
```

### Interactive Documentation

```bash
GET /
```

Returns OpenAPI 3.0 specification with all endpoints, schemas, and examples.

## Error Handling

SQLitREST returns PostgREST-compatible error responses:

### Error Response Format

```json
{
  "code": "PGRST301",
  "message": "Authentication failed",
  "details": "Invalid or missing JWT token",
  "hint": "Include a valid JWT token in the Authorization header"
}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| PGRST301 | 401 | Authentication failed |
| PGRST302 | 403 | Permission denied |
| PGRST303 | 404 | Resource not found |
| PGRST304 | 400 | Validation error |
| PGRST500 | 500 | Internal server error |
| PGRST501 | 500 | Database error |

### Common Errors

```bash
# Authentication failed
{
  "code": "PGRST301",
  "message": "Authentication failed",
  "details": "Invalid JWT token"
}

# Permission denied
{
  "code": "PGRST302", 
  "message": "Permission denied",
  "details": "Row Level Security policy violation"
}

# Validation error
{
  "code": "PGRST304",
  "message": "Invalid request", 
  "details": "Invalid filter operator: xyz"
}

# Database error
{
  "code": "PGRST501",
  "message": "Database error",
  "details": "UNIQUE constraint failed: users.email"
}
```

## Performance Tips

### Use Specific Columns

```bash
# Good - specific columns
GET /users?select=id,name,email

# Avoid - select all when not needed
GET /users?select=*
```

### Use Limits

```bash
# Good - paginate results
GET /users?limit=50&offset=0

# Avoid - too many results
GET /users
```

### Use Indexes

```sql
-- Create indexes for common filters
CREATE INDEX idx_users_age ON users(age);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_created_at ON users(created_at);
```

### Use Appropriate Media Types

```bash
# JSON for APIs
curl -H "Accept: application/json" /users

# CSV for data export
curl -H "Accept: text/csv" /users > export.csv

# Single object for known single results
curl -H "Accept: application/vnd.pgrst.object" /users?id=eq.1
```

## Examples

### Complete CRUD Session

```bash
# 1. Start server and get admin token
./sqlitrest serve &
ADMIN_TOKEN=$(./sqlitrest generate-token | grep "Admin Token" | cut -d' ' -f3)

# 2. Create table (via SQL or existing)
# 3. Create user
curl -X POST -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "John", "email": "john@example.com", "age": 25}' \
  http://localhost:34334/users

# 4. Get user with embedding
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  "http://localhost:34334/users?select=*,posts(title)&id=eq.1"

# 5. Update user
curl -X PATCH -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"age": 26}' \
  "http://localhost:34334/users?id=eq.1"

# 6. Export to CSV
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Accept: text/csv" \
  http://localhost:34334/users > users.csv
```

### Advanced Filtering

```bash
# Complex query: Active users older than 25, named John OR admins
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:34334/users?and=(age.gt.25,status.eq.active)&or=(name.ilike.john*,role.eq.admin)"

# Recent posts by active users with comments
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:34334/posts?and=(created_at=gte.2024-01-01)&select=*,users(name),comments(content)&order=created_at.desc"
```

---

This API reference covers all SQLitREST features. For more examples and deployment guides, see the [main documentation](README.md).