#!/bin/bash

echo "🧪 Testing Recovery Middleware for Literary Lions"
echo "================================================"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to test HTTP response
test_response() {
    local url=$1
    local expected_status=$2
    local test_name=$3
    
    echo -e "\n${YELLOW}Testing: $test_name${NC}"
    echo "URL: $url"
    
    # Make request and capture status code
    response=$(curl -s -w "%{http_code}" -o /dev/null "$url" 2>/dev/null)
    status_code=$response
    
    if [ "$status_code" -eq "$expected_status" ]; then
        echo -e "${GREEN}✅ PASS${NC} - Status code: $status_code"
    else
        echo -e "${RED}❌ FAIL${NC} - Expected: $expected_status, Got: $status_code"
    fi
}

# Check if server is running
echo "Checking if server is running on localhost:8080..."
if ! curl -s http://localhost:8080 > /dev/null; then
    echo -e "${RED}❌ Server is not running on localhost:8080${NC}"
    echo "Please start the server first with: go run main.go"
    exit 1
fi

echo -e "${GREEN}✅ Server is running${NC}"

# Test normal operation
test_response "http://localhost:8080/" 200 "Normal home page (should work)"

# Test 404 error handling
test_response "http://localhost:8080/nonexistent-page" 404 "404 Not Found (should be handled gracefully)"

# Test panic recovery (only if not in production)
test_response "http://localhost:8080/test-panic" 500 "Panic recovery (should return 500)"

# Test 500 error handling
test_response "http://localhost:8080/test-500" 500 "500 Internal Server Error (should be handled gracefully)"

echo -e "\n${YELLOW}================================================${NC}"
echo -e "${GREEN}🎉 Recovery middleware testing completed!${NC}"
echo -e "\nThe middleware should:"
echo -e "  ✅ Handle 5XX HTTP response codes gracefully"
echo -e "  ✅ Handle HTTP status 400 and 500 errors gracefully"
echo -e "  ✅ Recover from panics without crashing the server"
echo -e "  ✅ Log detailed error information"
echo -e "  ✅ Render nice error pages for users" 