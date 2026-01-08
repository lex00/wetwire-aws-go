package kiro

// Persona represents a simulated user persona for automated testing.
type Persona struct {
	Name        string
	Description string
	// Responses are predefined answers to common clarifying questions.
	// These are sent to kiro-cli when the agent asks questions.
	Responses []string
}

// Personas contains all available test personas.
var Personas = map[string]*Persona{
	"beginner": {
		Name:        "beginner",
		Description: "New to AWS, asks many clarifying questions",
		Responses: []string{
			"I'm not sure, what do you recommend?",
			"Yes, that sounds good",
			"I don't know what that means, can you explain?",
			"Sure, go ahead",
			"What's the default?",
			"Yes please",
			"OK",
		},
	},
	"intermediate": {
		Name:        "intermediate",
		Description: "Familiar with AWS basics, asks targeted questions",
		Responses: []string{
			"Yes, use the defaults",
			"That works for me",
			"Go ahead with your recommendation",
			"Yes",
			"Sounds good",
			"Please proceed",
		},
	},
	"expert": {
		Name:        "expert",
		Description: "Deep AWS knowledge, asks advanced questions",
		Responses: []string{
			"Use KMS encryption with a customer managed key",
			"Enable cross-region replication",
			"Add lifecycle policies for cost optimization",
			"Yes, with proper IAM boundaries",
			"Include CloudWatch alarms",
			"Yes",
		},
	},
	"terse": {
		Name:        "terse",
		Description: "Gives minimal responses",
		Responses: []string{
			"yes",
			"ok",
			"fine",
			"yes",
			"do it",
			"ok",
		},
	},
	"verbose": {
		Name:        "verbose",
		Description: "Provides detailed context",
		Responses: []string{
			"Yes, I'd like to proceed with that approach. We're building a data lake for our analytics team and need to ensure data durability.",
			"That sounds like a good solution. Our use case involves storing large JSON files that are processed daily by our ETL pipeline.",
			"I agree with your recommendation. We need to comply with our company's security requirements which mandate encryption at rest.",
			"Please proceed. This is for a production workload so reliability is important.",
			"Yes, that configuration looks correct for our needs.",
			"Go ahead and implement it.",
		},
	},
}

// GetPersona returns a persona by name.
func GetPersona(name string) (*Persona, bool) {
	p, ok := Personas[name]
	return p, ok
}

// AllPersonaNames returns all available persona names.
func AllPersonaNames() []string {
	return []string{"beginner", "intermediate", "expert", "terse", "verbose"}
}
