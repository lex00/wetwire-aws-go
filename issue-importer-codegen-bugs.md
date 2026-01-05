# Importer Code Generation Bugs

## Summary

The `wetwire-aws import` command generates Go code with several categories of errors that prevent compilation. After importing 254 AWS CloudFormation sample templates, ~69 examples build successfully (55.6% success rate).

## Progress

### Fixed Issues
- [x] **Invalid Go identifiers** - Variable names with hyphens (e.g., `Port-1ICMP`) now sanitized to valid identifiers (`PortNeg1ICMP`)
- [x] **Unknown resource types** - Now skipped with a comment instead of generating broken `resources/unknown` import
- [x] **Excessive intrinsics imports** - Import only added when intrinsic types are actually used
- [x] **List() helper import** - Added intrinsics import when `List()` is used
- [x] **Parameters in conditions** - Pre-scan conditions for parameter references before generating params
- [x] **Lint rules added** - WAW013-WAW016 (undefined refs, unused imports, Ref{}/GetAtt{} style)
- [x] **Style enforcement** - Importer never generates Ref{} or GetAtt{} (uses direct refs and field access)
- [x] **Integration test** - TestExamplesBuild validates 12 complex templates compile
- [x] **go mod tidy** - Importer now runs `go mod tidy` after generating scaffold files
- [x] **go.mod replace directive** - Fixed path in generated examples

### Remaining Issues

| Category | Count | Description |
|----------|-------|-------------|
| UNUSED_IMPORT | ~21 | Intrinsics package still imported but not used in some edge cases |
| TYPE_MISMATCH | ~17 | Using wrong types (e.g., `Ref` where `[]any` expected) |
| OTHER | ~8 | Various other issues |
| UNDEFINED_REF | ~4 | CloudFormation Conditions referenced but not generated |
| CMD_DIR_CONFLICT | 3 | Package name conflicts with `cmd/` directory |

## Remaining Root Causes

### 1. Remaining Unused Imports (UNUSED_IMPORT)

Some edge cases still get intrinsics import added when not needed. The WAW014 lint rule catches these.

### 2. Type Mismatches (TYPE_MISMATCH)

Some fields expect specific types but receive incompatible values:
- Using `Ref` struct where `[]any` is expected
- Using resource references where attribute references are needed

### 3. Undefined Condition References (UNDEFINED_REF)

A few CloudFormation Conditions are referenced in `Fn::If` but the condition variables aren't generated or accessible.

### 4. Package Name Conflicts (CMD_DIR_CONFLICT)

When the package name matches the `cmd` directory name, Go build fails.

## Acceptance Criteria

- [x] All 254 AWS sample templates import without syntax errors (252/254 = 99%)
- [ ] Generated code compiles successfully (`go build ./...`) - 69/124 = 55.6%
- [x] Unknown resource types are handled gracefully (skipped with comment)
- [x] Variable names are valid Go identifiers
- [x] Add linting rules to catch remaining issues (WAW013-WAW016)
- [x] Integration test to catch regressions (TestExamplesBuild)
- [ ] No unused imports in generated code (WAW014 catches, ~21 remaining)
- [ ] Condition references resolve correctly (~4 remaining)
