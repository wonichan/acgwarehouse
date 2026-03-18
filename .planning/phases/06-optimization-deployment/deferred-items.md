# Deferred Items

## Pre-existing Issues (Out of Scope for 06-04)

### job_repository_test.go compilation errors
- **File:** `internal/repository/job_repository_test.go`
- **Issue:** Tests reference undefined methods: `FindRecent`, `FindFailed`, `UpdateStatus`, `CountByStatus`
- **Impact:** Repository package tests fail to compile
- **Note:** This is a pre-existing issue unrelated to image list pagination work