#!/bin/bash

# Integration test script for kvstore with timeout and detailed logging

set -e

# Set timeout for entire script (10 seconds)
( sleep 10 && echo "[$(date +%H:%M:%S)] TIMEOUT: Test exceeded 10 seconds!" && kill -TERM $$ ) &
TIMEOUT_PID=$!

# Cleanup function
cleanup() {
    echo "[$(date +%H:%M:%S)] Cleanup: Starting cleanup process..."
    kill $TIMEOUT_PID 2>/dev/null || true
    if [ -n "$SERVER_PID" ]; then
        echo "[$(date +%H:%M:%S)] Cleanup: Killing server PID $SERVER_PID..."
        kill $SERVER_PID 2>/dev/null || true
    fi
    echo "[$(date +%H:%M:%S)] Cleanup: Killing any processes on port 8080..."
    lsof -ti:8080 | xargs kill -9 2>/dev/null || true
    echo "[$(date +%H:%M:%S)] Cleanup: Complete"
}

trap cleanup EXIT

echo "[$(date +%H:%M:%S)] === Integration Test Suite ==="

# Kill any existing server
echo "[$(date +%H:%M:%S)] Cleaning up any existing processes..."
lsof -ti:8080 | xargs kill -9 2>/dev/null || true
sleep 1

# Build the application
echo "[$(date +%H:%M:%S)] Building application..."
make build
echo "[$(date +%H:%M:%S)] Build complete"

# Start the server
echo "[$(date +%H:%M:%S)] Starting server..."
KVSTORE_PERSISTENCE_TYPE=${KVSTORE_PERSISTENCE_TYPE:-memory} \
KVSTORE_PERSISTENCE_PATH=${KVSTORE_PERSISTENCE_PATH:-./data.json} \
../bin/kvstore &
SERVER_PID=$!
echo "[$(date +%H:%M:%S)] Server PID: $SERVER_PID"

# Wait for server to start
echo "[$(date +%H:%M:%S)] Waiting for server to start..."
server_ready=false
for i in {1..10}; do
    echo "[$(date +%H:%M:%S)] Checking server status (attempt $i/10)..."
    if curl -s --max-time 2 http://localhost:8080/health >/dev/null 2>&1; then
        echo "[$(date +%H:%M:%S)] Server is ready!"
        server_ready=true
        break
    fi
    sleep 1
done

if [ "$server_ready" = false ]; then
    echo "[$(date +%H:%M:%S)] ERROR: Server failed to start after 10 seconds"
    exit 1
fi

# Function to test endpoint
test_endpoint() {
    local method=$1
    local url=$2
    local data=$3
    local expected_status=$4
    
    echo "[$(date +%H:%M:%S)] Testing $method $url..."
    
    if [ -n "$data" ]; then
        response=$(curl -s --max-time 5 -w "\n%{http_code}" -X $method -H "Content-Type: application/json" -d "$data" $url 2>&1)
    else
        response=$(curl -s --max-time 5 -w "\n%{http_code}" -X $method $url 2>&1)
    fi
    
    status=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$status" = "$expected_status" ]; then
        echo "[$(date +%H:%M:%S)]   ✓ Success (Status: $status)"
        if [ -n "$body" ] && [ "$body" != "" ]; then
            echo "[$(date +%H:%M:%S)]   Response: $body"
        fi
    else
        echo "[$(date +%H:%M:%S)]   ✗ Failed (Expected: $expected_status, Got: $status)"
        echo "[$(date +%H:%M:%S)]   Response: $body"
        exit 1
    fi
}

echo ""
echo "[$(date +%H:%M:%S)] === API Tests ==="

# Test health endpoint
test_endpoint "GET" "http://localhost:8080/health" "" "200"

# Test list keys (empty)
test_endpoint "GET" "http://localhost:8080/api/kv" "" "200"

# Test create key
test_endpoint "POST" "http://localhost:8080/api/kv/test-key" '{"value":"test-value"}' "201"

# Test get key
test_endpoint "GET" "http://localhost:8080/api/kv/test-key" "" "200"

# Test update key
test_endpoint "PUT" "http://localhost:8080/api/kv/test-key" '{"value":"updated-value"}' "200"

# Test list keys (with data)
test_endpoint "GET" "http://localhost:8080/api/kv" "" "200"

# Test delete key
test_endpoint "DELETE" "http://localhost:8080/api/kv/test-key" "" "200"

# Test get deleted key
test_endpoint "GET" "http://localhost:8080/api/kv/test-key" "" "404"

# Test key with space (returns 404 as key doesn't exist)
test_endpoint "GET" "http://localhost:8080/api/kv/%20" "" "404"

echo ""
echo "[$(date +%H:%M:%S)] === Web Interface Test ==="

# Test web interface
echo "[$(date +%H:%M:%S)] Testing web interface..."
if curl -s --max-time 5 http://localhost:8080/ | grep -q "Key-Value Store"; then
    echo "[$(date +%H:%M:%S)]   ✓ Web interface accessible"
else
    echo "[$(date +%H:%M:%S)]   ✗ Web interface not accessible"
    exit 1
fi

echo ""
echo "[$(date +%H:%M:%S)] === Persistence Test ==="

# Add some data
echo "[$(date +%H:%M:%S)] Adding test data for persistence..."
curl -s --max-time 5 -X POST -H "Content-Type: application/json" -d '{"value":"persist-test"}' http://localhost:8080/api/kv/persist-key

# Kill server
echo "[$(date +%H:%M:%S)] Stopping server for persistence test..."
kill $SERVER_PID
echo "[$(date +%H:%M:%S)] Waiting for server to stop..."
wait $SERVER_PID 2>/dev/null || true

# Restart server
echo "[$(date +%H:%M:%S)] Restarting server..."
KVSTORE_PERSISTENCE_TYPE=${KVSTORE_PERSISTENCE_TYPE:-memory} \
KVSTORE_PERSISTENCE_PATH=${KVSTORE_PERSISTENCE_PATH:-./data.json} \
../bin/kvstore &
SERVER_PID=$!
echo "[$(date +%H:%M:%S)] New server PID: $SERVER_PID"

# Wait for server to start
echo "[$(date +%H:%M:%S)] Waiting for server to restart..."
sleep 2

# Check if server is ready
server_ready=false
for i in {1..5}; do
    echo "[$(date +%H:%M:%S)] Checking restarted server (attempt $i/5)..."
    if curl -s --max-time 2 http://localhost:8080/health >/dev/null 2>&1; then
        server_ready=true
        break
    fi
    sleep 1
done

if [ "$server_ready" = false ]; then
    echo "[$(date +%H:%M:%S)] ERROR: Server failed to restart"
    exit 1
fi

# Check if data persisted
echo "[$(date +%H:%M:%S)] Testing data persistence..."
response=$(curl -s --max-time 5 http://localhost:8080/api/kv/persist-key)
if echo "$response" | grep -q "persist-test"; then
    echo "[$(date +%H:%M:%S)]   ✓ Data persisted successfully"
    echo "[$(date +%H:%M:%S)]   Response: $response"
else
    echo "[$(date +%H:%M:%S)]   ✗ Data not persisted"
    echo "[$(date +%H:%M:%S)]   Response: $response"
fi

# Success - cancel timeout
kill $TIMEOUT_PID 2>/dev/null || true

echo ""
echo "[$(date +%H:%M:%S)] === All tests completed! ==="