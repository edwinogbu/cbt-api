// Package grading is the shared, configurable grading engine report
// cards (Mid-Term, Main Termly, Promotion) are meant to build on
// instead of each hardcoding its own grade-boundary logic - replacing
// three pre-existing, mutually INCONSISTENT hardcoded grade functions
// found in this codebase: dto.GetGrade (A>=70), exam_repo.go's
// getGradeFromPercentage (A>=70, same), and parent_service.go's
// calculateGrade (A>=80, genuinely different - the parent portal and
// the student's own report card disagreed on a 70-79% score before
// this package existed).
//
// Design principle, enforced throughout this package: a school that
// never configures anything sees IDENTICAL output to before this
// package existed. Missing or invalid configuration is never an error -
// it falls back to BuiltinBands/BuiltinPassMark, the exact boundaries
// every hardcoded function above used. ResolveScale's Source field lets
// a caller (a report card, an admin UI) distinguish "using the default -
// nothing configured" from "using this school's own configured scale".
package grading

import (
	"context"
	"fmt"
	"sort"

	"cbt-api/internal/models"

	"gorm.io/gorm"
)

// Band is a resolved grade boundary - independent of whether it came
// from a school's own GradingScale or the built-in default, once
// resolved they're handled identically by FromBands/ComputeGrade.
type Band struct {
	Grade      string
	MinPercent float64
	GradePoint float64
	Remark     string
}

// BuiltinBands are the exact boundaries dto.GetGrade and exam_repo.go's
// getGradeFromPercentage used before this package existed - kept
// byte-for-byte so a school with no configured GradingScale sees
// identical grades to before.
var BuiltinBands = []Band{
	{Grade: "A", MinPercent: 70, GradePoint: 5.0, Remark: "Excellent"},
	{Grade: "B", MinPercent: 60, GradePoint: 4.0, Remark: "Very Good"},
	{Grade: "C", MinPercent: 50, GradePoint: 3.0, Remark: "Good"},
	{Grade: "D", MinPercent: 45, GradePoint: 2.0, Remark: "Fair"},
	{Grade: "E", MinPercent: 40, GradePoint: 1.0, Remark: "Pass"},
	{Grade: "F", MinPercent: 0, GradePoint: 0.0, Remark: "Fail"},
}

// BuiltinPassMark is the >=50 pass/fail boundary hardcoded at
// exam_service.go:3089 before this package existed.
const BuiltinPassMark = 50.0

// DefaultAssessmentMode identifies today's existing, unweighted
// subject-total behavior (buildSubjectTermlyResult sums every result's
// raw score and marks for the subject, regardless of ExamType) - used
// as a Source value once assessment-component weighting (modeled by
// models.AssessmentScheme) is resolved and applied. Not yet wired into
// report generation as of this package's first version - see the
// engagement's own verification report for what remains.
const DefaultAssessmentMode = "unweighted_pooled"

// Builtin grades a percentage using BuiltinBands/BuiltinPassMark only -
// exactly what dto.GetGrade and exam_repo.go's getGradeFromPercentage
// did before this package existed. Exported so those two functions can
// become one-line wrappers around it with provably identical behavior
// (see internal/cbt/dto/exam_dto.go's GetGrade and
// internal/cbt/repository/exam_repo.go's getGradeFromPercentage).
func Builtin(percentage float64) (grade string, gradePoint float64, remark string) {
	band := FromBands(BuiltinBands, percentage)
	return band.Grade, band.GradePoint, band.Remark
}

// FromBands finds the matching band for a percentage: the band with the
// highest MinPercent that is <= percentage. Bands store only a lower
// bound (matching every `if pct >= X` chain this replaces) - the next-
// highest band's MinPercent is this band's implicit upper bound, so
// there is no gap/overlap representable by construction. Validate on
// save (ValidateBands) that every band's MinPercent is unique and that
// some band has MinPercent = 0, so every 0-100 percentage resolves to
// a band; FromBands itself just trusts that and falls back to the
// lowest-MinPercent band (or a zero-value Band, empty Grade, if none
// exist at all) rather than panicking on malformed input.
func FromBands(bands []Band, percentage float64) Band {
	if len(bands) == 0 {
		return Band{}
	}
	sorted := make([]Band, len(bands))
	copy(sorted, bands)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].MinPercent > sorted[j].MinPercent })
	for _, b := range sorted {
		if percentage >= b.MinPercent {
			return b
		}
	}
	return sorted[len(sorted)-1]
}

