// Package discover provides AST-based discovery of CloudFormation resource declarations.
//
// It parses Go source files looking for package-level variable declarations
// of the form:
//
//	var MyBucket = s3.Bucket{...}
//
// and extracts resource metadata including dependencies on other resources.
package discover

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	wetwire "github.com/lex00/wetwire-aws-go"
)

// knownResourcePackages maps package names to CloudFormation service prefixes.
// This must include all 263+ AWS services supported by wetwire-aws.
var knownResourcePackages = map[string]string{
	"accessanalyzer":            "AWS::AccessAnalyzer",
	"acmpca":                    "AWS::ACMPCA",
	"aiops":                     "AWS::AIOps",
	"amazonmq":                  "AWS::AmazonMQ",
	"amplify":                   "AWS::Amplify",
	"amplifyuibuilder":          "AWS::AmplifyUIBuilder",
	"apigateway":                "AWS::ApiGateway",
	"apigatewayv2":              "AWS::ApiGatewayV2",
	"appconfig":                 "AWS::AppConfig",
	"appflow":                   "AWS::AppFlow",
	"appintegrations":           "AWS::AppIntegrations",
	"applicationautoscaling":    "AWS::ApplicationAutoScaling",
	"applicationinsights":       "AWS::ApplicationInsights",
	"applicationsignals":        "AWS::ApplicationSignals",
	"appmesh":                   "AWS::AppMesh",
	"apprunner":                 "AWS::AppRunner",
	"appstream":                 "AWS::AppStream",
	"appsync":                   "AWS::AppSync",
	"apptest":                   "AWS::AppTest",
	"aps":                       "AWS::APS",
	"arcregionswitch":           "AWS::ARCRegionSwitch",
	"arczonalshift":             "AWS::ARCZonalShift",
	"ask":                       "AWS::ASK",
	"athena":                    "AWS::Athena",
	"auditmanager":              "AWS::AuditManager",
	"autoscaling":               "AWS::AutoScaling",
	"autoscalingplans":          "AWS::AutoScalingPlans",
	"b2bi":                      "AWS::B2BI",
	"backup":                    "AWS::Backup",
	"backupgateway":             "AWS::BackupGateway",
	"batch":                     "AWS::Batch",
	"bcmdataexports":            "AWS::BCMDataExports",
	"bedrock":                   "AWS::Bedrock",
	"bedrockagentcore":          "AWS::BedrockAgentCore",
	"billing":                   "AWS::Billing",
	"billingconductor":          "AWS::BillingConductor",
	"budgets":                   "AWS::Budgets",
	"cases":                     "AWS::Cases",
	"cassandra":                 "AWS::Cassandra",
	"ce":                        "AWS::CE",
	"certificatemanager":        "AWS::CertificateManager",
	"chatbot":                   "AWS::Chatbot",
	"cleanrooms":                "AWS::CleanRooms",
	"cleanroomsml":              "AWS::CleanRoomsML",
	"cloud9":                    "AWS::Cloud9",
	"cloudformation":            "AWS::CloudFormation",
	"cloudfront":                "AWS::CloudFront",
	"cloudtrail":                "AWS::CloudTrail",
	"cloudwatch":                "AWS::CloudWatch",
	"codeartifact":              "AWS::CodeArtifact",
	"codebuild":                 "AWS::CodeBuild",
	"codecommit":                "AWS::CodeCommit",
	"codeconnections":           "AWS::CodeConnections",
	"codedeploy":                "AWS::CodeDeploy",
	"codeguruprofiler":          "AWS::CodeGuruProfiler",
	"codegurureviewer":          "AWS::CodeGuruReviewer",
	"codepipeline":              "AWS::CodePipeline",
	"codestar":                  "AWS::CodeStar",
	"codestarconnections":       "AWS::CodeStarConnections",
	"codestarnotifications":     "AWS::CodeStarNotifications",
	"cognito":                   "AWS::Cognito",
	"comprehend":                "AWS::Comprehend",
	"config":                    "AWS::Config",
	"connect":                   "AWS::Connect",
	"connectcampaigns":          "AWS::ConnectCampaigns",
	"connectcampaignsv2":        "AWS::ConnectCampaignsV2",
	"controltower":              "AWS::ControlTower",
	"cur":                       "AWS::CUR",
	"customerprofiles":          "AWS::CustomerProfiles",
	"databrew":                  "AWS::DataBrew",
	"datapipeline":              "AWS::DataPipeline",
	"datasync":                  "AWS::DataSync",
	"datazone":                  "AWS::DataZone",
	"dax":                       "AWS::DAX",
	"deadline":                  "AWS::Deadline",
	"detective":                 "AWS::Detective",
	"devopsagent":               "AWS::DevOpsAgent",
	"devopsguru":                "AWS::DevOpsGuru",
	"directoryservice":          "AWS::DirectoryService",
	"dlm":                       "AWS::DLM",
	"dms":                       "AWS::DMS",
	"docdb":                     "AWS::DocDB",
	"docdbelastic":              "AWS::DocDBElastic",
	"dsql":                      "AWS::DSQL",
	"dynamodb":                  "AWS::DynamoDB",
	"ec2":                       "AWS::EC2",
	"ecr":                       "AWS::ECR",
	"ecs":                       "AWS::ECS",
	"efs":                       "AWS::EFS",
	"eks":                       "AWS::EKS",
	"elasticache":               "AWS::ElastiCache",
	"elasticbeanstalk":          "AWS::ElasticBeanstalk",
	"elasticloadbalancing":      "AWS::ElasticLoadBalancing",
	"elasticloadbalancingv2":    "AWS::ElasticLoadBalancingV2",
	"elasticsearch":             "AWS::Elasticsearch",
	"emr":                       "AWS::EMR",
	"emrcontainers":             "AWS::EMRContainers",
	"emrserverless":             "AWS::EMRServerless",
	"entityresolution":          "AWS::EntityResolution",
	"events":                    "AWS::Events",
	"eventschemas":              "AWS::EventSchemas",
	"evidently":                 "AWS::Evidently",
	"evs":                       "AWS::EVS",
	"finspace":                  "AWS::FinSpace",
	"fis":                       "AWS::FIS",
	"fms":                       "AWS::FMS",
	"forecast":                  "AWS::Forecast",
	"frauddetector":             "AWS::FraudDetector",
	"fsx":                       "AWS::FSx",
	"gamelift":                  "AWS::GameLift",
	"globalaccelerator":         "AWS::GlobalAccelerator",
	"glue":                      "AWS::Glue",
	"grafana":                   "AWS::Grafana",
	"greengrass":                "AWS::Greengrass",
	"greengrassv2":              "AWS::GreengrassV2",
	"groundstation":             "AWS::GroundStation",
	"guardduty":                 "AWS::GuardDuty",
	"healthimaging":             "AWS::HealthImaging",
	"healthlake":                "AWS::HealthLake",
	"iam":                       "AWS::IAM",
	"identitystore":             "AWS::IdentityStore",
	"imagebuilder":              "AWS::ImageBuilder",
	"inspector":                 "AWS::Inspector",
	"inspectorv2":               "AWS::InspectorV2",
	"internetmonitor":           "AWS::InternetMonitor",
	"invoicing":                 "AWS::Invoicing",
	"iot":                       "AWS::IoT",
	"iotanalytics":              "AWS::IoTAnalytics",
	"iotcoredeviceadvisor":      "AWS::IoTCoreDeviceAdvisor",
	"iotevents":                 "AWS::IoTEvents",
	"iotfleetwise":              "AWS::IoTFleetWise",
	"iotsitewise":               "AWS::IoTSiteWise",
	"iotthingsgraph":            "AWS::IoTThingsGraph",
	"iottwinmaker":              "AWS::IoTTwinMaker",
	"iotwireless":               "AWS::IoTWireless",
	"ivs":                       "AWS::IVS",
	"ivschat":                   "AWS::IVSChat",
	"kafkaconnect":              "AWS::KafkaConnect",
	"kendra":                    "AWS::Kendra",
	"kendraranking":             "AWS::KendraRanking",
	"kinesis":                   "AWS::Kinesis",
	"kinesisanalytics":          "AWS::KinesisAnalytics",
	"kinesisanalyticsv2":        "AWS::KinesisAnalyticsV2",
	"kinesisfirehose":           "AWS::KinesisFirehose",
	"kinesisvideo":              "AWS::KinesisVideo",
	"kms":                       "AWS::KMS",
	"lakeformation":             "AWS::LakeFormation",
	"lambda":                    "AWS::Lambda",
	"launchwizard":              "AWS::LaunchWizard",
	"lex":                       "AWS::Lex",
	"licensemanager":            "AWS::LicenseManager",
	"lightsail":                 "AWS::Lightsail",
	"location":                  "AWS::Location",
	"logs":                      "AWS::Logs",
	"lookoutequipment":          "AWS::LookoutEquipment",
	"lookoutvision":             "AWS::LookoutVision",
	"m2":                        "AWS::M2",
	"macie":                     "AWS::Macie",
	"managedblockchain":         "AWS::ManagedBlockchain",
	"mediaconnect":              "AWS::MediaConnect",
	"mediaconvert":              "AWS::MediaConvert",
	"medialive":                 "AWS::MediaLive",
	"mediapackage":              "AWS::MediaPackage",
	"mediapackagev2":            "AWS::MediaPackageV2",
	"mediastore":                "AWS::MediaStore",
	"mediatailor":               "AWS::MediaTailor",
	"memorydb":                  "AWS::MemoryDB",
	"mpa":                       "AWS::MPA",
	"msk":                       "AWS::MSK",
	"mwaa":                      "AWS::MWAA",
	"neptune":                   "AWS::Neptune",
	"neptunegraph":              "AWS::NeptuneGraph",
	"networkfirewall":           "AWS::NetworkFirewall",
	"networkmanager":            "AWS::NetworkManager",
	"notifications":             "AWS::Notifications",
	"notificationscontacts":     "AWS::NotificationsContacts",
	"oam":                       "AWS::Oam",
	"observabilityadmin":        "AWS::ObservabilityAdmin",
	"odb":                       "AWS::ODB",
	"omics":                     "AWS::Omics",
	"opensearchserverless":      "AWS::OpenSearchServerless",
	"opensearchservice":         "AWS::OpenSearchService",
	"opsworks":                  "AWS::OpsWorks",
	"organizations":             "AWS::Organizations",
	"osis":                      "AWS::OSIS",
	"panorama":                  "AWS::Panorama",
	"paymentcryptography":       "AWS::PaymentCryptography",
	"pcaconnectorad":            "AWS::PCAConnectorAD",
	"pcaconnectorscep":          "AWS::PCAConnectorSCEP",
	"pcs":                       "AWS::PCS",
	"personalize":               "AWS::Personalize",
	"pinpoint":                  "AWS::Pinpoint",
	"pinpointemail":             "AWS::PinpointEmail",
	"pipes":                     "AWS::Pipes",
	"proton":                    "AWS::Proton",
	"qbusiness":                 "AWS::QBusiness",
	"qldb":                      "AWS::QLDB",
	"quicksight":                "AWS::QuickSight",
	"ram":                       "AWS::RAM",
	"rbin":                      "AWS::Rbin",
	"rds":                       "AWS::RDS",
	"redshift":                  "AWS::Redshift",
	"redshiftserverless":        "AWS::RedshiftServerless",
	"refactorspaces":            "AWS::RefactorSpaces",
	"rekognition":               "AWS::Rekognition",
	"resiliencehub":             "AWS::ResilienceHub",
	"resourceexplorer2":         "AWS::ResourceExplorer2",
	"resourcegroups":            "AWS::ResourceGroups",
	"robomaker":                 "AWS::RoboMaker",
	"rolesanywhere":             "AWS::RolesAnywhere",
	"route53":                   "AWS::Route53",
	"route53profiles":           "AWS::Route53Profiles",
	"route53recoverycontrol":    "AWS::Route53RecoveryControl",
	"route53recoveryreadiness":  "AWS::Route53RecoveryReadiness",
	"route53resolver":           "AWS::Route53Resolver",
	"rtbfabric":                 "AWS::RTBFabric",
	"rum":                       "AWS::RUM",
	"s3":                        "AWS::S3",
	"s3express":                 "AWS::S3Express",
	"s3objectlambda":            "AWS::S3ObjectLambda",
	"s3outposts":                "AWS::S3Outposts",
	"s3tables":                  "AWS::S3Tables",
	"s3vectors":                 "AWS::S3Vectors",
	"sagemaker":                 "AWS::SageMaker",
	"scheduler":                 "AWS::Scheduler",
	"sdb":                       "AWS::SDB",
	"secretsmanager":            "AWS::SecretsManager",
	"securityhub":               "AWS::SecurityHub",
	"securitylake":              "AWS::SecurityLake",
	"serverless":                "AWS::Serverless",
	"servicecatalog":            "AWS::ServiceCatalog",
	"servicecatalogappregistry": "AWS::ServiceCatalogAppRegistry",
	"servicediscovery":          "AWS::ServiceDiscovery",
	"ses":                       "AWS::SES",
	"shield":                    "AWS::Shield",
	"signer":                    "AWS::Signer",
	"simspaceweaver":            "AWS::SimSpaceWeaver",
	"smsvoice":                  "AWS::SMSVoice",
	"sns":                       "AWS::SNS",
	"sqs":                       "AWS::SQS",
	"ssm":                       "AWS::SSM",
	"ssmcontacts":               "AWS::SSMContacts",
	"ssmguiconnect":             "AWS::SSMGuiConnect",
	"ssmincidents":              "AWS::SSMIncidents",
	"ssmquicksetup":             "AWS::SSMQuickSetup",
	"sso":                       "AWS::SSO",
	"stepfunctions":             "AWS::StepFunctions",
	"supportapp":                "AWS::SupportApp",
	"synthetics":                "AWS::Synthetics",
	"systemsmanagersap":         "AWS::SystemsManagerSAP",
	"timestream":                "AWS::Timestream",
	"transfer":                  "AWS::Transfer",
	"verifiedpermissions":       "AWS::VerifiedPermissions",
	"voiceid":                   "AWS::VoiceID",
	"vpclattice":                "AWS::VpcLattice",
	"waf":                       "AWS::WAF",
	"wafregional":               "AWS::WAFRegional",
	"wafv2":                     "AWS::WAFv2",
	"wisdom":                    "AWS::Wisdom",
	"workspaces":                "AWS::WorkSpaces",
	"workspacesinstances":       "AWS::WorkSpacesInstances",
	"workspacesthinclient":      "AWS::WorkSpacesThinClient",
	"workspacesweb":             "AWS::WorkSpacesWeb",
	"xray":                      "AWS::XRay",
}

