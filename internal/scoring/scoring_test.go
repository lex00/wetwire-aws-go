package scoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetermineGrade(t *testing.T) {
	tests := []struct {
		total    int
		expected string
	}{
		{15, GradeExcellent},
		{14, GradeExcellent},
		{13, GradeExcellent},
		{12, GradeSuccess},
		{11, GradeSuccess},
		{10, GradeSuccess},
		{9, GradePartial},
		{8, GradePartial},
		{7, GradePartial},
		{6, GradePartial},
		{5, GradeFailure},
		{4, GradeFailure},
		{3, GradeFailure},
		{0, GradeFailure},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := DetermineGrade(tt.total)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestCalculateCompleteness(t *testing.T) {
	tests := []struct {
		name     string
		metrics  Metrics
		expected int
	}{
		{
			name:     "all resources generated",
			metrics:  Metrics{ResourcesRequired: 3, ResourcesGenerated: 3},
			expected: 3,
		},
		{
			name:     "more than required",
			metrics:  Metrics{ResourcesRequired: 3, ResourcesGenerated: 5},
			expected: 3,
		},
		{
			name:     "50-99% generated",
			metrics:  Metrics{ResourcesRequired: 4, ResourcesGenerated: 3},
			expected: 2,
		},
		{
			name:     "less than 50%",
			metrics:  Metrics{ResourcesRequired: 10, ResourcesGenerated: 3},
			expected: 1,
		},
		{
			name:     "none generated",
			metrics:  Metrics{ResourcesRequired: 3, ResourcesGenerated: 0},
			expected: 0,
		},
		{
			name:     "no requirements - many files",
			metrics:  Metrics{ResourcesRequired: 0, FilesCreated: 5},
			expected: 3,
		},
		{
			name:     "no requirements - some files",
			metrics:  Metrics{ResourcesRequired: 0, FilesCreated: 1},
			expected: 2,
		},
		{
			name:     "no requirements - no files",
			metrics:  Metrics{ResourcesRequired: 0, FilesCreated: 0},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateCompleteness(tt.metrics)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestCalculateLintQuality(t *testing.T) {
	tests := []struct {
		name     string
		metrics  Metrics
		expected int
	}{
		{
			name:     "clean first try",
			metrics:  Metrics{LintPassed: true, LintCycles: 0},
			expected: 3,
		},
		{
			name:     "one fix cycle",
			metrics:  Metrics{LintPassed: true, LintCycles: 1},
			expected: 3,
		},
		{
			name:     "two fix cycles",
			metrics:  Metrics{LintPassed: true, LintCycles: 2},
			expected: 2,
		},
		{
			name:     "three fix cycles",
			metrics:  Metrics{LintPassed: true, LintCycles: 3},
			expected: 2,
		},
		{
			name:     "four fix cycles",
			metrics:  Metrics{LintPassed: true, LintCycles: 4},
			expected: 1,
		},
		{
			name:     "many fix cycles",
			metrics:  Metrics{LintPassed: true, LintCycles: 6},
			expected: 0,
		},
		{
			name:     "lint failed",
			metrics:  Metrics{LintPassed: false, LintCycles: 2},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateLintQuality(tt.metrics)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestCalculateCodeQuality(t *testing.T) {
	tests := []struct {
		name     string
		metrics  Metrics
		expected int
	}{
		{
			name: "excellent",
			metrics: Metrics{
				BuildPassed:        true,
				LintPassed:         true,
				LintCycles:         0,
				HasProperStructure: true,
			},
			expected: 3,
		},
		{
			name: "good structure some fixes",
			metrics: Metrics{
				BuildPassed:        true,
				LintPassed:         true,
				LintCycles:         3,
				HasProperStructure: true,
			},
			expected: 2,
		},
		{
			name: "lint passed but no structure",
			metrics: Metrics{
				BuildPassed:        true,
				LintPassed:         true,
				LintCycles:         2,
				HasProperStructure: false,
			},
			expected: 1,
		},
		{
			name: "build failed",
			metrics: Metrics{
				BuildPassed: false,
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateCodeQuality(tt.metrics)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestCalculateOutputValidity(t *testing.T) {
	tests := []struct {
		name     string
		metrics  Metrics
		expected int
	}{
		{
			name: "full validation",
			metrics: Metrics{
				BuildPassed:    true,
				LintPassed:     true,
				ValidatePassed: true,
			},
			expected: 3,
		},
		{
			name: "validate passed lint failed",
			metrics: Metrics{
				BuildPassed:    true,
				LintPassed:     false,
				ValidatePassed: true,
			},
			expected: 2,
		},
		{
			name: "build passed validate failed",
			metrics: Metrics{
				BuildPassed:    true,
				ValidatePassed: false,
			},
			expected: 1,
		},
		{
			name: "build failed",
			metrics: Metrics{
				BuildPassed: false,
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateOutputValidity(tt.metrics)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestCalculateQuestionEfficiency(t *testing.T) {
	tests := []struct {
		name     string
		metrics  Metrics
		expected int
	}{
		{
			name:     "exactly right",
			metrics:  Metrics{QuestionsAsked: 3, ExpectedQuestions: 3},
			expected: 3,
		},
		{
			name:     "within one",
			metrics:  Metrics{QuestionsAsked: 2, ExpectedQuestions: 3},
			expected: 2,
		},
		{
			name:     "within three",
			metrics:  Metrics{QuestionsAsked: 5, ExpectedQuestions: 2},
			expected: 1,
		},
		{
			name:     "too many questions",
			metrics:  Metrics{QuestionsAsked: 10, ExpectedQuestions: 2},
			expected: 0,
		},
		{
			name:     "no expected - no questions",
			metrics:  Metrics{QuestionsAsked: 0, ExpectedQuestions: 0},
			expected: 3,
		},
		{
			name:     "no expected - few questions",
			metrics:  Metrics{QuestionsAsked: 2, ExpectedQuestions: 0},
			expected: 2,
		},
		{
			name:     "no expected - many questions",
			metrics:  Metrics{QuestionsAsked: 5, ExpectedQuestions: 0},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateQuestionEfficiency(tt.metrics)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestCalculate(t *testing.T) {
	// Test excellent score
	excellent := Metrics{
		ResourcesRequired:  3,
		ResourcesGenerated: 3,
		LintPassed:         true,
		LintCycles:         0,
		BuildPassed:        true,
		ValidatePassed:     true,
		HasProperStructure: true,
		QuestionsAsked:     2,
		ExpectedQuestions:  2,
	}
	score := Calculate(excellent)
	assert.Equal(t, 3, score.Completeness)
	assert.Equal(t, 3, score.LintQuality)
	assert.Equal(t, 3, score.CodeQuality)
	assert.Equal(t, 3, score.OutputValidity)
	assert.Equal(t, 3, score.QuestionEfficiency)
	assert.Equal(t, 15, score.Total)
	assert.Equal(t, GradeExcellent, score.Grade)

	// Test failure score
	failure := Metrics{
		ResourcesRequired:  3,
		ResourcesGenerated: 0,
		LintPassed:         false,
		BuildPassed:        false,
		QuestionsAsked:     10, // Too many questions
		ExpectedQuestions:  2,
	}
	score = Calculate(failure)
	assert.Equal(t, 0, score.Completeness)
	assert.Equal(t, 0, score.LintQuality)
	assert.Equal(t, 0, score.CodeQuality)
	assert.Equal(t, 0, score.OutputValidity)
	assert.True(t, score.Total <= 5) // Should be failure grade
	assert.Equal(t, GradeFailure, score.Grade)

	// Test partial score
	partial := Metrics{
		ResourcesRequired:  4,
		ResourcesGenerated: 3,
		LintPassed:         true,
		LintCycles:         3,
		BuildPassed:        true,
		ValidatePassed:     false,
	}
	score = Calculate(partial)
	assert.True(t, score.Total >= PartialThreshold)
	assert.True(t, score.Total < SuccessThreshold)
}
