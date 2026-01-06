package serverless

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wetwire "github.com/lex00/wetwire-aws-go"
)

// TestResourceTypes verifies all 9 SAM resource types return correct CloudFormation types.
func TestResourceTypes(t *testing.T) {
	tests := []struct {
		name     string
		resource wetwire.Resource
		expected string
	}{
		{"Function", Function{}, "AWS::Serverless::Function"},
		{"Api", Api{}, "AWS::Serverless::Api"},
		{"HttpApi", HttpApi{}, "AWS::Serverless::HttpApi"},
		{"SimpleTable", SimpleTable{}, "AWS::Serverless::SimpleTable"},
		{"LayerVersion", LayerVersion{}, "AWS::Serverless::LayerVersion"},
		{"StateMachine", StateMachine{}, "AWS::Serverless::StateMachine"},
		{"Application", Application{}, "AWS::Serverless::Application"},
		{"Connector", Connector{}, "AWS::Serverless::Connector"},
		{"GraphQLApi", GraphQLApi{}, "AWS::Serverless::GraphQLApi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.resource.ResourceType())
		})
	}
}

// TestFunctionSerialization tests that Function serializes to valid JSON.
func TestFunctionSerialization(t *testing.T) {
	fn := Function{
		Handler:    "bootstrap",
		Runtime:    "provided.al2",
		MemorySize: 128,
		Timeout:    30,
		CodeUri:    "./hello-world/",
		Environment: &Function_Environment{
			Variables: map[string]any{
				"TABLE_NAME": "my-table",
			},
		},
	}

	data, err := json.Marshal(fn)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))

	assert.Equal(t, "bootstrap", parsed["Handler"])
	assert.Equal(t, "provided.al2", parsed["Runtime"])
	assert.Equal(t, float64(128), parsed["MemorySize"])
	assert.Equal(t, float64(30), parsed["Timeout"])

	env := parsed["Environment"].(map[string]any)
	vars := env["Variables"].(map[string]any)
	assert.Equal(t, "my-table", vars["TABLE_NAME"])
}

// TestApiSerialization tests that Api serializes to valid JSON.
func TestApiSerialization(t *testing.T) {
	api := Api{
		StageName: "prod",
		Cors: &Api_CorsConfiguration{
			AllowOrigin: "'*'",
		},
	}

	data, err := json.Marshal(api)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))

	assert.Equal(t, "prod", parsed["StageName"])
	cors := parsed["Cors"].(map[string]any)
	assert.Equal(t, "'*'", cors["AllowOrigin"])
}

// TestHttpApiSerialization tests that HttpApi serializes to valid JSON.
func TestHttpApiSerialization(t *testing.T) {
	httpApi := HttpApi{
		StageName: "v1",
	}

	data, err := json.Marshal(httpApi)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))

	assert.Equal(t, "v1", parsed["StageName"])
}

// TestSimpleTableSerialization tests that SimpleTable serializes to valid JSON.
func TestSimpleTableSerialization(t *testing.T) {
	table := SimpleTable{
		TableName: "my-table",
		PrimaryKey: &SimpleTable_PrimaryKey{
			Name:  "id",
			Type_: "String",
		},
	}

	data, err := json.Marshal(table)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))

	assert.Equal(t, "my-table", parsed["TableName"])
	pk := parsed["PrimaryKey"].(map[string]any)
	assert.Equal(t, "id", pk["Name"])
	assert.Equal(t, "String", pk["Type"])
}

// TestLayerVersionSerialization tests that LayerVersion serializes to valid JSON.
func TestLayerVersionSerialization(t *testing.T) {
	layer := LayerVersion{
		LayerName:          "my-layer",
		ContentUri:         "./layer/",
		CompatibleRuntimes: []any{"provided.al2", "python3.9"},
	}

	data, err := json.Marshal(layer)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))

	assert.Equal(t, "my-layer", parsed["LayerName"])
	assert.Equal(t, "./layer/", parsed["ContentUri"])
}

// TestStateMachineSerialization tests that StateMachine serializes to valid JSON.
func TestStateMachineSerialization(t *testing.T) {
	sm := StateMachine{
		Name:  "my-state-machine",
		Type_: "STANDARD",
	}

	data, err := json.Marshal(sm)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))

	assert.Equal(t, "my-state-machine", parsed["Name"])
	assert.Equal(t, "STANDARD", parsed["Type"])
}