// Options configures the discovery process.
type Options struct {
	// Packages to scan (e.g., "./infra/...")
	Packages []string
	// Verbose enables debug output
	Verbose bool
}

// Result contains all discovered resources and any errors.
type Result struct {
	// Resources maps logical name to discovered resource
	Resources map[string]wetwire.DiscoveredResource
	// Parameters maps logical name to discovered parameter
	Parameters map[string]wetwire.DiscoveredParameter
	// Outputs maps logical name to discovered output
	Outputs map[string]wetwire.DiscoveredOutput
	// Mappings maps logical name to discovered mapping
	Mappings map[string]wetwire.DiscoveredMapping
	// Conditions maps logical name to discovered condition
	Conditions map[string]wetwire.DiscoveredCondition
	// AllVars tracks all package-level var declarations (including non-resources)
	// Used to avoid false positives when checking dependencies
	AllVars map[string]bool
	// VarAttrRefs tracks AttrRefUsages for all variables (including property types)
	// Key is variable name, value includes AttrRefs and referenced var names with field paths
	VarAttrRefs map[string]VarAttrRefInfo
	// Errors encountered during parsing
	Errors []error
}

// VarAttrRefInfo tracks AttrRef usages and variable references for a single variable
type VarAttrRefInfo struct {
	AttrRefs []wetwire.AttrRefUsage
	// VarRefs maps field path to referenced variable name
	VarRefs map[string]string
}

