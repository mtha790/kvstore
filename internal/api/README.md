# API Package

This package provides HTTP API handlers for the key-value store service.

## Structure

- `handlers.go` - Main API handlers for key-value operations
- `handlers_test.go` - Comprehensive tests for all handlers
- `middleware.go` - HTTP middleware (logging, CORS, recovery)
- `middleware_test.go` - Tests for all middleware
- `router.go` - HTTP routing and request dispatching
- `router_test.go` - Tests for routing functionality
- `docs_handler.go` - Swagger UI and OpenAPI documentation handlers

## API Endpoints

### Key-Value Operations

- `GET /api/kv/{key}` - Retrieve a value by key
  - Returns: `200 OK` with JSON response containing key and value
  - Returns: `404 Not Found` if key doesn't exist
  - Returns: `400 Bad Request` for invalid keys

- `POST /api/kv/{key}` - Create a new key-value pair
  - Body: `{"value": "string"}`
  - Returns: `201 Created` if key is new
  - Returns: `200 OK` if key already exists (updates)
  - Returns: `400 Bad Request` for invalid input

- `PUT /api/kv/{key}` - Create or update a key-value pair
  - Body: `{"value": "string"}`
  - Returns: `201 Created` if key is new
  - Returns: `200 OK` if key already exists (updates)
  - Returns: `400 Bad Request` for invalid input

- `DELETE /api/kv/{key}` - Delete a key-value pair
  - Returns: `200 OK` with deleted key-value data
  - Returns: `404 Not Found` if key doesn't exist
  - Returns: `400 Bad Request` for invalid keys

- `GET /api/kv` - List all keys in the store
  - Returns: `200 OK` with JSON array of keys
  - Returns empty array if store is empty

### Health Check

- `GET /health` - Service health check
  - Returns: `200 OK` with service status

### API Documentation

- `GET /api/docs` - Interactive Swagger UI documentation
  - Returns: HTML page with Swagger UI interface
  - Allows testing all API endpoints interactively

- `GET /docs/openapi.yaml` - OpenAPI 3.0 specification
  - Returns: YAML file with complete API specification
  - Can be imported into API tools like Postman or Insomnia

## Response Format

All responses use JSON format with appropriate content-type headers:

### Success Responses

```json
// GET /api/kv/{key}
{
  "key": "example-key",
  "value": {
    "data": "example-value",
    "created_at": "2025-08-02T15:00:00Z",
    "updated_at": "2025-08-02T15:00:00Z", 
    "version": 1
  }
}

// POST/PUT /api/kv/{key}
{
  "key": "example-key",
  "value": {
    "data": "example-value",
    "created_at": "2025-08-02T15:00:00Z",
    "updated_at": "2025-08-02T15:00:00Z",
    "version": 1
  },
  "created": true
}

// DELETE /api/kv/{key}
{
  "key": "example-key",
  "value": {
    "data": "example-value",
    "created_at": "2025-08-02T15:00:00Z",
    "updated_at": "2025-08-02T15:00:00Z",
    "version": 1
  },
  "deleted": true
}

// GET /api/kv
{
  "keys": ["key1", "key2", "key3"]
}
```

### Error Responses

```json
{
  "message": "error description",
  "code": "ERROR_CODE" // optional
}
```

## HTTP Status Codes

- `200 OK` - Successful operation
- `201 Created` - Resource created successfully
- `400 Bad Request` - Invalid request (invalid key, empty value, malformed JSON)
- `404 Not Found` - Key not found or endpoint not found
- `405 Method Not Allowed` - HTTP method not supported for endpoint
- `500 Internal Server Error` - Server error

## Features

### Middleware Chain

1. **Recovery Middleware** - Catches panics and returns 500 error
2. **Logging Middleware** - Logs all HTTP requests and responses
3. **CORS Middleware** - Adds CORS headers for browser compatibility

### Dependency Injection

All handlers receive dependencies through constructor injection:
- `store.Store` interface for key-value operations
- `*logger.Logger` for structured logging

### Comprehensive Testing

- Unit tests for all handlers with various scenarios
- Middleware tests for all functionality
- Router tests for request routing
- Mock store implementation for isolated testing
- 100% test coverage of core functionality

### Error Handling

- Proper HTTP status codes for all error conditions
- Structured error responses with clear messages
- Context-aware operations with timeouts
- Graceful handling of store errors

### Logging

- Request/response logging with structured data
- Error logging with context information
- Performance metrics (response time, status codes)
- Debug logging for troubleshooting

## Usage

```go
// Create API router
store := memory.NewMemoryStore()
logger := logger.Default()
handler := api.SetupRoutes(store, logger)

// Start HTTP server
http.ListenAndServe(":8080", handler)
```