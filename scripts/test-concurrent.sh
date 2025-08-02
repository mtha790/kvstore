#!/bin/bash

# Comprehensive concurrent testing script for the memory store
# This script runs all concurrent tests with the race detector enabled

set -e

echo "Running comprehensive concurrent tests with race detector..."
echo "=========================================================="

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to run a test and report results
run_test() {
    local test_name=$1
    local test_pattern=$2
    local timeout=${3:-30s}
    
    echo -e "${YELLOW}Running $test_name...${NC}"
    if go test -race -v ./internal/store -run "$test_pattern" -timeout "$timeout"; then
        echo -e "${GREEN}✓ $test_name passed${NC}"
        echo ""
    else
        echo -e "${RED}✗ $test_name failed${NC}"
        exit 1
    fi
}

# Function to run benchmarks
run_benchmark() {
    local bench_name=$1
    local bench_pattern=$2
    local benchtime=${3:-1s}
    
    echo -e "${YELLOW}Running $bench_name...${NC}"
    if go test -bench="$bench_pattern" -benchtime="$benchtime" ./internal/store; then
        echo -e "${GREEN}✓ $bench_name completed${NC}"
        echo ""
    else
        echo -e "${RED}✗ $bench_name failed${NC}"
        exit 1
    fi
}

echo "1. Basic concurrent access test"
run_test "Basic Concurrent Access" "TestMemoryStore_ConcurrentAccess"

echo "2. Multiple readers/writers race conditions test"
run_test "Concurrent Reader/Writer" "TestMemoryStore_ConcurrentReaderWriter" "60s"

echo "3. CompareAndSwap high contention test"
run_test "CompareAndSwap High Contention" "TestMemoryStore_CompareAndSwapHighContention"

echo "4. Memory consistency verification test"
run_test "Memory Consistency" "TestMemoryStore_MemoryConsistency" "60s"

echo "5. Deadlock detection test"
run_test "Deadlock Detection" "TestMemoryStore_DeadlockDetection" "60s"

echo "6. Stress tests with configurable parameters"
run_test "Stress Tests" "TestMemoryStore_StressTest" "120s"

echo "7. Edge cases (version overflow, large datasets)"
run_test "Edge Cases" "TestMemoryStore_EdgeCases" "120s"

echo "8. Context cancellation under load"
run_test "Context Cancellation" "TestMemoryStore_ContextCancellation"

echo -e "${YELLOW}Running performance benchmarks...${NC}"
echo "=========================================="

echo "9. Read performance benchmark"
run_benchmark "Read Performance" "BenchmarkMemoryStore_Read"

echo "10. Write performance benchmark" 
run_benchmark "Write Performance" "BenchmarkMemoryStore_Write"

echo "11. Mixed workload benchmark"
run_benchmark "Mixed Workload" "BenchmarkMemoryStore_MixedWorkload"

echo "12. CompareAndSwap performance benchmark"
run_benchmark "CompareAndSwap Performance" "BenchmarkMemoryStore_CompareAndSwap"

echo "13. High contention benchmarks"
run_benchmark "High Contention Read" "BenchmarkMemoryStore_HighContentionRead"
run_benchmark "High Contention Write" "BenchmarkMemoryStore_HighContentionWrite"

echo "14. Scalability test (different goroutine counts)"
run_benchmark "Scalability Test" "BenchmarkMemoryStore_ScalabilityTest" "2s"

echo -e "${GREEN}=========================================================="
echo -e "All concurrent tests and benchmarks passed successfully!"
echo -e "The MemoryStore implementation is race-free and performant."
echo -e "==========================================================${NC}"

echo ""
echo "Test Summary:"
echo "- Race detector: ENABLED ✓"
echo "- Concurrent operations: TESTED ✓"
echo "- Memory consistency: VERIFIED ✓"
echo "- Deadlock detection: PASSED ✓"
echo "- Performance benchmarks: COMPLETED ✓"
echo "- Stress tests: PASSED ✓"
echo ""
echo "The implementation successfully handles:"
echo "- Up to 1000 concurrent goroutines"
echo "- High-contention scenarios"
echo "- Version overflow edge cases"
echo "- Large datasets (100k+ keys)"
echo "- Context cancellation under load"