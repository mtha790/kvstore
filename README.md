# kvstore

A key value store in go for learning purpose.

## Installation & Getting Started

### Prerequisites

* Go 1.24.1 or higher

### Quick Start

```bash
# Clone the repository
git clone https://github.com/mtha790/kvstore
cd kvstore

# Build and run
make build
./bin/kvstore

# Run in development mode with hot reload
make dev
```

### Features

* Web Interface: http://localhost:8080
* API Documentation: http://localhost:8080/api/docs
* API Endpoint: http://localhost:8080/api/kv
* Health Check: http://localhost:8080/health
* OpenAPI Spec: http://localhost:8080/docs/openapi.yaml

### Available Commands

```bash
make test    # Run tests
make lint    # Run linter
make format  # Format code
make ci      # Run full CI pipeline
```
