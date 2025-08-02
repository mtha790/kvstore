# kvstore

Main entry point for the key-value store application.

## Usage

```bash
# Build
make build

# Run with default settings
./bin/kvstore

# Run with file persistence
KVSTORE_PERSISTENCE_TYPE=file KVSTORE_PERSISTENCE_PATH=./data.json ./bin/kvstore
```

## Configuration

See the main README.md for configuration options.