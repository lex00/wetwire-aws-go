// Package scoring implements the 5-dimension scoring system per spec 6.4.
package scoring

import (
	wetwire "github.com/lex00/wetwire-aws-go"
)

// Dimension constants for max scores.
const (
	MaxDimensionScore = 3
	MaxTotalScore     = 15
)

// Grade thresholds per spec 6.4.
const (
	ExcellentThreshold = 13
	SuccessThreshold   = 10
	PartialThreshold   = 6
)

// Grade names.
const (
	GradeExcellent = "Excellent"
	GradeSuccess   = "Success"
	GradePartial   = "Partial"
	GradeFailure   = "Failure"
)

// Metrics contains the raw metrics used to calculate scores.
type Metrics struct {
	// ResourcesRequired is the number of resources expected.
	ResourcesRequired int
	// ResourcesGenerated is the number of resources actually generated.
	ResourcesGenerated int
	// LintCycles is the number of lint/fix cycles needed.
	LintCycles int
	// LintPassed indicates if linting ultimately passed.
	LintPassed bool
	// BuildPassed indicates if the build passed.
	BuildPassed bool
	// ValidatePassed indicates if validation passed.
	ValidatePassed bool
	// QuestionsAsked is the number of clarifying questions asked.
	QuestionsAsked int
	// ExpectedQuestions is the expected number of questions (varies by persona).
	ExpectedQuestions int
	// FilesCreated is the number of files created.
	FilesCreated int
	// HasProperStructure indicates if code follows idiomatic patterns.
	HasProperStructure bool
}

// Calculate computes a TestScore from the given metrics.
func Calculate(m Metrics) wetwire.TestScore {
	score := wetwire.TestScore{}

	// Completeness (0-3): Were all required resources generated?
	score.Completeness = calculateCompleteness(m)

	// Lint Quality (0-3): How many lint cycles needed? (fewer = better)
	score.LintQuality = calculateLintQuality(m)

	// Code Quality (0-3): Does code follow idiomatic patterns?
	score.CodeQuality = calculateCodeQuality(m)

	// Output Validity (0-3): Does generated output validate?
	score.OutputValidity = calculateOutputValidity(m)

	// Question Efficiency (0-3): Appropriate number of clarifying questions?
	score.QuestionEfficiency = calculateQuestionEfficiency(m)

	// Calculate total
	score.Total = score.Completeness + score.LintQuality + score.CodeQuality +
		score.OutputValidity + score.QuestionEfficiency

	// Determine grade
	score.Grade = DetermineGrade(score.Total)

	return score
}

// DetermineGrade returns the grade based on total score.
func DetermineGrade(total int) string {
	switch {
	case total >= ExcellentThreshold:
		return GradeExcellent
	case total >= SuccessThreshold:
		return GradeSuccess
	case total >= PartialThreshold:
		return GradePartial
	default:
		return GradeFailure
	}
}

// calculateCompleteness scores resource generation completeness.
// 0: No resources generated or major missing
// 1: Less than 50% of required resources
// 2: 50-99% of required resources
// 3: 100% of required resources
func calculateCompleteness(m Metrics) int {
	if m.ResourcesRequired == 0 {
		// If no requirements specified, use files created as proxy
		if m.FilesCreated >= 3 {
			return 3
		}
		if m.FilesCreated >= 1 {
			return 2
		}
		return 0
	}

	if m.ResourcesGenerated == 0 {
		return 0
	}

	ratio := float64(m.ResourcesGenerated) / float64(m.ResourcesRequired)
	switch {
	case ratio >= 1.0:
		return 3
	case ratio >= 0.5:
		return 2
	default:
		return 1
	}
}

// calculateLintQuality scores based on lint cycles.
// 0: Lint never passed or > 5 cycles
// 1: 4-5 lint cycles
// 2: 2-3 lint cycles
// 3: 0-1 lint cycles (clean first try or minor fix)
func calculateLintQuality(m Metrics) int {
	if !m.LintPassed {
		return 0
	}

	switch {
	case m.LintCycles <= 1:
		return 3
	case m.LintCycles <= 3:
		return 2
	case m.LintCycles <= 5:
		return 1
	default:
		return 0
	}
}

// calculateCodeQuality scores code structure and patterns.
// 0: Major issues (doesn't compile, wrong structure)
// 1: Basic structure but issues
// 2: Good structure with minor issues
// 3: Excellent idiomatic code
func calculateCodeQuality(m Metrics) int {
	if !m.BuildPassed {
		return 0
	}

	if m.HasProperStructure {
		if m.LintPassed && m.LintCycles <= 1 {
			return 3
		}
		return 2
	}

	if m.LintPassed {
		return 1
	}

	return 0
}

// calculateOutputValidity scores output validation.
// 0: Output doesn't validate
// 1: Build passed but validation failed
// 2: Validation passed with warnings
// 3: Full validation passed (build + lint + validate)
func calculateOutputValidity(m Metrics) int {
	if !m.BuildPassed {
		return 0
	}

	if m.ValidatePassed {
		if m.LintPassed {
			return 3
		}
		return 2
	}

	// Build passed but validate failed
	return 1
}

// calculateQuestionEfficiency scores clarifying questions appropriateness.
// 0: Too many questions (> expected + 3) or necessary questions not asked
// 1: More questions than expected
// 2: Close to expected (within 1)
// 3: Exactly right number or no questions needed
func calculateQuestionEfficiency(m Metrics) int {
	if m.ExpectedQuestions == 0 {
		// For terse personas or simple tasks, fewer questions is better
		switch {
		case m.QuestionsAsked == 0:
			return 3
		case m.QuestionsAsked <= 2:
			return 2
		case m.QuestionsAsked <= 4:
			return 1
		default:
			return 0
		}
	}

	diff := abs(m.QuestionsAsked - m.ExpectedQuestions)
	switch {
	case diff == 0:
		return 3
	case diff <= 1:
		return 2
	case diff <= 3:
		return 1
	default:
		return 0
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