// Discover scans Go packages for CloudFormation resource declarations.
func Discover(opts Options) (*Result, error) {
	result := &Result{
		Resources:   make(map[string]wetwire.DiscoveredResource),
		Parameters:  make(map[string]wetwire.DiscoveredParameter),
		Outputs:     make(map[string]wetwire.DiscoveredOutput),
		Mappings:    make(map[string]wetwire.DiscoveredMapping),
		Conditions:  make(map[string]wetwire.DiscoveredCondition),
		AllVars:     make(map[string]bool),
		VarAttrRefs: make(map[string]VarAttrRefInfo),
	}

	for _, pkg := range opts.Packages {
		if err := discoverPackage(pkg, result, opts); err != nil {
			return nil, fmt.Errorf("discovering %s: %w", pkg, err)
		}
	}

	// Validate dependencies - only flag truly undefined references
	// Skip vars that are defined locally (including property type blocks)
	for name, res := range result.Resources {
		for _, dep := range res.Dependencies {
			// Skip if it's a known resource
			if _, ok := result.Resources[dep]; ok {
				continue
			}
			// Skip if it's a local var declaration (e.g., Tag blocks, property types)
			if result.AllVars[dep] {
				continue
			}
			result.Errors = append(result.Errors, fmt.Errorf(
				"%s:%d: %s references undefined resource %q",
				res.File, res.Line, name, dep,
			))
		}
	}

	return result, nil
}

