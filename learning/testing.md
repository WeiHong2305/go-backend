# Go Testing Conventions

**File naming**: `*_test.go` - Go toolchain only compiles them during `go test`, never in production builds.

**Package naming**: 2 options:
- `package service` (same package) - can access unexported functions
- `package service_test` (external) - tests only the public API, like a real consumer would

**Function naming**: Starts with `Test` + uppercase letter

**Subtests** with t.Run

**Table-driven tests** - most common Go pattern

**No assertion library by default** - just use `if` statements

**Helper function**: mark with `t.Helper()` so error line numbers point to the caller

**Test location**: test files live next to the code they test

**Mocks**: no built-in mock framework. Common approaches:
- Hand-rolled structs with function fields
- `gomock`/`mockgen` for generated mocks
- Interfaces make everything testable - that's why Go emphasizes small interfaces

**Running**:
`go test ./...`
`go test ./internal/service`
`go test -run TestGetMovie`
`go test -v ./...`
`go test -cover ./...`