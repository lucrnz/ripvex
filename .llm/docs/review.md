## Your Task
Perform a thorough analysis of this codebase to identify issues, bugs, security vulnerabilities, and problems that could prevent this application from being production-ready.

## Scope
- Analyze all source code files, configuration files, and infrastructure definitions
- Ignore the contents of the `.llm/docs` directory
- Do not flag the Go version as an issue (your knowledge may be outdated)
- Do not flag missing unit tests (out of scope)
- Do not flag missing E2E tests (out of scope)

## What to Look For

- Security
- Reliability
- Production Readiness
- Code Quality

## Issue Severity Categories

### Critical
Issues that pose an immediate security risk, could cause data loss or corruption, or would cause the application to fail catastrophically in production. These must be fixed before deployment.

### High
Issues that could lead to significant problems under certain conditions, such as performance degradation under load, security weaknesses that are harder to exploit, or bugs that affect core functionality. Should be addressed before production deployment.

### Medium
Issues that impact code maintainability, could cause problems in edge cases, or represent deviations from best practices that may lead to future bugs. Should be planned for remediation.

### Low
Minor code quality issues, style inconsistencies, or small improvements that would make the codebase better but pose no immediate risk. Address as time permits.

## Output

### 1. Review Report
Save your findings to `review.md` in the root directory of the project with this structure:
```
# Codebase Review

## Summary
Brief overview of findings organized by severity count.

## Critical Issues
For each issue include:
- File path and line number (if applicable)
- Description of the issue
- Why it's critical
- Suggested fix

## High Issues
[Same format as above]

## Medium Issues
[Same format as above]

## Low Issues
[Same format as above]
```

If no issues are found, save the file with "No issues found".

### 2. Review Results JSON
Save a file named `review-results.json` in the root directory of the project with the following structure:
```json
{
  "containsCriticalIssues": true
}
```

Set `containsCriticalIssues` to `true` if you found any Critical severity issues, otherwise set it to `false`.