func discoverPackage(pattern string, result *Result, opts Options) error {
	// Handle ./... pattern
	pattern = strings.TrimSuffix(pattern, "/...")
	recursive := strings.HasSuffix(pattern, "...")
	if recursive {
		pattern = strings.TrimSuffix(pattern, "...")
	}

	// Get absolute path
	absPath, err := filepath.Abs(pattern)
	if err != nil {
		return err
	}

	if recursive {
		return filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return discoverDir(path, result, opts)
			}
			return nil
		})
	}

	return discoverDir(absPath, result, opts)
}

func discoverDir(dir string, result *Result, opts Options) error {
	fset := token.NewFileSet()

	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		// Directory might not contain Go files
		if os.IsNotExist(err) || strings.Contains(err.Error(), "no Go files") {
			return nil
		}
		return err
	}

	for _, pkg := range pkgs {
		for filename, file := range pkg.Files {
			discoverFile(fset, filename, file, result, opts)
		}
	}

	return nil
}

func discoverFile(fset *token.FileSet, filename string, file *ast.File, result *Result, opts Options) {
	// Build import map: alias -> package path
	imports := make(map[string]string)
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		var name string
		if imp.Name != nil {
			name = imp.Name.Name
		} else {
			// Use last component of path
			parts := strings.Split(path, "/")
			name = parts[len(parts)-1]
		}
		imports[name] = path
	}

	// Find package-level var declarations
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			continue
		}

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok || len(valueSpec.Names) != 1 || len(valueSpec.Values) != 1 {
				continue
			}

			name := valueSpec.Names[0].Name
			value := valueSpec.Values[0]

			// Skip blank identifier
			if name == "_" {
				continue
			}

			// Track ALL var declarations to avoid false positive undefined references
			result.AllVars[name] = true

			// Check if it's a composite literal (Type{...})
			compLit, ok := value.(*ast.CompositeLit)
			if !ok {
				continue
			}

			// Get the type
			typeName, pkgName := extractTypeName(compLit.Type)
			if typeName == "" {
				continue
			}

			pos := fset.Position(valueSpec.Pos())

			// Check for intrinsic types (Parameter, Output, Mapping, Condition types)
			if isIntrinsicPackage(pkgName, imports) || pkgName == "" {
				switch typeName {
				case "Parameter":
					result.Parameters[name] = wetwire.DiscoveredParameter{
						Name: name,
						File: filename,
						Line: pos.Line,
					}
					continue
				case "Output":
					// Extract AttrRef usages from output fields
					_, attrRefs := extractDependencies(compLit, imports)
					result.Outputs[name] = wetwire.DiscoveredOutput{
						Name:          name,
						File:          filename,
						Line:          pos.Line,
						AttrRefUsages: attrRefs,
					}
					continue
				case "Mapping":
					result.Mappings[name] = wetwire.DiscoveredMapping{
						Name: name,
						File: filename,
						Line: pos.Line,
					}
					continue
				case "Equals", "And", "Or", "Not":
					result.Conditions[name] = wetwire.DiscoveredCondition{
						Name: name,
						Type: typeName,
						File: filename,
						Line: pos.Line,
					}
					continue
				}
			}

			// Extract dependencies, AttrRef usages, and var refs from field values
			// Do this for ALL composite literals, not just resource packages
			deps, attrRefs, varRefs := extractDependenciesWithVarRefs(compLit, imports)

			// Track AttrRefs for all variables (including property types and intrinsics)
			result.VarAttrRefs[name] = VarAttrRefInfo{
				AttrRefs: attrRefs,
				VarRefs:  varRefs,
			}

			// Check if this is a known resource package
			if _, known := knownResourcePackages[pkgName]; !known {
				continue
			}

			// Skip property types (e.g., Bucket_ServerSideEncryptionRule)
			// These contain "_" and are nested types, not CloudFormation resources
			if strings.Contains(typeName, "_") {
				continue
			}

			result.Resources[name] = wetwire.DiscoveredResource{
				Name:          name,
				Type:          fmt.Sprintf("%s.%s", pkgName, typeName),
				Package:       file.Name.Name,
				File:          filename,
				Line:          pos.Line,
				Dependencies:  deps,
				AttrRefUsages: attrRefs,
			}
		}
	}
}

