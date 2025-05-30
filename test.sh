#!/bin/bash

# Test runner script for kubedepot

set -e

# Default options
RUN_INTEGRATION=false
INTERACTIVE=true

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -i|--integration)
            RUN_INTEGRATION=true
            shift
            ;;
        --no-integration)
            RUN_INTEGRATION=false
            shift
            ;;
        --non-interactive|--ci)
            INTERACTIVE=false
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo "Options:"
            echo "  -i, --integration     Run integration tests"
            echo "  --no-integration      Skip integration tests (default)"
            echo "  --non-interactive     Run without prompts (good for CI)"
            echo "  --ci                  Alias for --non-interactive"
            echo "  -h, --help           Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

echo "🧪 Running tests for kubedepot..."
echo

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Function to run tests in a specific package
run_package_tests() {
    local package=$1
    local description=$2

    print_status $BLUE "📦 Testing $description..."
    if go test -v "$package"; then
        print_status $GREEN "✅ $description tests passed"
    else
        print_status $RED "❌ $description tests failed"
        exit 1
    fi
    echo
}

# Function to run tests with coverage
run_with_coverage() {
    local package=$1
    local description=$2
    local coverfile=$3

    print_status $BLUE "📊 Testing $description with coverage..."
    if go test -v -coverprofile="$coverfile" "$package"; then
        go tool cover -html="$coverfile" -o "${coverfile%.out}.html"
        coverage=$(go tool cover -func="$coverfile" | grep total: | awk '{print $3}')
        print_status $GREEN "✅ $description tests passed - coverage: $coverage - report: ${coverfile%.out}.html"
    else
        print_status $RED "❌ $description tests failed"
        exit 1
    fi
    echo
}

# Clean up any existing coverage files
rm -f *.out *.html

print_status $YELLOW "🚀 Starting test suite..."
echo

# Run unit tests for main package
run_with_coverage "./cmd/kubedepot" "Main Application" "main.cover.out"

# Run unit tests for config package
run_with_coverage "./internal/config" "Configuration Package" "config.cover.out"

# Run unit tests for server package
run_with_coverage "./internal/server" "Server Package" "server.cover.out"

# Run short tests (excluding integration tests)
print_status $BLUE "⚡ Running unit tests only (short mode)..."
if go test -short -v ./...; then
    print_status $GREEN "✅ Unit tests passed"
else
    print_status $RED "❌ Unit tests failed"
    exit 1
fi
echo

# Handle integration tests based on mode
if [ "$INTERACTIVE" = true ] && [ "$RUN_INTEGRATION" = false ]; then
    # Ask user if they want to run integration tests
    read -p "🔗 Run integration tests? (they start a real server) [y/N]: " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        RUN_INTEGRATION=true
    fi
fi

if [ "$RUN_INTEGRATION" = true ]; then
    print_status $BLUE "🔗 Running integration tests..."
    if go test -v ./cmd/kubedepot -run TestIntegration; then
        print_status $GREEN "✅ Integration tests passed"
    else
        print_status $RED "❌ Integration tests failed"
        exit 1
    fi
    echo
else
    print_status $YELLOW "⏭️  Skipping integration tests"
    echo
fi

# Run all tests with race detection
print_status $BLUE "🏁 Running race detection tests..."
if go test -race -short ./...; then
    print_status $GREEN "✅ Race detection tests passed"
else
    print_status $RED "❌ Race condition detected"
    exit 1
fi
echo

# Generate overall coverage report
print_status $BLUE "📈 Generating overall coverage report..."
echo "mode: set" > coverage.out
grep -h -v "^mode:" *.cover.out >> coverage.out || true
if [ -f coverage.out ]; then
    go tool cover -html=coverage.out -o coverage.html
    total_coverage=$(go tool cover -func=coverage.out | grep total: | awk '{print $3}')
    print_status $GREEN "📊 Overall coverage: $total_coverage - report: coverage.html"
fi
echo

# Display individual package coverage summary
print_status $BLUE "📋 Coverage Summary by Package:"
if [ -f main.cover.out ]; then
    main_coverage=$(go tool cover -func=main.cover.out | grep total: | awk '{print $3}')
    print_status $YELLOW "   📦 Main Package: $main_coverage (Note: Low coverage expected - main.go has minimal testable code)"
fi
if [ -f config.cover.out ]; then
    config_coverage=$(go tool cover -func=config.cover.out | grep total: | awk '{print $3}')
    print_status $YELLOW "   ⚙️  Config Package: $config_coverage"
fi
if [ -f server.cover.out ]; then
    server_coverage=$(go tool cover -func=server.cover.out | grep total: | awk '{print $3}')
    print_status $YELLOW "   🖥️  Server Package: $server_coverage"
fi
echo

print_status $GREEN "🎉 All tests completed successfully!"
print_status $YELLOW "📁 Generated files:"
print_status $YELLOW "   - coverage.html (overall coverage)"
print_status $YELLOW "   - main.cover.html (main package coverage)"
print_status $YELLOW "   - config.cover.html (config package coverage)"
print_status $YELLOW "   - server.cover.html (server package coverage)"