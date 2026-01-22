package lint

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

type InvalidEnumValue struct{}

func (r InvalidEnumValue) ID() string { return "WAW011" }
func (r InvalidEnumValue) Description() string {
	return "Validate enum property values against allowed values"
}

// enumFieldInfo holds the service and enum name for a field.
type enumFieldInfo struct {
	service  string
	enumName string
}

// enumFields maps CloudFormation property names to their enum info.
// This allows the linter to validate string literals assigned to these fields.
var enumFields = map[string]enumFieldInfo{
	// S3 enums
	"StorageClass": {"s3", "StorageClass"},

	// EC2 enums
	"InstanceType": {"ec2", "InstanceType"},

	// Lambda enums
	"Runtime":     {"lambda", "Runtime"},
	"PackageType": {"lambda", "PackageType"},

	// RDS enums
	"Engine":        {"rds", "Engine"},
	"EngineVersion": {"rds", "EngineVersion"},

	// DynamoDB enums
	"BillingMode":    {"dynamodb", "BillingMode"},
	"StreamViewType": {"dynamodb", "StreamViewType"},
	"TableClass":     {"dynamodb", "TableClass"},

	// ECS enums
	"LaunchType":         {"ecs", "LaunchType"},
	"NetworkMode":        {"ecs", "NetworkMode"},
	"SchedulingStrategy": {"ecs", "SchedulingStrategy"},
}

func (r InvalidEnumValue) Check(file *ast.File, fset *token.FileSet) []Issue {
	var issues []Issue

	// Check if the enums package is available
	hasEnumsImport := false
	for _, imp := range file.Imports {
		if imp.Path != nil && strings.Contains(imp.Path.Value, "cloudformation-schema-go/enums") {
			hasEnumsImport = true
			break
		}
	}

	ast.Inspect(file, func(n ast.Node) bool {
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}

		// Get field name
		fieldIdent, ok := kv.Key.(*ast.Ident)
		if !ok {
			return true
		}

		// Check if this is a known enum field
		enumInfo, ok := enumFields[fieldIdent.Name]
		if !ok {
			return true
		}

		// Check if value is a string literal
		lit, ok := kv.Value.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			// Skip non-string values (could be intrinsics, selectors, etc.)
			return true
		}

		value := strings.Trim(lit.Value, `"`)

		// Use the enums package to validate
		if !isValidEnumValue(enumInfo.service, enumInfo.enumName, value) {
			pos := fset.Position(lit.Pos())
			allowed := getAllowedEnumValues(enumInfo.service, enumInfo.enumName)

			// Limit displayed values if there are too many
			displayAllowed := allowed
			if len(allowed) > 5 {
				displayAllowed = append(allowed[:5], "...")
			}

			suggestion := fmt.Sprintf("Use one of: %s", strings.Join(displayAllowed, ", "))
			if !hasEnumsImport {
				suggestion += " (or import enums package and use constants)"
			}

			issues = append(issues, Issue{
				Rule:       r.ID(),
				Message:    fmt.Sprintf("Invalid %s value: %q", fieldIdent.Name, value),
				Suggestion: suggestion,
				File:       pos.Filename,
				Line:       pos.Line,
				Column:     pos.Column,
				Severity:   SeverityError,
			})
		}

		return true
	})

	return issues
}

// isValidEnumValue checks if a value is valid for the given enum.
// This is a local implementation to avoid import cycles with the enums package.
func isValidEnumValue(service, enumName, value string) bool {
	allowed := getAllowedEnumValues(service, enumName)
	for _, v := range allowed {
		if v == value {
			return true
		}
	}
	return false
}