// isIntrinsicPackage checks if the package is the intrinsics package.
func isIntrinsicPackage(pkgName string, imports map[string]string) bool {
	if pkgName == "" {
		// Dot-imported, check if intrinsics is dot-imported
		for alias, path := range imports {
			if alias == "." && strings.HasSuffix(path, "/intrinsics") {
				return true
			}
		}
		return false
	}
	if pkgName == "intrinsics" {
		return true
	}
	// Check if the package alias points to the intrinsics package
	if path, ok := imports[pkgName]; ok {
		return strings.HasSuffix(path, "/intrinsics")
	}
	return false
}

// extractTypeName extracts the type name and package from a type expression.
// For s3.Bucket, returns ("Bucket", "s3").
func extractTypeName(expr ast.Expr) (typeName, pkgName string) {
	switch t := expr.(type) {
	case *ast.SelectorExpr:
		// pkg.Type
		if ident, ok := t.X.(*ast.Ident); ok {
			return t.Sel.Name, ident.Name
		}
	case *ast.Ident:
		// Just Type (unqualified)
		return t.Name, ""
	}
	return "", ""
}

// extractDependencies finds references to other resources in composite literal fields.
// It looks for patterns like:
//   - OtherResource (identifier reference)
//   - OtherResource.Arn (selector for AttrRef)
//
// Returns both dependencies and AttrRef usages for GetAtt resolution.
func extractDependencies(lit *ast.CompositeLit, imports map[string]string) ([]string, []wetwire.AttrRefUsage) {
	deps, attrRefs, _ := extractDependenciesWithVarRefs(lit, imports)
	return deps, attrRefs
}

