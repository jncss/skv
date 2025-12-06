# SKV CLI - Quick Start

## Install

```bash
go install github.com/jncss/skv/tools/cli@latest
```

Or build from source:

```bash
git clone https://github.com/jncss/skv.git
cd skv/tools/cli
make build
sudo cp skv /usr/local/bin/  # optional: install globally
```

## Verify Installation

```bash
skv help
```

## First Steps

```bash
# Create a database and add some data
skv put mydb.skv name "Alice"
skv put mydb.skv email "alice@example.com"
skv put mydb.skv role "admin"

# Read data
skv get mydb.skv name

# List all keys
skv keys mydb.skv

# Show all data
skv foreach mydb.skv

# Check database stats
skv verify mydb.skv
```

## Examples

See [EXAMPLES.md](EXAMPLES.md) for comprehensive usage examples.

## Full Documentation

See [README.md](README.md) for complete command reference.
