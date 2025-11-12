# Testing Guide for NOFX

This document provides comprehensive information about the testing infrastructure in the NOFX project.

## Table of Contents

- [Overview](#overview)
- [Frontend Testing](#frontend-testing)
- [Backend Testing](#backend-testing)
- [Running Tests](#running-tests)
- [Writing Tests](#writing-tests)
- [Test Coverage](#test-coverage)
- [CI/CD Integration](#cicd-integration)
- [Best Practices](#best-practices)

## Overview

NOFX uses a comprehensive testing suite to ensure code quality and reliability:

- **Frontend**: Vitest + React Testing Library
- **Backend**: Go's built-in testing package + testify (optional)
- **CI/CD**: Automated tests run on every pull request

### Why Testing Matters

As an AI-powered cryptocurrency trading system, NOFX handles:
- **Financial transactions** - bugs can result in monetary losses
- **Real-time trading decisions** - reliability is critical
- **User authentication** - security vulnerabilities must be prevented
- **External API integrations** - proper error handling is essential

## Frontend Testing

### Technology Stack

- **Vitest**: Fast, Vite-native test runner
- **React Testing Library**: Component testing utilities
- **@testing-library/user-event**: User interaction simulation
- **jsdom/happy-dom**: Browser environment simulation

### Test Structure

```
web/
├── src/
│   ├── components/
│   │   ├── LoginPage.tsx
│   │   └── LoginPage.test.tsx       # Component test
│   ├── hooks/
│   │   ├── useSystemConfig.ts
│   │   └── useSystemConfig.test.ts  # Hook test
│   ├── test/
│   │   ├── setup.ts                 # Test configuration
│   │   └── utils.tsx                # Test utilities
│   └── ...
├── vitest.config.ts                 # Vitest configuration
└── package.json                     # Test scripts
```

### Running Frontend Tests

```bash
cd web

# Run tests in watch mode (interactive)
npm test

# Run tests once (CI mode)
npm run test:run

# Run tests with UI
npm run test:ui

# Run tests with coverage
npm run test:coverage

# Run tests in watch mode
npm run test:watch
```

### Frontend Test Examples

#### Component Test
```typescript
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { LoginPage } from './LoginPage';

describe('LoginPage', () => {
  it('renders login form', () => {
    render(<LoginPage />);
    expect(screen.getByRole('heading')).toBeInTheDocument();
  });

  it('handles form submission', async () => {
    render(<LoginPage />);
    const emailInput = screen.getByLabelText(/email/i);
    await userEvent.type(emailInput, 'test@example.com');
    // ... more assertions
  });
});
```

#### Hook Test
```typescript
import { renderHook, waitFor } from '@testing-library/react';
import { useSystemConfig } from './useSystemConfig';

describe('useSystemConfig', () => {
  it('fetches config successfully', async () => {
    const { result } = renderHook(() => useSystemConfig());

    expect(result.current.loading).toBe(true);

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(result.current.config).toBeDefined();
  });
});
```

## Backend Testing

### Technology Stack

- **Go testing package**: Built-in testing framework
- **testify/assert**: Assertion library (install when network available)
- **testify/mock**: Mocking utilities (install when network available)
- **Race detector**: Concurrent code testing

### Test Structure

```
nofx/
├── auth/
│   ├── auth.go
│   └── auth_test.go           # Auth package tests
├── pool/
│   ├── coin_pool.go
│   └── coin_pool_test.go      # Coin pool tests
├── config/
│   ├── database.go
│   └── database_test.go       # Database integration tests
└── ...
```

### Running Backend Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with race detection
go test -race ./...

# Run tests with coverage
go test -cover ./...

# Generate detailed coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run tests in a specific package
go test ./auth -v

# Run a specific test
go test ./auth -run TestHashPassword -v

# Run benchmarks
go test -bench=. ./...
```

### Backend Test Examples

#### Unit Test
```go
func TestHashPassword(t *testing.T) {
    password := "testPassword123!"

    hash, err := HashPassword(password)
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }

    if hash == password {
        t.Error("Hash should not equal plain password")
    }
}
```

#### Table-Driven Test
```go
func TestCheckPassword(t *testing.T) {
    tests := []struct {
        name     string
        password string
        hash     string
        expected bool
    }{
        {
            name:     "Correct password",
            password: "correct",
            hash:     hashPassword("correct"),
            expected: true,
        },
        {
            name:     "Wrong password",
            password: "wrong",
            hash:     hashPassword("correct"),
            expected: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := CheckPassword(tt.password, tt.hash)
            if result != tt.expected {
                t.Errorf("Expected %v, got %v", tt.expected, result)
            }
        })
    }
}
```

#### Integration Test
```go
func TestDatabaseOperations(t *testing.T) {
    // Create temporary test database
    tmpDir := t.TempDir()
    db, err := NewDatabase(filepath.Join(tmpDir, "test.db"))
    if err != nil {
        t.Fatalf("Failed to create database: %v", err)
    }
    defer db.Close()

    // Test database operations
    err = db.SetSystemConfig("key", "value")
    if err != nil {
        t.Fatalf("Failed to set config: %v", err)
    }

    value, err := db.GetSystemConfig("key")
    if err != nil {
        t.Fatalf("Failed to get config: %v", err)
    }

    if value != "value" {
        t.Errorf("Expected 'value', got %s", value)
    }
}
```

#### Benchmark Test
```go
func BenchmarkHashPassword(b *testing.B) {
    password := "testPassword123!"
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = HashPassword(password)
    }
}
```

## Writing Tests

### General Guidelines

1. **Test naming**: Use descriptive names that explain what is being tested
   - Good: `TestHashPassword_ValidInput_ReturnsHash`
   - Bad: `TestFunc1`

2. **Test structure**: Follow the AAA pattern
   - **Arrange**: Set up test data and dependencies
   - **Act**: Execute the code being tested
   - **Assert**: Verify the results

3. **Test isolation**: Each test should be independent
   - Use `beforeEach`/`afterEach` in frontend tests
   - Use `t.TempDir()` for backend tests requiring files

4. **Mock external dependencies**: Don't call real APIs in tests
   - Frontend: Use `vi.mock()` or MSW
   - Backend: Use interfaces and test doubles

### What to Test

#### High Priority (Must Test)
- **Authentication logic** - security critical
- **Trading operations** - financial impact
- **Decision engine** - core business logic
- **Payment/transaction handling** - monetary operations
- **User input validation** - security and data integrity

#### Medium Priority (Should Test)
- **API endpoints** - integration points
- **Database operations** - data persistence
- **State management** - application consistency
- **Utility functions** - reusable logic

#### Low Priority (Nice to Have)
- **UI components** - visual consistency
- **Styling logic** - presentation
- **Configuration loading** - initialization

### What NOT to Test

- Third-party library internals
- Generated code
- Simple getters/setters with no logic
- Trivial functions that just pass data through

## Test Coverage

### Coverage Goals

- **Backend**: Aim for 70-80% overall (90%+ for critical paths)
  - Trading logic: 90%+
  - Authentication: 90%+
  - Decision engine: 90%+
  - Database operations: 80%+
  - Utilities: 60%+

- **Frontend**: Aim for 60-70% overall
  - Critical user flows: 80%+
  - Components: 60%+
  - Hooks: 70%+
  - Utilities: 60%+

### Viewing Coverage Reports

**Frontend:**
```bash
cd web
npm run test:coverage
# Open web/coverage/index.html in browser
```

**Backend:**
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Coverage Configuration

Frontend coverage thresholds are configured in `web/vitest.config.ts`:

```typescript
coverage: {
  thresholds: {
    lines: 60,
    functions: 60,
    branches: 60,
    statements: 60,
  },
}
```

## CI/CD Integration

### Automated Testing

Tests run automatically on every pull request via GitHub Actions (`.github/workflows/pr-checks.yml`):

1. **Go tests**: `go test -v -race -coverprofile=coverage.out ./...`
2. **Frontend tests**: `npm run test:run`
3. **Coverage reports**: Generated for both frontend and backend

### Pull Request Requirements

Before merging, PRs must:
- ✅ Pass all backend tests
- ✅ Pass all frontend tests
- ✅ Pass security checks
- ✅ Build successfully
- ℹ️ Maintain reasonable test coverage (recommended but not enforced)

### Running Tests Locally Before Push

**Quick check:**
```bash
# Backend
go test ./... && go build

# Frontend
cd web && npm run test:run && npm run build
```

**Full check (matching CI):**
```bash
# Backend
go fmt ./...
go vet ./...
go test -v -race -coverprofile=coverage.out ./...
go build -v

# Frontend
cd web
npm ci
npm run test:run
npm run test:coverage
npm run build
```

## Best Practices

### 1. Mock External Dependencies

**Frontend:**
```typescript
// Mock API calls
vi.mock('../lib/api', () => ({
  fetchTraders: vi.fn().mockResolvedValue([...]),
}));

// Mock external services
vi.mock('../services/exchange', () => ({
  BinanceAPI: vi.fn().mockImplementation(() => ({
    getTrades: vi.fn().mockResolvedValue([]),
  })),
}));
```

**Backend:**
```go
// Use interfaces for dependencies
type ExchangeAPI interface {
    GetPrice(symbol string) (float64, error)
}

// Create mock implementation for tests
type MockExchange struct {
    PriceToReturn float64
    ErrorToReturn error
}

func (m *MockExchange) GetPrice(symbol string) (float64, error) {
    return m.PriceToReturn, m.ErrorToReturn
}
```

### 2. Test Error Paths

Always test both success and failure cases:

```typescript
describe('fetchData', () => {
  it('handles success', async () => {
    // Test successful data fetch
  });

  it('handles network error', async () => {
    // Test network failure
  });

  it('handles invalid response', async () => {
    // Test malformed data
  });
});
```

### 3. Use Descriptive Assertions

```typescript
// ❌ Bad
expect(result).toBe(true);

// ✅ Good
expect(userIsAuthenticated).toBe(true);
expect(result.errors).toHaveLength(0);
expect(trader.status).toBe('active');
```

### 4. Clean Up Resources

**Frontend:**
```typescript
afterEach(() => {
  cleanup(); // Automatically done by testing-library
  vi.clearAllMocks();
});
```

**Backend:**
```go
func TestWithDatabase(t *testing.T) {
    tmpDir := t.TempDir() // Automatically cleaned up
    db, err := NewDatabase(filepath.Join(tmpDir, "test.db"))
    if err != nil {
        t.Fatal(err)
    }
    defer db.Close()

    // Test code...
}
```

### 5. Avoid Test Interdependence

```typescript
// ❌ Bad - tests depend on execution order
let sharedState;
test('creates user', () => {
  sharedState = createUser();
});
test('deletes user', () => {
  deleteUser(sharedState); // Breaks if first test fails
});

// ✅ Good - each test is independent
test('creates user', () => {
  const user = createUser();
  expect(user).toBeDefined();
});
test('deletes user', () => {
  const user = createUser();
  const result = deleteUser(user);
  expect(result.success).toBe(true);
});
```

### 6. Test Race Conditions (Backend)

For concurrent code, always use the race detector:

```bash
go test -race ./trader
```

```go
func TestConcurrentTrading(t *testing.T) {
    trader := NewTrader()

    // Run multiple operations concurrently
    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            trader.ExecuteTrade()
        }()
    }
    wg.Wait()

    // Verify state is consistent
}
```

## Troubleshooting

### Common Issues

**Frontend:**

1. **Import errors**: Check path aliases in `vitest.config.ts`
2. **Module not found**: Ensure mocks use correct paths
3. **Timeout errors**: Increase timeout for slow operations
4. **DOM not available**: Verify `environment: 'jsdom'` in config

**Backend:**

1. **Import cycle**: Refactor to break circular dependencies
2. **Race conditions**: Run with `-race` flag to detect
3. **Test database conflicts**: Use `t.TempDir()` for isolation
4. **Port already in use**: Don't hardcode ports in tests

### Getting Help

- Review example tests in the codebase
- Check the [Vitest documentation](https://vitest.dev/)
- Read [Go testing package docs](https://pkg.go.dev/testing)
- Consult [React Testing Library guides](https://testing-library.com/docs/react-testing-library/intro/)

## Future Improvements

- [ ] Add E2E tests with Playwright/Cypress
- [ ] Implement visual regression testing
- [ ] Add performance/load testing
- [ ] Set up mutation testing
- [ ] Create test data factories
- [ ] Add contract testing for APIs
- [ ] Implement snapshot testing for components

## Contributing

When adding new features:

1. Write tests FIRST (TDD approach recommended)
2. Ensure tests pass locally before pushing
3. Maintain or improve code coverage
4. Update this documentation if adding new testing patterns

## License

This testing infrastructure is part of the NOFX project.