// extractDependenciesWithVarRefs is like extractDependencies but also returns variable references
// with their field paths, for recursive AttrRef resolution.
func extractDependenciesWithVarRefs(lit *ast.CompositeLit, imports map[string]string) ([]string, []wetwire.AttrRefUsage, map[string]string) {
	var deps []string
	var attrRefs []wetwire.AttrRefUsage
	varRefs := make(map[string]string) // field path -> var name
	seen := make(map[string]bool)

	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}

		// Get the field name for the path
		fieldName := ""
		if ident, ok := kv.Key.(*ast.Ident); ok {
			fieldName = ident.Name
		}

		// Recursively find dependencies in the value
		findDepsWithVarRefs(kv.Value, &deps, &attrRefs, varRefs, seen, imports, fieldName)
	}

	return deps, attrRefs, varRefs
}

func findDepsWithVarRefs(expr ast.Expr, deps *[]string, attrRefs *[]wetwire.AttrRefUsage, varRefs map[string]string, seen map[string]bool, imports map[string]string, fieldPath string) {
	switch v := expr.(type) {
	case *ast.Ident:
		// Could be a reference to another resource or variable
		// Skip if it's a known package or common identifier
		name := v.Name
		if _, isImport := imports[name]; isImport {
			return
		}
		if isCommonIdent(name) {
			return
		}
		// Heuristic: starts with uppercase = likely a resource/var reference
		if len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' {
			if !seen[name] {
				*deps = append(*deps, name)
				seen[name] = true
			}
			// Track this variable reference with its field path
			if varRefs != nil && fieldPath != "" {
				varRefs[fieldPath] = name
			}
		}

	case *ast.SelectorExpr:
		// Could be Resource.Attr or pkg.Type
		if ident, ok := v.X.(*ast.Ident); ok {
			name := ident.Name
			// Skip package selectors
			if _, isImport := imports[name]; isImport {
				return
			}
			// This is likely Resource.Attribute (e.g., LambdaRole.Arn)
			if len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' {
				if !seen[name] {
					*deps = append(*deps, name)
					seen[name] = true
				}
				// Record the AttrRef usage for GetAtt resolution
				*attrRefs = append(*attrRefs, wetwire.AttrRefUsage{
					ResourceName: name,
					Attribute:    v.Sel.Name,
					FieldPath:    fieldPath,
				})
			}
		}

	case *ast.CompositeLit:
		// Nested struct, check its elements
		for _, elt := range v.Elts {
			if kv, ok := elt.(*ast.KeyValueExpr); ok {
				// Build nested field path
				nestedPath := fieldPath
				if ident, ok := kv.Key.(*ast.Ident); ok {
					if fieldPath != "" {
						nestedPath = fieldPath + "." + ident.Name
					} else {
						nestedPath = ident.Name
					}
				}
				findDepsWithVarRefs(kv.Value, deps, attrRefs, varRefs, seen, imports, nestedPath)
			} else {
				findDepsWithVarRefs(elt, deps, attrRefs, varRefs, seen, imports, fieldPath)
			}
		}

	case *ast.UnaryExpr:
		// Handle &Type{...}
		findDepsWithVarRefs(v.X, deps, attrRefs, varRefs, seen, imports, fieldPath)

	case *ast.CallExpr:
		// Handle function calls - check arguments
		for _, arg := range v.Args {
			findDepsWithVarRefs(arg, deps, attrRefs, varRefs, seen, imports, fieldPath)
		}

	case *ast.SliceExpr:
		findDepsWithVarRefs(v.X, deps, attrRefs, varRefs, seen, imports, fieldPath)

	case *ast.IndexExpr:
		findDepsWithVarRefs(v.X, deps, attrRefs, varRefs, seen, imports, fieldPath)
		findDepsWithVarRefs(v.Index, deps, attrRefs, varRefs, seen, imports, fieldPath)
	}
}

