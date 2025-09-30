# Dead Code Cleanup Tracking

This file tracks the progress of removing dead code found by staticcheck and golangci-lint.
**DO NOT COMMIT THIS FILE** - it's for tracking purposes only.

## Progress Overview
- [ ] Unused Functions (3 total)
- [ ] Unused Struct Fields (9 total)
- [ ] Unused Function Parameters (6 total)
- [ ] Code Style Issues (4 total)
- [ ] Deprecated API Usages (2 total)

---

## 1. Unused Functions (3)

### 1.1 `(*Model).runDiffFormatter` - cmd/app/model_pager.go:84
- [ ] Remove function
- [ ] Verify no callers exist
- [ ] Run tests
- [ ] Build app
- [ ] Commit

### 1.2 `(*Model).initializeUpdateService` - cmd/app/upgrade.go:211
- [ ] Remove function
- [ ] Verify no callers exist
- [ ] Run tests
- [ ] Build app
- [ ] Commit

### 1.3 `(*Model).schedulePeriodicUpdateCheck` - cmd/app/upgrade.go:249
- [ ] Remove function
- [ ] Verify no callers exist
- [ ] Run tests
- [ ] Build app
- [ ] Commit

---

## 2. Unused Struct Fields (9)

### 2.1 ErrorAnalytics struct - pkg/errors/analytics.go
All fields unused (entire struct appears dead):
- [ ] Remove `mu` field (line 10)
- [ ] Remove `errorHistory` field (line 11)
- [ ] Remove `patterns` field (line 12)
- [ ] Remove `metrics` field (line 13)
- [ ] Remove `maxHistory` field (line 14)
- [ ] Check if entire struct/file can be removed
- [ ] Run tests
- [ ] Build app
- [ ] Commit

### 2.2 Error Handler - pkg/errors/handler.go
- [ ] Remove `errorHistory` field (line 41)
- [ ] Remove `historyMu` field (line 42)
- [ ] Run tests
- [ ] Build app
- [ ] Commit

### 2.3 EnhancedArgoApiService - pkg/services/enhanced.go
- [ ] Remove `watchCancel` field (line 18)
- [ ] Remove `mu` field (line 19)
- [ ] Run tests
- [ ] Build app
- [ ] Commit

---

## 3. Unused Function Parameters (6)

### 3.1 handleNoDiffModeKeys - cmd/app/input_handlers.go:392
- [ ] Remove unused `msg` parameter OR use it if needed
- [ ] Update function signature
- [ ] Run tests
- [ ] Build app
- [ ] Commit

### 3.2 openTextPager - cmd/app/model_pager.go:23
- [ ] Remove unused `title` parameter
- [ ] Update all call sites
- [ ] Run tests
- [ ] Build app
- [ ] Commit

### 3.3 parseLogLineParts - cmd/app/view.go:128
- [ ] Remove error return (always nil)
- [ ] Update all call sites
- [ ] Run tests
- [ ] Build app
- [ ] Commit

### 3.4 recoverStream - pkg/services/recovery.go:102
- [ ] Remove unused `originalErr` parameter
- [ ] Update all call sites
- [ ] Run tests
- [ ] Build app
- [ ] Commit

### 3.5 neatServiceAccount - pkg/neat/neat.go:181
- [ ] Remove error return (always nil)
- [ ] Update all call sites
- [ ] Run tests
- [ ] Build app
- [ ] Commit

### 3.6 neatEmpty - pkg/neat/neat.go:214
- [ ] Remove error return (always nil)
- [ ] Update all call sites
- [ ] Run tests
- [ ] Build app
- [ ] Commit

---

## 4. Code Style Issues (4)

### 4.1 Ineffectual assignment - cmd/app/view_banner.go:133
- [ ] Fix ineffectual assignment to `rb`
- [ ] Run tests
- [ ] Build app
- [ ] Commit

### 4.2 for-select pattern - cmd/app/upgrade.go:258
- [ ] Replace `for { select {} }` with `for range`
- [ ] Run tests
- [ ] Build app
- [ ] Commit

### 4.3 Nil check for map - cmd/app/view.go:953
- [ ] Remove unnecessary nil check before len()
- [ ] Run tests
- [ ] Build app
- [ ] Commit

### 4.4 Deprecated Style.Copy() - cmd/app/view_modals.go:341,346,352,356
- [ ] Replace `.Copy()` with direct assignment (4 locations)
- [ ] Run tests
- [ ] Build app
- [ ] Commit

---

## 5. Deprecated API Usages (2)

### 5.1 strings.Title - cmd/app/view.go:934
- [ ] Replace `strings.Title` with `golang.org/x/text/cases`
- [ ] Add import if needed
- [ ] Run tests
- [ ] Build app
- [ ] Commit

### 5.2 pool.Subjects - pkg/trust/trust_test.go (6 locations)
- [ ] Update test to use modern API
- [ ] Run tests
- [ ] Build app
- [ ] Commit

---

## Notes
- Each change should be committed separately
- Always run tests and build before committing
- Test command: `go test ./...`
- Build command: `go build ./cmd/app`
