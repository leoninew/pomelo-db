---
name: pomelo-db
description: >
  Execute SQL queries using pomelo-db CLI tool (a local database client like mysql/psql).
  Use when: user wants to list datasources, query databases, check table structure, or inspect data.
  This is a LOCAL tool - just run pomelo-db commands directly, no remote login needed.

  IMPORTANT: pomelo-db loads datasources from .env in the CURRENT WORKING DIRECTORY.
  Always `cd` to the project directory before running pomelo-db commands.
  Use `-a` to add datasources (creates .env automatically if it doesn't exist).
argument-hint: <datasource> <sql>
allowed-tools: Bash(pomelo-db:*)
---

# Database Query

Execute SQL queries using pomelo-db CLI tool (readonly by default, use `-w`/`--allow-write` for write operations).

**This is a LOCAL database client tool** - it runs on your machine and connects to databases via local config file (`config.yaml`) or `.env` file.
No K8s/SSH/remote login required.

## Quick Setup

Add datasources to `.env` file using the `-a` flag:

```bash
# Add a SQLite datasource
pomelo-db -a mydb=sqlite://./data/app.db

# Add a MySQL datasource
pomelo-db -a prod=mysql://user:pass@host:3306/database

# List all datasources
pomelo-db -l
```

**DSN Format:**
- `mysql://user:pass@host:port/db`
- `sqlite://./path/to/db`
- `sqlserver://user:pass@host:port/db`
- `vastbase://user:pass@host:port/db?schema=public`
- `opengauss://user:pass@host:port/db`
- `dm://user:pass@host:port/db`

## Usage

```bash
# Execute a query (readonly, JSON output)
pomelo-db -d <datasource> -e "<sql>"

# Execute a query (table output)
pomelo-db -d <datasource> -e "<sql>" -o table

# Execute a write operation (INSERT/UPDATE/DELETE)
pomelo-db -d <datasource> -e "<sql>" -w
```

**Common mistakes to avoid:**
- ❌ Datasource name not found — run `-l` first to verify
- ❌ Using `-` instead of `_` in datasource names
- ❌ Write operations without `--allow-write` / `-w` flag

## Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--list` | `-l` | List all configured datasources |
| `--datasource` | `-d` | Datasource name (required for queries) |
| `--execute` | `-e` | SQL query string |
| `--file` | `-f` | SQL file path (alternative to -e) |
| `--output` | `-o` | Output format: `json` (default) or `table` |
| `--timeout` | `-t` | Query timeout in seconds (default: 30) |
| `--allow-write` | `-w` | Allow write operations (INSERT/UPDATE/DELETE) |
| `--add` | `-a` | Add a datasource to .env (format: name=dsn) |
| `--remove` | `-r` | Remove a datasource from .env |
| `--show-config` | `-s` | Show config for a specific datasource |
| `--verbose` | `-v` | Verbose output |

## Constraints

- Default readonly mode: only SELECT, SHOW, DESCRIBE, PRAGMA queries are allowed
- Use `-w`/`--allow-write` to allow write operations (INSERT/UPDATE/DELETE)
- Supported database types: mysql, sqlserver, dm, opengauss, vastbase, sqlite

## Examples

```bash
# Add a datasource
pomelo-db -a mydb=sqlite://./data.db
pomelo-db -a prod=mysql://user:pass@host:3306/database

# List datasources
pomelo-db -l

# Execute a query
pomelo-db -d mydb -e "SELECT * FROM users LIMIT 10"

# Table output format
pomelo-db -d mydb -e "SELECT * FROM users LIMIT 10" -o table

# Execute a write operation
pomelo-db -d mydb -e "INSERT INTO users (name) VALUES ('Alice')" -w

# Show datasource config (password masked)
pomelo-db -s mydb

# Remove a datasource
pomelo-db -r mydb
```
