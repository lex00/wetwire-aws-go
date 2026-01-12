// Package rds_aurora demonstrates idiomatic wetwire patterns for RDS Aurora.
//
// This example shows:
// - Flat block-style declarations (named top-level vars)
// - Everything wrapped (direct references, no Ref()/GetAtt() calls)
// - Extracted inline configs (no nested struct literals)
// - Clear comments
//
// Architecture:
//
//	Aurora PostgreSQL Cluster
//	|
//	+-- Writer Instance (primary)
//	|
//	+-- Reader Instance (replica)
//	|
//	+-- DB Subnet Group (spans multiple AZs)
//	|
//	+-- Cluster Parameter Group (tuning settings)
package rds_aurora

import (
	. "github.com/lex00/wetwire-aws-go/intrinsics"
	"github.com/lex00/wetwire-aws-go/resources/rds"
)

// ----------------------------------------------------------------------------
// DB Subnet Group
// ----------------------------------------------------------------------------

// DBSubnetGroup defines which subnets the Aurora cluster can use.
// In production, you would reference actual subnet resources.
var DBSubnetGroup = rds.DBSubnetGroup{
	DBSubnetGroupName:        Sub{String: "${AWS::StackName}-db-subnet-group"},
	DBSubnetGroupDescription: "Subnet group for Aurora cluster",
	SubnetIds: []any{
		Param("PrivateSubnetA"), // In production, reference actual subnet resources
		Param("PrivateSubnetB"),
	},
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-db-subnet-group"}},
	},
}

// ----------------------------------------------------------------------------
// DB Cluster Parameter Group
// ----------------------------------------------------------------------------

// AuroraClusterParameterGroup defines cluster-level parameters.
var AuroraClusterParameterGroup = rds.DBClusterParameterGroup{
	DBClusterParameterGroupName: Sub{String: "${AWS::StackName}-aurora-params"},
	Description:                 "Custom parameter group for Aurora PostgreSQL",
	Family:                      "aurora-postgresql15",
	Parameters: Json{
		// Enable logical replication for CDC
		"rds.logical_replication": "1",
		// Longer timeout for migrations
		"statement_timeout": "3600000",
	},
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-aurora-params"}},
	},
}

// ----------------------------------------------------------------------------
// Aurora Cluster
// ----------------------------------------------------------------------------

// AuroraCluster is the Aurora PostgreSQL cluster.
// Note: Direct references to DBSubnetGroup, AuroraClusterParameterGroup, DBSecurityGroup - no Ref() needed!
var AuroraCluster = rds.DBCluster{
	DBClusterIdentifier:         Sub{String: "${AWS::StackName}-aurora-cluster"},
	Engine:                      "aurora-postgresql",
	EngineVersion:               "15.4",
	DatabaseName:                "appdb",
	MasterUsername:              "admin",
	ManageMasterUserPassword:    true, // Use Secrets Manager for password
	DBSubnetGroupName:           DBSubnetGroup,
	DBClusterParameterGroupName: AuroraClusterParameterGroup,
	VpcSecurityGroupIds:         []any{DBSecurityGroup},
	BackupRetentionPeriod:       7,
	PreferredBackupWindow:       "03:00-04:00",
	PreferredMaintenanceWindow:  "sun:04:00-sun:05:00",
	StorageEncrypted:            true,
	DeletionProtection:          true,
	EnableCloudwatchLogsExports: []any{"postgresql"},
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-aurora-cluster"}},
	},
}

// ----------------------------------------------------------------------------
// Aurora Instances
// ----------------------------------------------------------------------------

// AuroraWriterInstance is the primary (writer) instance.
// Note: Direct references to AuroraCluster and DBSubnetGroup - no Ref() needed!
var AuroraWriterInstance = rds.DBInstance{
	DBInstanceIdentifier:    Sub{String: "${AWS::StackName}-aurora-writer"},
	DBClusterIdentifier:     AuroraCluster,
	DBInstanceClass:         "db.r6g.large",
	Engine:                  "aurora-postgresql",
	DBSubnetGroupName:       DBSubnetGroup,
	PubliclyAccessible:      false,
	AutoMinorVersionUpgrade: true,
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-aurora-writer"}},
		Tag{Key: "Role", Value: "writer"},
	},
}

// AuroraReaderInstance is the replica (reader) instance.
// Note: Direct references to AuroraCluster and DBSubnetGroup - no Ref() needed!
var AuroraReaderInstance = rds.DBInstance{
	DBInstanceIdentifier:    Sub{String: "${AWS::StackName}-aurora-reader"},
	DBClusterIdentifier:     AuroraCluster,
	DBInstanceClass:         "db.r6g.large",
	Engine:                  "aurora-postgresql",
	DBSubnetGroupName:       DBSubnetGroup,
	PubliclyAccessible:      false,
	AutoMinorVersionUpgrade: true,
	PromotionTier:           1, // Failover priority
	Tags: []any{
		Tag{Key: "Name", Value: Sub{String: "${AWS::StackName}-aurora-reader"}},
		Tag{Key: "Role", Value: "reader"},
	},
}
