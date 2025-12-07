## Your Task
Perform a thorough analysis of this Go CLI application to identify issues, bugs, security vulnerabilities, and problems that could prevent it from being production-ready.

## Analysis Approach
### Step 1: Understand the Codebase
Before flagging issues, first understand:
- Entry points and command structure
- Key dependencies (go.mod)
- Configuration handling

### Step 2: Systematic Review
Analyze in this order:
1. Entry points (main.go, cmd/ files)
2. Core business logic
3. External integrations (HTTP clients, databases, file I/O)
4. Configuration and flag handling
5. Utility/helper packages

## Scope
- Analyze all `.go` files, `go.mod`, `go.sum`, and configuration files
- Review any Dockerfiles, CI configs, or infrastructure definitions
- Ignore the contents of the `.llm/docs` directory
- Do not flag the Go version as an issue
- Do not flag missing unit tests or E2E tests

## What to Look For

### Security
- Command injection via exec.Command with unsanitized input
- Path traversal vulnerabilities in file operations
- Hardcoded secrets, API keys, or credentials
- Insecure TLS configurations (disabled verification, weak ciphers)
- SQL injection or other injection vulnerabilities
- Unsafe deserialization of user input
- Sensitive data in logs or error messages
- Improper file permissions

### Go-Specific Issues
- Nil pointer dereferences (especially after type assertions without ok check)
- Goroutine leaks (goroutines that never terminate)
- Race conditions (shared state without synchronization)
- Deferred function calls in loops
- Unchecked errors (especially from Close(), Write(), etc.)
- Context misuse (not propagating cancellation, ignoring deadlines)
- Improper use of sync primitives (unlock without lock, double unlock)
- Resource leaks (unclosed files, HTTP response bodies, database connections)

### CLI-Specific Issues
- Missing or incorrect exit codes
- No graceful shutdown on SIGINT/SIGTERM
- Improper stdin/stdout/stderr handling
- Flag parsing edge cases
- Missing or misleading help text for commands
- Silent failures that should produce user-visible errors

### Reliability
- Panics that should be errors
- Missing error context (bare errors without wrapping)
- Infinite loops or recursion without bounds
- Missing timeouts on network operations
- Unbounded resource consumption (memory, goroutines, file handles)
- Missing input validation

### Production Readiness
- Inadequate logging (missing context, wrong levels)
- Missing or broken health checks
- Configuration that only works in development
- Debug code or TODO comments indicating incomplete features
- Improper handling of environment variables

## Issue Severity Categories

### Critical
Immediate security risks, data loss/corruption potential, or catastrophic failures:
- Remote code execution vulnerabilities
- Authentication/authorization bypasses
- Unhandled panics in critical paths
- Data corruption bugs
- Credential exposure

### High
Significant problems under certain conditions:
- Race conditions in concurrent code
- Resource leaks that accumulate over time
- Security weaknesses requiring specific conditions to exploit
- Bugs affecting core functionality
- Missing input validation on external input

### Medium
Maintainability impacts or edge case problems:
- Error handling that swallows context
- Code patterns that are brittle or error-prone
- Deviations from Go idioms that could cause future bugs
- Missing timeouts that could cause hangs

### Low
Minor improvements with no immediate risk:
- Code style inconsistencies
- Opportunities for simplification
- Minor performance improvements
- Documentation gaps in public APIs

## What NOT to Flag
- Idiomatic Go patterns even if they seem unusual
- Use of standard library functions as intended
- Code style preferences (gofmt handles this)
- Missing features unless they're documented as TODO with security implications
- Dependencies being "old" unless there are known security vulnerabilities
- Lack of generics in code that predates Go 1.18

## Output

### 1. Review Report
Save findings to `review.md`:

```
# Codebase Review

## Summary
Brief overview: what the application does, total issues found by severity.

| Severity | Count |
|----------|-------|
| Critical | X     |
| High     | X     |
| Medium   | X     |
| Low      | X     |

## Critical Issues

### [C-1] <Short descriptive title>
**File:** `path/to/file.go:42`
**Category:** Security | Reliability | etc.

**Description:**
What the issue is and why it matters.

**Code:**
```go
// The problematic code snippet
```

**Impact:**
Specific consequences if not addressed.

**Suggested Fix:**
```go
// How to fix it
```
---

## High Issues
[Same format]

## Medium Issues
[Same format]

## Low Issues
[Same format]

## Notes
Any observations about the codebase that don't fit into issues but are worth mentioning.
```

### 2. Review Results JSON
Save to `review-results.json`:
```json
{
  "containsCriticalIssues": true,
}
```

Set `containsCriticalIssues` to `true` only if Critical severity issues were found.
```