// getAllowedEnumValues returns the allowed values for an enum.
// This is embedded here to avoid runtime dependency on the enums package.
var enumAllowedValues = map[string]map[string][]string{
	"s3": {
		"StorageClass": {
			"STANDARD", "REDUCED_REDUNDANCY", "STANDARD_IA", "ONEZONE_IA",
			"INTELLIGENT_TIERING", "GLACIER", "DEEP_ARCHIVE", "OUTPOSTS",
			"GLACIER_IR", "SNOW", "EXPRESS_ONEZONE",
		},
	},
	"ec2": {
		"InstanceType": {
			// Common instance types (subset for linting purposes)
			"t2.micro", "t2.small", "t2.medium", "t2.large", "t2.xlarge", "t2.2xlarge",
			"t3.micro", "t3.small", "t3.medium", "t3.large", "t3.xlarge", "t3.2xlarge",
			"t3a.micro", "t3a.small", "t3a.medium", "t3a.large", "t3a.xlarge", "t3a.2xlarge",
			"m5.large", "m5.xlarge", "m5.2xlarge", "m5.4xlarge", "m5.8xlarge", "m5.12xlarge",
			"m6i.large", "m6i.xlarge", "m6i.2xlarge", "m6i.4xlarge", "m6i.8xlarge",
			"c5.large", "c5.xlarge", "c5.2xlarge", "c5.4xlarge", "c5.9xlarge",
			"r5.large", "r5.xlarge", "r5.2xlarge", "r5.4xlarge", "r5.8xlarge",
		},
	},
	"lambda": {
		"Runtime": {
			"nodejs18.x", "nodejs20.x", "nodejs22.x",
			"python3.9", "python3.10", "python3.11", "python3.12", "python3.13",
			"java11", "java17", "java21",
			"dotnet6", "dotnet8",
			"ruby3.2", "ruby3.3",
			"provided", "provided.al2", "provided.al2023",
		},
		"PackageType":  {"Zip", "Image"},
		"Architecture": {"x86_64", "arm64"},
	},
	"rds": {
		"Engine": {
			"mysql", "mariadb", "postgres", "oracle-ee", "oracle-se2",
			"sqlserver-ee", "sqlserver-se", "sqlserver-ex", "sqlserver-web",
			"aurora", "aurora-mysql", "aurora-postgresql",
		},
	},
	"dynamodb": {
		"BillingMode":    {"PROVISIONED", "PAY_PER_REQUEST"},
		"StreamViewType": {"KEYS_ONLY", "NEW_IMAGE", "OLD_IMAGE", "NEW_AND_OLD_IMAGES"},
		"TableClass":     {"STANDARD", "STANDARD_INFREQUENT_ACCESS"},
	},
	"ecs": {
		"LaunchType":         {"EC2", "FARGATE", "EXTERNAL"},
		"NetworkMode":        {"bridge", "host", "awsvpc", "none"},
		"SchedulingStrategy": {"REPLICA", "DAEMON"},
	},
}

func getAllowedEnumValues(service, enumName string) []string {
	if svc, ok := enumAllowedValues[service]; ok {
		if vals, ok := svc[enumName]; ok {
			return vals
		}
	}
	return nil
}

// PreferEnumConstant detects raw string literals used for enum properties
// and suggests using typed enum constants instead.
//
// Example:
//
//	// Bad - raw string literal
//	Runtime: "python3.12",
//	StorageClass: "STANDARD",
//
//	// Good - typed enum constants
//	Runtime: enums.LambdaRuntimePython312,
//	StorageClass: enums.S3StorageClassStandard,
type PreferEnumConstant struct{}

func (r PreferEnumConstant) ID() string { return "WAW012" }
func (r PreferEnumConstant) Description() string {
	return "Use typed enum constants instead of raw strings"
}