// TestPropertyTypes tests that nested property types work correctly.
func TestPropertyTypes(t *testing.T) {
	t.Run("Function_Environment", func(t *testing.T) {
		env := Function_Environment{
			Variables: map[string]any{
				"KEY1": "value1",
				"KEY2": "value2",
			},
		}

		data, err := json.Marshal(env)
		require.NoError(t, err)

		var parsed map[string]any
		require.NoError(t, json.Unmarshal(data, &parsed))

		vars := parsed["Variables"].(map[string]any)
		assert.Equal(t, "value1", vars["KEY1"])
		assert.Equal(t, "value2", vars["KEY2"])
	})

	t.Run("Function_VpcConfig", func(t *testing.T) {
		vpc := Function_VpcConfig{
			SecurityGroupIds: []any{"sg-12345", "sg-67890"},
			SubnetIds:        []any{"subnet-abc", "subnet-def"},
		}

		data, err := json.Marshal(vpc)
		require.NoError(t, err)

		var parsed map[string]any
		require.NoError(t, json.Unmarshal(data, &parsed))

		secGroups := parsed["SecurityGroupIds"].([]any)
		assert.Len(t, secGroups, 2)
		assert.Equal(t, "sg-12345", secGroups[0])
	})

	t.Run("Api_CorsConfiguration", func(t *testing.T) {
		cors := Api_CorsConfiguration{
			AllowOrigin:  "'*'",
			AllowMethods: "'GET,POST'",
			AllowHeaders: "'Content-Type'",
		}

		data, err := json.Marshal(cors)
		require.NoError(t, err)

		var parsed map[string]any
		require.NoError(t, json.Unmarshal(data, &parsed))

		assert.Equal(t, "'*'", parsed["AllowOrigin"])
		assert.Equal(t, "'GET,POST'", parsed["AllowMethods"])
	})

	t.Run("SimpleTable_PrimaryKey", func(t *testing.T) {
		pk := SimpleTable_PrimaryKey{
			Name:  "userId",
			Type_: "String",
		}

		data, err := json.Marshal(pk)
		require.NoError(t, err)

		var parsed map[string]any
		require.NoError(t, json.Unmarshal(data, &parsed))

		assert.Equal(t, "userId", parsed["Name"])
		assert.Equal(t, "String", parsed["Type"])
	})
}

// TestResourceImplementsInterface verifies all resources implement wetwire.Resource.
func TestResourceImplementsInterface(t *testing.T) {
	// This test ensures compile-time interface satisfaction
	var _ wetwire.Resource = Function{}
	var _ wetwire.Resource = Api{}
	var _ wetwire.Resource = HttpApi{}
	var _ wetwire.Resource = SimpleTable{}
	var _ wetwire.Resource = LayerVersion{}
	var _ wetwire.Resource = StateMachine{}
	var _ wetwire.Resource = Application{}
	var _ wetwire.Resource = Connector{}
	var _ wetwire.Resource = GraphQLApi{}
}

// TestFunctionWithAttrRef tests that Function can reference other resources.
func TestFunctionWithAttrRef(t *testing.T) {
	fn := Function{
		Handler: "bootstrap",
		Runtime: "provided.al2",
		Role: wetwire.AttrRef{
			Resource:  "MyRole",
			Attribute: "Arn",
		},
	}

	data, err := json.Marshal(fn)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))

	// Role should be serialized as Fn::GetAtt
	role := parsed["Role"].(map[string]any)
	getAtt := role["Fn::GetAtt"].([]any)
	assert.Equal(t, "MyRole", getAtt[0])
	assert.Equal(t, "Arn", getAtt[1])
}

// TestOmitEmptyFields tests that empty fields are omitted from JSON.
func TestOmitEmptyFields(t *testing.T) {
	fn := Function{
		Handler: "bootstrap",
		Runtime: "provided.al2",
		// All other fields are zero values
	}

	data, err := json.Marshal(fn)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))

	// Only Handler and Runtime should be present
	assert.Equal(t, "bootstrap", parsed["Handler"])
	assert.Equal(t, "provided.al2", parsed["Runtime"])

	// These fields should not be present
	_, hasMemorySize := parsed["MemorySize"]
	assert.False(t, hasMemorySize, "MemorySize should be omitted when zero")

	_, hasTimeout := parsed["Timeout"]
	assert.False(t, hasTimeout, "Timeout should be omitted when zero")

	_, hasEnvironment := parsed["Environment"]
	assert.False(t, hasEnvironment, "Environment should be omitted when nil")
}
