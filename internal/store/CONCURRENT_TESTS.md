# Concurrent Testing Documentation

This document describes the comprehensive concurrent testing suite for the MemoryStore implementation.

## Overview

The concurrent testing suite ensures that the MemoryStore implementation is thread-safe, race-free, and performs well under high contention. All tests are designed to be run with Go's race detector (`-race` flag) to catch potential race conditions.

## Test Categories

### 1. Race Condition Tests

#### TestMemoryStore_ConcurrentReaderWriter
- **Purpose**: Tests multiple readers and writers under race conditions
- **Parameters**: 50 readers, 20 writers, 100 keys, 5-second duration
- **Validates**: No race conditions between concurrent read/write operations
- **Key Metrics**: Read/write operation counts, error rates

#### TestMemoryStore_CompareAndSwapHighContention
- **Purpose**: Tests CompareAndSwap operations under high contention
- **Parameters**: 100 goroutines, 50 attempts each, single contended key
- **Validates**: CAS semantics work correctly under contention
- **Expected**: Many CAS failures due to concurrent modifications

### 2. Memory Consistency Tests

#### TestMemoryStore_MemoryConsistency
- **Purpose**: Verifies memory consistency under concurrent access
- **Parameters**: 50 keys, 20 goroutines, 3-second duration
- **Validates**: 
  - Version numbers always increase
  - Timestamps are consistent (UpdatedAt >= CreatedAt)
  - No inconsistent intermediate states visible

### 3. Deadlock Detection Tests

#### TestMemoryStore_DeadlockDetection
- **Purpose**: Detects potential deadlocks in mixed operations
- **Parameters**: 50 goroutines, 10 keys, mixed operations
- **Operations Tested**: Get, Set, List, Size, Clear, CompareAndSwap
- **Detection Method**: Progress monitoring - flags if operations appear stuck

### 4. Stress Tests

#### TestMemoryStore_StressTest
Configurable stress tests with multiple scenarios:

1. **100 goroutines, 1000 ops each**: 70% reads, 30% writes
2. **500 goroutines, 500 ops each**: 50% reads, 50% writes  
3. **1000 goroutines, 100 ops each**: 90% reads, 10% writes

**Metrics Tracked**:
- Operations per second
- Error rates
- Total operations completed
- Test duration

### 5. Edge Case Tests

#### TestMemoryStore_EdgeCases

##### HighVersionNumbers
- Tests version number handling up to 10,000 updates
- Validates version increment consistency
- Ensures no integer overflow issues

##### LargeNumberOfKeys
- Tests with 100,000 keys
- Validates performance with large datasets
- Random access pattern verification

### 6. Context Cancellation Tests

#### TestMemoryStore_ContextCancellation

##### CancelledContextDuringOperations
- Tests graceful handling of context cancellation
- 50 goroutines performing operations when context is cancelled
- Verifies store remains functional after cancellation

##### TimeoutDuringHighLoad
- Tests timeout behavior with expired contexts
- Validates proper error return (context.DeadlineExceeded)

## Benchmark Tests

### Performance Benchmarks

#### BenchmarkMemoryStore_Read
- **Purpose**: Measures read operation performance
- **Setup**: 1000 pre-populated keys
- **Pattern**: Random key access

#### BenchmarkMemoryStore_Write
- **Purpose**: Measures write operation performance
- **Pattern**: Unique keys per operation

#### BenchmarkMemoryStore_MixedWorkload
- **Purpose**: Measures mixed read/write performance
- **Ratio**: 70% reads, 30% writes
- **Setup**: 1000 pre-populated keys

#### BenchmarkMemoryStore_CompareAndSwap
- **Purpose**: Measures CAS operation performance
- **Setup**: 100 keys with high contention
- **Expected**: Many ConcurrentModification errors

### Contention Benchmarks

#### BenchmarkMemoryStore_HighContentionRead/Write
- **Purpose**: Measures performance under high contention
- **Setup**: Only 10 keys to maximize contention
- **Validates**: Performance degradation under contention

### Scalability Benchmarks

#### BenchmarkMemoryStore_ScalabilityTest
- **Purpose**: Tests performance scaling with goroutine count
- **Goroutine counts**: 1, 2, 4, 8, 16, 32, 64, 128
- **Workload**: 80% reads, 20% writes
- **Metrics**: Operations per second vs goroutine count

## Running the Tests

### Individual Test Execution
```bash
# Run with race detector
go test -race -v ./internal/store -run TestMemoryStore_ConcurrentReaderWriter

# Run benchmarks
go test -bench=BenchmarkMemoryStore_Read ./internal/store
```

### Comprehensive Test Suite
```bash
# Run the provided script
./scripts/test-concurrent.sh
```

### Key Flags

- `-race`: Enables race detector (REQUIRED for concurrent tests)
- `-v`: Verbose output showing test progress
- `-timeout`: Prevents tests from hanging (important for concurrent tests)
- `-benchtime`: Controls benchmark duration

## Expected Results

### Passing Criteria

1. **Zero race conditions** detected by Go's race detector
2. **No deadlocks** in any test scenario
3. **Memory consistency** maintained under all conditions
4. **Error rates** remain at zero for normal operations
5. **Performance** scales reasonably with goroutine count

### Performance Baselines

Typical performance on modern hardware:
- **Read operations**: ~200-500 ns/op
- **Write operations**: ~300-600 ns/op
- **Mixed workload**: ~300-700 ns/op
- **Throughput**: 100k-500k ops/second depending on workload

### Failure Indicators

1. **Race detector warnings**: Indicates thread safety issues
2. **Deadlock timeouts**: Suggests locking problems
3. **Memory consistency violations**: Data corruption under concurrency
4. **Excessive error rates**: Implementation bugs
5. **Performance degradation**: Scalability issues

## Test Configuration

Tests are designed to be:
- **Deterministic**: Same behavior across runs
- **Scalable**: Adjustable parameters for different environments
- **Comprehensive**: Cover all concurrent access patterns
- **Fast**: Complete in reasonable time for CI/CD

## Maintenance

When modifying the store implementation:

1. **Always run** concurrent tests with `-race` flag
2. **Verify benchmarks** don't regress significantly
3. **Add new tests** for new concurrent access patterns
4. **Update timeouts** if test complexity increases
5. **Document** any new edge cases discovered

## Integration with CI/CD

Recommended CI configuration:
```yaml
- name: Concurrent Tests
  run: |
    go test -race -v ./internal/store -timeout 5m
    go test -bench=. ./internal/store -benchtime=1s
```

The test suite is designed to catch concurrency bugs early in the development cycle and ensure the MemoryStore implementation remains robust under production workloads.