// enumConstantMap maps (service, enumName, value) to the constant name.
// This is used to suggest the appropriate constant when a raw string is detected.
var enumConstantMap = map[string]map[string]map[string]string{
	"lambda": {
		"Runtime": {
			"python3.9":       "LambdaRuntimePython39",
			"python3.10":      "LambdaRuntimePython310",
			"python3.11":      "LambdaRuntimePython311",
			"python3.12":      "LambdaRuntimePython312",
			"python3.13":      "LambdaRuntimePython313",
			"nodejs18.x":      "LambdaRuntimeNodejs18X",
			"nodejs20.x":      "LambdaRuntimeNodejs20X",
			"nodejs22.x":      "LambdaRuntimeNodejs22X",
			"java11":          "LambdaRuntimeJava11",
			"java17":          "LambdaRuntimeJava17",
			"java21":          "LambdaRuntimeJava21",
			"dotnet6":         "LambdaRuntimeDotnet6",
			"dotnet8":         "LambdaRuntimeDotnet8",
			"ruby3.2":         "LambdaRuntimeRuby32",
			"ruby3.3":         "LambdaRuntimeRuby33",
			"provided.al2":    "LambdaRuntimeProvidedAl2",
			"provided.al2023": "LambdaRuntimeProvidedAl2023",
		},
		"PackageType": {
			"Zip":   "LambdaPackageTypeZip",
			"Image": "LambdaPackageTypeImage",
		},
		"Architecture": {
			"x86_64": "LambdaArchitectureX8664",
			"arm64":  "LambdaArchitectureArm64",
		},
	},
	"s3": {
		"StorageClass": {
			"STANDARD":            "S3StorageClassStandard",
			"REDUCED_REDUNDANCY":  "S3StorageClassReducedRedundancy",
			"STANDARD_IA":         "S3StorageClassStandardIa",
			"ONEZONE_IA":          "S3StorageClassOnezoneIa",
			"INTELLIGENT_TIERING": "S3StorageClassIntelligentTiering",
			"GLACIER":             "S3StorageClassGlacier",
			"DEEP_ARCHIVE":        "S3StorageClassDeepArchive",
			"GLACIER_IR":          "S3StorageClassGlacierIr",
		},
		"ServerSideEncryption": {
			"AES256":       "S3ServerSideEncryptionAes256",
			"aws:kms":      "S3ServerSideEncryptionAwsKms",
			"aws:kms:dsse": "S3ServerSideEncryptionAwsKmsDsse",
		},
	},
	"ec2": {
		"VolumeType": {
			"gp2":      "Ec2VolumeTypeGp2",
			"gp3":      "Ec2VolumeTypeGp3",
			"io1":      "Ec2VolumeTypeIo1",
			"io2":      "Ec2VolumeTypeIo2",
			"st1":      "Ec2VolumeTypeSt1",
			"sc1":      "Ec2VolumeTypeSc1",
			"standard": "Ec2VolumeTypeStandard",
		},
	},
	"ecs": {
		"LaunchType": {
			"EC2":      "EcsLaunchTypeEc2",
			"FARGATE":  "EcsLaunchTypeFargate",
			"EXTERNAL": "EcsLaunchTypeExternal",
		},
		"NetworkMode": {
			"bridge": "EcsNetworkModeBridge",
			"host":   "EcsNetworkModeHost",
			"awsvpc": "EcsNetworkModeAwsvpc",
			"none":   "EcsNetworkModeNone",
		},
		"SchedulingStrategy": {
			"REPLICA": "EcsSchedulingStrategyReplica",
			"DAEMON":  "EcsSchedulingStrategyDaemon",
		},
	},
	"dynamodb": {
		"BillingMode": {
			"PROVISIONED":     "DynamodbBillingModeProvisioned",
			"PAY_PER_REQUEST": "DynamodbBillingModePayPerRequest",
		},
		"StreamViewType": {
			"KEYS_ONLY":          "DynamodbStreamViewTypeKeysOnly",
			"NEW_IMAGE":          "DynamodbStreamViewTypeNewImage",
			"OLD_IMAGE":          "DynamodbStreamViewTypeOldImage",
			"NEW_AND_OLD_IMAGES": "DynamodbStreamViewTypeNewAndOldImages",
		},
		"TableClass": {
			"STANDARD":                   "DynamodbTableClassStandard",
			"STANDARD_INFREQUENT_ACCESS": "DynamodbTableClassStandardInfrequentAccess",
		},
	},
}

// enumFieldToService maps field names to their service for constant lookup.
var enumFieldToService = map[string]string{
	"Runtime":            "lambda",
	"PackageType":        "lambda",
	"Architectures":      "lambda",
	"StorageClass":       "s3",
	"SSEAlgorithm":       "s3",
	"VolumeType":         "ec2",
	"LaunchType":         "ecs",
	"NetworkMode":        "ecs",
	"SchedulingStrategy": "ecs",
	"BillingMode":        "dynamodb",
	"StreamViewType":     "dynamodb",
	"TableClass":         "dynamodb",
}

// enumFieldToEnumName maps field names to enum names (for cases where they differ).
var enumFieldToEnumName = map[string]string{
	"SSEAlgorithm":  "ServerSideEncryption",
	"Architectures": "Architecture",
}

