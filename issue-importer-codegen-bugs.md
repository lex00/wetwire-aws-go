# Importer Code Generation Bugs

## Summary

The `wetwire-aws import` command generates Go code with several categories of errors that prevent compilation. After importing 254 AWS CloudFormation sample templates, ~62 examples build successfully (50% success rate).

## Progress

### Fixed Issues
- [x] **Invalid Go identifiers** - Variable names with hyphens (e.g., `Port-1ICMP`) now sanitized to valid identifiers (`PortNeg1ICMP`)
- [x] **Unknown resource types** - Now skipped with a comment instead of generating broken `resources/unknown` import
- [x] **Excessive intrinsics imports** - Import only added when intrinsic types are actually used
- [x] **List() helper import** - Added intrinsics import when `List()` is used

### Remaining Issues

| Category | Count | Description |
|----------|-------|-------------|
| UNDEFINED_REF | 21 | CloudFormation Conditions referenced but not generated as Go variables |
| UNUSED_IMPORT | 20 | Intrinsics package still imported but not used in some cases |
| TYPE_MISMATCH | 8 | Using wrong types (e.g., `Ref` where `[]any` expected) |
| OTHER | 6 | Various other issues |
| UNKNOWN_FIELD | 3 | Generated struct field names that don't exist |
| CMD_DIR_CONFLICT | 3 | Package name conflicts with `cmd/` directory |
| MISSING_GOSUM | 1 | go.sum issues |

## Remaining Root Causes

### 1. Undefined Condition References (UNDEFINED_REF)

CloudFormation Conditions are referenced in `Fn::If` but the condition variables aren't always generated or accessible in the file that references them.

### 2. Remaining Unused Imports (UNUSED_IMPORT)

Some files still get intrinsics import added when not needed. Need to trace all code paths that add the import.

### 3. Type Mismatches (TYPE_MISMATCH)

Some fields expect specific types but receive incompatible values:
- Using `Ref` struct where `[]any` is expected
- Using resource references where attribute references are needed

### 4. Unknown Struct Fields (UNKNOWN_FIELD)

Generated field names don't match actual struct definitions:
- `ResourceTypeProp` in `ec2.CapacityReservationFleet_TagSpecification`
- `ManagedRuleGroupStatement` in `wafv2.RuleGroup_Statement`

### 5. Package Name Conflicts (CMD_DIR_CONFLICT)

When the package name matches the `cmd` directory name, Go build fails.

## Acceptance Criteria

- [ ] All 254 AWS sample templates import without syntax errors
- [ ] Generated code compiles successfully (`go build ./...`)
- [ ] No unused imports in generated code
- [ ] Unknown resource types are handled gracefully (skipped with comment)
- [ ] Variable names are valid Go identifiers
- [ ] Condition references resolve correctly
- [ ] Add linting rules to catch remaining issues that can't be fixed in importer
