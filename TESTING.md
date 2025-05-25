# Testing Guide

This document describes the testing strategy and setup for the gogetkubeconfig project.

## Test Structure

The project uses Go's standard testing framework with the following test files:

1. **`cmd/gogetkubeconfig/main_test.go`** - Tests for main application logic
2. **`internal/server/server_test.go`** - HTTP server endpoint tests
3. **`internal/server/util_test.go`** - Utility function tests
4. **`cmd/gogetkubeconfig/integration_test.go`** - End-to-end integration tests

## Test Data Management

### Organized Testdata Directory Structure

The testdata is organized into separate directories for different test scenarios:

```
testdata/
├── valid-configs/        # Only valid kubeconfig files (for "get all" operations)
│   ├── dev.yaml
│   ├── prod.yaml
│   ├── integration-dev.yaml
│   ├── integration-prod.yaml
│   └── valid-test.yaml
├── invalid-configs/      # Only invalid kubeconfig files (for error testing)
│   └── invalid.yaml
├── mixed-configs/        # All files including invalid (for comprehensive testing)
│   ├── dev.yaml
│   ├── prod.yaml
│   ├── integration-dev.yaml
│   ├── integration-prod.yaml
│   ├── valid-test.yaml
│   └── invalid.yaml
├── kubeconfigs/          # Source files for unit test copying
│   └── [all files]
└── templates/            # HTML templates for web interface testing
    └── index.html
```

### Test Server Creation Functions

The tests use different server creation functions depending on the test scenario:

#### `createTestServerValid(t *testing.T)`
- **Purpose**: Tests that need only valid configs (e.g., "get all" operations)
- **Uses**: `testdata/valid-configs/` directory
- **Contains**: 5 valid kubeconfig files
- **Best for**: API endpoint tests, successful operations

#### `createTestServerInvalid(t *testing.T)`
- **Purpose**: Error testing scenarios
- **Uses**: `testdata/invalid-configs/` directory
- **Contains**: 1 invalid kubeconfig file
- **Best for**: Testing error handling, validation failures

#### `createTestServer(t *testing.T)`
- **Purpose**: Comprehensive testing including edge cases
- **Uses**: `testdata/mixed-configs/` directory
- **Contains**: 6 files (5 valid + 1 invalid)
- **Best for**: Testing file listing, mixed scenarios

#### `createTestServerForTemplates(t *testing.T)`
- **Purpose**: Template rendering tests
- **Uses**: Valid configs + copies templates to temp directory
- **Best for**: Web interface testing, template rendering

### Benefits of This Approach

1. **No File Copying**: Tests use testdata directories directly - much faster!
2. **Clear Test Intent**: Function names indicate what type of configs are used
3. **Isolated Test Scenarios**: Each test type uses appropriate data
4. **Easy Maintenance**: Add new test files to appropriate directory
5. **Unit Test Support**: `kubeconfigs/` directory provides source files for unit test copying

### Example Usage

```go
// Test successful operations with valid configs only
func TestSuccessfulOperation(t *testing.T) {
    server, _ := createTestServerValid(t)
    // Test will use 5 valid configs
}

// Test error handling with invalid configs
func TestErrorHandling(t *testing.T) {
    server, _ := createTestServerInvalid(t)
    // Test will use 1 invalid config
}

// Test comprehensive scenarios
func TestMixedScenarios(t *testing.T) {
    server, _ := createTestServer(t)
    // Test will use 6 configs (5 valid + 1 invalid)
}
```

## Running Tests

### Quick Test Run - Non-Interactive
```bash
# Run all tests without prompts (good for CI/scripts)
./test.sh --non-interactive

# Same as above, using shorter alias
./test.sh --ci

# Run all tests including integration tests
./test.sh --integration --non-interactive
```

### Interactive Test Run (Original Behavior)
```bash
# Make the test script executable
chmod +x test.sh

# Run the full test suite (will ask about integration tests)
./test.sh

# Run with integration tests
./test.sh --integration
```

### Command Line Options
```bash
# Show all available options
./test.sh --help

# Available options:
#   -i, --integration     Run integration tests
#   --no-integration      Skip integration tests (default)
#   --non-interactive     Run without prompts (good for CI)
#   --ci                  Alias for --non-interactive
#   -h, --help           Show help message
```

### Manual Test Execution
```bash
# Run specific test packages
go test ./cmd/gogetkubeconfig/ -v
go test ./internal/server/ -v

# Run with coverage
go test ./internal/server/ -cover -v

# Run integration tests only
go test ./cmd/gogetkubeconfig/ -run TestIntegration -v

# Run tests in short mode (skips integration tests)
go test -short ./...

# Run with race detection
go test -race ./...
```

## Test Coverage

The test suite provides comprehensive coverage:

- **Main Package**: ~69% coverage
- **Server Package**: ~76% coverage
- **Overall**: ~74% coverage

Coverage reports are generated in HTML format:
- `coverage.html` - Overall coverage
- `main.cover.html` - Main package coverage
- `server.cover.html` - Server package coverage

## Test Types

### Unit Tests
- Test individual functions and methods
- Use mocked dependencies where appropriate
- Fast execution, no external dependencies

### Integration Tests
- Test full HTTP server functionality
- Use real HTTP requests and responses
- Test end-to-end workflows
- Skipped in short mode (`go test -short`)

### Error Testing
- Test error conditions and edge cases
- Use invalid test data from `invalid-configs/`
- Verify proper error handling and logging

### Template Tests
- Test HTML template rendering
- Use copied templates in temp directories
- Verify web interface functionality

## Adding New Tests

### Adding Test Data
1. **Valid configs**: Add to `testdata/valid-configs/`
2. **Invalid configs**: Add to `testdata/invalid-configs/`
3. **Templates**: Add to `testdata/templates/`
4. **Mixed scenarios**: Files are automatically synced to `mixed-configs/`

### Writing Tests
1. Choose appropriate server creation function based on test needs
2. Use table-driven tests for multiple scenarios
3. Include both positive and negative test cases
4. Add integration tests for new endpoints

### Test Naming
- Use descriptive test function names
- Group related tests with subtests
- Follow Go testing conventions