// ResolveAttrRefs recursively collects all AttrRefUsages for a variable by following
// its variable references and prefixing paths appropriately.
func (r *Result) ResolveAttrRefs(varName string) []wetwire.AttrRefUsage {
	visited := make(map[string]bool)
	return r.resolveAttrRefsRecursive(varName, "", visited)
}

func (r *Result) resolveAttrRefsRecursive(varName, pathPrefix string, visited map[string]bool) []wetwire.AttrRefUsage {
	if visited[varName] {
		return nil
	}
	visited[varName] = true

	info, ok := r.VarAttrRefs[varName]
	if !ok {
		return nil
	}

	var result []wetwire.AttrRefUsage

	// Add direct AttrRefs with path prefix
	for _, ref := range info.AttrRefs {
		fullPath := ref.FieldPath
		if pathPrefix != "" {
			fullPath = pathPrefix + "." + ref.FieldPath
		}
		result = append(result, wetwire.AttrRefUsage{
			ResourceName: ref.ResourceName,
			Attribute:    ref.Attribute,
			FieldPath:    fullPath,
		})
	}

	// Recursively resolve variable references
	for fieldPath, refVarName := range info.VarRefs {
		fullPath := fieldPath
		if pathPrefix != "" {
			fullPath = pathPrefix + "." + fieldPath
		}
		nested := r.resolveAttrRefsRecursive(refVarName, fullPath, visited)
		result = append(result, nested...)
	}

	return result
}

// isCommonIdent returns true for identifiers that are likely not resource names.
func isCommonIdent(name string) bool {
	common := map[string]bool{
		// Go built-ins
		"true": true, "false": true, "nil": true,
		"string": true, "int": true, "bool": true, "float64": true,
		"any": true, "error": true,

		// Intrinsic function types (from intrinsics package)
		"Ref": true, "Sub": true, "Join": true, "GetAtt": true,
		"Select": true, "Split": true, "If": true, "Equals": true,
		"And": true, "Or": true, "Not": true, "Condition": true,
		"FindInMap": true, "Base64": true, "Cidr": true, "GetAZs": true,
		"ImportValue": true, "Transform": true, "Json": true,
		"Parameter": true, "Output": true, "Mapping": true,

		// Pseudo-parameter constants (from intrinsics package)
		"AWS_ACCOUNT_ID": true, "AWS_NOTIFICATION_ARNS": true,
		"AWS_NO_VALUE": true, "AWS_PARTITION": true,
		"AWS_REGION": true, "AWS_STACK_ID": true,
		"AWS_STACK_NAME": true, "AWS_URL_SUFFIX": true,
	}
	return common[name]
}