func (r PreferEnumConstant) Check(file *ast.File, fset *token.FileSet) []Issue {
	var issues []Issue

	ast.Inspect(file, func(n ast.Node) bool {
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}

		// Get field name
		fieldIdent, ok := kv.Key.(*ast.Ident)
		if !ok {
			return true
		}

		// Check if this is a known enum field
		service, ok := enumFieldToService[fieldIdent.Name]
		if !ok {
			return true
		}

		// Determine enum name (may differ from field name)
		enumName := fieldIdent.Name
		if mapped, ok := enumFieldToEnumName[fieldIdent.Name]; ok {
			enumName = mapped
		}

		// Check if value is a string literal
		lit, ok := kv.Value.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			// Skip non-string values (could be constants, selectors, etc.)
			return true
		}

		value := strings.Trim(lit.Value, `"`)

		// Look up the constant name
		if serviceEnums, ok := enumConstantMap[service]; ok {
			if enumValues, ok := serviceEnums[enumName]; ok {
				if constName, ok := enumValues[value]; ok {
					pos := fset.Position(lit.Pos())
					issues = append(issues, Issue{
						Rule:       r.ID(),
						Message:    fmt.Sprintf("Use enums.%s instead of %q", constName, value),
						Suggestion: "enums." + constName,
						File:       pos.Filename,
						Line:       pos.Line,
						Column:     pos.Column,
						Severity:   SeverityWarning,
					})
				}
			}
		}

		return true
	})

	return issues
}

// UndefinedReference detects identifiers that look like resource/parameter
// references but might not be defined (useful for catching import codegen issues).
//
// This is a heuristic check - it flags PascalCase identifiers used in
// field values that aren't common patterns like intrinsics or type names.
type UndefinedReference struct{}

func (r UndefinedReference) ID() string { return "WAW013" }
func (r UndefinedReference) Description() string {
	return "Potential undefined reference (resource or parameter)"
}

// knownIdentifiers are common identifiers that shouldn't be flagged
var knownIdentifiers = map[string]bool{
	"true": true, "false": true, "nil": true,
	// Intrinsics from dot import
	"Sub": true, "Ref": true, "GetAtt": true, "Join": true, "Select": true,
	"If": true, "Equals": true, "And": true, "Or": true, "Not": true,
	"Base64": true, "Split": true, "FindInMap": true, "Cidr": true,
	"GetAZs": true, "ImportValue": true, "Condition": true, "Transform": true,
	// Pseudo-parameters
	"AWS_REGION": true, "AWS_ACCOUNT_ID": true, "AWS_STACK_NAME": true,
	"AWS_STACK_ID": true, "AWS_PARTITION": true, "AWS_URL_SUFFIX": true,
	"AWS_NO_VALUE": true, "AWS_NOTIFICATION_ARNS": true,
	// Helper functions
	"List": true, "Param": true, "Output": true,
	// Policy types
	"PolicyDocument": true, "PolicyStatement": true, "DenyStatement": true,
	"ServicePrincipal": true, "AWSPrincipal": true, "AllPrincipal": true,
	"FederatedPrincipal": true, "Json": true, "Any": true, "Tag": true,
}

func (r UndefinedReference) Check(file *ast.File, fset *token.FileSet) []Issue {
	// When called without package context, only check against current file definitions
	return r.checkWithDefined(file, fset, nil)
}

// CheckWithContext implements PackageAwareRule for cross-file reference checking.
func (r UndefinedReference) CheckWithContext(file *ast.File, fset *token.FileSet, ctx *PackageContext) []Issue {
	return r.checkWithDefined(file, fset, ctx)
}