// ValidateBands checks the invariants FromBands relies on: at least one
// band, no two bands sharing a MinPercent, and some band with
// MinPercent = 0 (so every percentage from 0-100 resolves to a band
// rather than silently falling through to the lowest one for a
// percentage below it, which would misgrade). Called when a
// GradingScale is created/updated - never on read, since a scale that
// somehow failed validation after being saved should still degrade to
// "closest matching band", not break report generation.
func ValidateBands(bands []Band) error {
	if len(bands) == 0 {
		return fmt.Errorf("a grading scale must have at least one band")
	}
	seen := make(map[float64]bool, len(bands))
	hasZero := false
	for _, b := range bands {
		if b.Grade == "" {
			return fmt.Errorf("every band must have a grade label")
		}
		if seen[b.MinPercent] {
			return fmt.Errorf("duplicate band boundary at %.2f%% (grade %s)", b.MinPercent, b.Grade)
		}
		seen[b.MinPercent] = true
		if b.MinPercent == 0 {
			hasZero = true
		}
		if b.MinPercent < 0 || b.MinPercent > 100 {
			return fmt.Errorf("band boundary %.2f%% (grade %s) must be between 0 and 100", b.MinPercent, b.Grade)
		}
	}
	if !hasZero {
		return fmt.Errorf("a grading scale must have a band starting at 0%% so every percentage resolves to a grade")
	}
	return nil
}

// GradeResult is what Service.ComputeGrade returns - everything a
// report card needs to display a grade and explain where its boundaries
// came from.
type GradeResult struct {
	Grade      string
	GradePoint float64
	Remark     string
	Passed     bool
	Source     string // "builtin_default" | "school" | "class_level"
	ScaleID    *string
}

// ResolvedScale is what Service.ResolveScale returns.
type ResolvedScale struct {
	Source   string
	ScaleID  *string
	Bands    []Band
	PassMark float64
}

// Service resolves and applies a school's configured grading rules,
// falling back to BuiltinBands/BuiltinPassMark whenever nothing is
// configured. A nil or zero-value Service (db == nil) always falls back
// to the built-in default - this is deliberate, not a bug: it lets
// ComputeGrade be called safely from any context, including ones that
// never wired up a real *gorm.DB (e.g. older tests), without panicking.
type Service struct {
	db *gorm.DB
}

// NewService builds a Service against a real database connection.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// ResolveScale looks up, in order: a GradingScale scoped to the given
// class level, then one scoped to the whole school, then falls back to
// the built-in default. classLevelID may be empty, in which case only
// the school-wide scope is tried.
func (s *Service) ResolveScale(ctx context.Context, schoolID, classLevelID string) ResolvedScale {
	fallback := ResolvedScale{Source: "builtin_default", Bands: BuiltinBands, PassMark: BuiltinPassMark}
	if s == nil || s.db == nil || schoolID == "" {
		return fallback
	}

	tryScope := func(source string, classLevel *string) (ResolvedScale, bool) {
		var scale models.GradingScale
		q := s.db.WithContext(ctx).Where("school_id = ? AND is_active = ?", schoolID, true)
		if classLevel == nil {
			q = q.Where("class_level_id IS NULL")
		} else {
			q = q.Where("class_level_id = ?", *classLevel)
		}
		if err := q.Preload("Bands").First(&scale).Error; err != nil {
			return ResolvedScale{}, false
		}
		if len(scale.Bands) == 0 {
			return ResolvedScale{}, false
		}
		bands := make([]Band, len(scale.Bands))
		for i, b := range scale.Bands {
			bands[i] = Band{Grade: b.Grade, MinPercent: b.MinPercent, GradePoint: b.GradePoint, Remark: b.Remark}
		}
		id := scale.ID
		return ResolvedScale{Source: source, ScaleID: &id, Bands: bands, PassMark: scale.PassMark}, true
	}

	if classLevelID != "" {
		if r, ok := tryScope("class_level", &classLevelID); ok {
			return r
		}
	}
	if r, ok := tryScope("school", nil); ok {
		return r
	}
	return fallback
}

// ComputeGrade is the single entry point every grade-producing call
// site in this codebase should eventually use. dto.GetGrade and
// exam_repo.go's getGradeFromPercentage have already been converted to
// thin wrappers around Builtin (this package's own built-in-only path,
// not this method) so their signatures and behavior are unchanged for
// every existing caller; callers that have real school/class-level
// context (the report-card generation path) should call this method
// directly instead, to actually honor a school's configured scale.
//
// parent_service.go's calculateGrade (A>=80, genuinely different from
// the other two) is deliberately NOT converted here - unifying it
// changes a parent-visible grade for 70-79% scores, which needs
// explicit product sign-off before changing, not a silent fix bundled
// into this refactor.
func (s *Service) ComputeGrade(ctx context.Context, schoolID, classLevelID string, percentage float64) GradeResult {
	resolved := s.ResolveScale(ctx, schoolID, classLevelID)
	band := FromBands(resolved.Bands, percentage)
	return GradeResult{
		Grade:      band.Grade,
		GradePoint: band.GradePoint,
		Remark:     band.Remark,
		Passed:     percentage >= resolved.PassMark,
		Source:     resolved.Source,
		ScaleID:    resolved.ScaleID,
	}
}
