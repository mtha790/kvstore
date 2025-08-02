# Scripts

Utility scripts for development and testing.

## Available Scripts

### test_integration.sh

Full integration test suite that tests:
- API endpoints
- Web interface
- Data persistence
- Server lifecycle

Usage:
```bash
# Run with default (memory) persistence
./scripts/test_integration.sh

# Run with file persistence
KVSTORE_PERSISTENCE_TYPE=file KVSTORE_PERSISTENCE_PATH=./data.json ./scripts/test_integration.sh
```

### test-concurrent.sh

Concurrent stress tests for the store implementation.

Usage:
```bash
./scripts/test-concurrent.sh
```