func (r UndefinedReference) checkWithDefined(file *ast.File, fset *token.FileSet, ctx *PackageContext) []Issue {
	var issues []Issue

	// Collect all defined identifiers in this file
	defined := make(map[string]bool)
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.VAR {
			for _, spec := range genDecl.Specs {
				if valueSpec, ok := spec.(*ast.ValueSpec); ok {
					for _, name := range valueSpec.Names {
						defined[name.Name] = true
					}
				}
			}
		}
	}

	// Check for undefined references in field values
	ast.Inspect(file, func(n ast.Node) bool {
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}

		// Check if value is a bare identifier
		ident, ok := kv.Value.(*ast.Ident)
		if !ok {
			return true
		}

		name := ident.Name

		// Skip known identifiers
		if knownIdentifiers[name] {
			return true
		}

		// Skip if defined in this file
		if defined[name] {
			return true
		}

		// Skip if defined in another file in the same package (cross-file reference)
		if ctx != nil && ctx.AllDefinedVars[name] {
			return true
		}

		// Flag PascalCase identifiers that look like resource/param references
		if len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' {
			pos := fset.Position(ident.Pos())
			issues = append(issues, Issue{
				Rule:       r.ID(),
				Message:    fmt.Sprintf("Potentially undefined reference: %s (check if resource/parameter is defined)", name),
				Suggestion: "// Ensure " + name + " is defined or imported",
				File:       pos.Filename,
				Line:       pos.Line,
				Column:     pos.Column,
				Severity:   SeverityWarning,
			})
		}

		return true
	})

	return issues
}

// UnusedIntrinsicsImport detects when the intrinsics package is imported
// but no intrinsic types are used in the file.
type UnusedIntrinsicsImport struct{}

func (r UnusedIntrinsicsImport) ID() string { return "WAW014" }
func (r UnusedIntrinsicsImport) Description() string {
	return "Intrinsics package imported but not used"
}

// intrinsicTypeNames are types from the intrinsics package
var intrinsicTypeNames = map[string]bool{
	"Sub": true, "SubWithMap": true, "Ref": true, "GetAtt": true,
	"Join": true, "Select": true, "If": true, "Equals": true,
	"And": true, "Or": true, "Not": true, "Base64": true,
	"Split": true, "FindInMap": true, "Cidr": true, "GetAZs": true,
	"ImportValue": true, "Condition": true, "Transform": true,
	"List": true, "Param": true, "Output": true,
	"PolicyDocument": true, "PolicyStatement": true, "DenyStatement": true,
	"ServicePrincipal": true, "AWSPrincipal": true, "AllPrincipal": true,
	"FederatedPrincipal": true, "Json": true, "Any": true, "Tag": true,
	// Pseudo-parameters (these are actually constants, not types)
	"AWS_REGION": true, "AWS_ACCOUNT_ID": true, "AWS_STACK_NAME": true,
	"AWS_STACK_ID": true, "AWS_PARTITION": true, "AWS_URL_SUFFIX": true,
	"AWS_NO_VALUE": true, "AWS_NOTIFICATION_ARNS": true,
}

func (r UnusedIntrinsicsImport) Check(file *ast.File, fset *token.FileSet) []Issue {
	var issues []Issue

	// Check if intrinsics is imported as dot import
	var intrinsicsImport *ast.ImportSpec
	for _, imp := range file.Imports {
		if imp.Path != nil && strings.Contains(imp.Path.Value, "intrinsics") {
			if imp.Name != nil && imp.Name.Name == "." {
				intrinsicsImport = imp
				break
			}
		}
	}

	if intrinsicsImport == nil {
		return issues // No dot import of intrinsics
	}

	// Check if any intrinsic types are used
	intrinsicsUsed := false
	ast.Inspect(file, func(n ast.Node) bool {
		if intrinsicsUsed {
			return false // Already found usage, stop searching
		}

		switch node := n.(type) {
		case *ast.Ident:
			if intrinsicTypeNames[node.Name] {
				intrinsicsUsed = true
				return false
			}
		case *ast.CompositeLit:
			// Check for struct literal type like Sub{...}
			if ident, ok := node.Type.(*ast.Ident); ok {
				if intrinsicTypeNames[ident.Name] {
					intrinsicsUsed = true
					return false
				}
			}
		}
		return true
	})

	if !intrinsicsUsed {
		pos := fset.Position(intrinsicsImport.Pos())
		issues = append(issues, Issue{
			Rule:       r.ID(),
			Message:    "Intrinsics package imported but no intrinsic types used",
			Suggestion: "// Remove unused import or use intrinsic types",
			File:       pos.Filename,
			Line:       pos.Line,
			Column:     pos.Column,
			Severity:   SeverityError,
		})
	}

	return issues
}
