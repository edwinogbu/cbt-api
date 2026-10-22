package grading

import (
	"context"

	"cbt-api/internal/models"
)

// ScoredExam is one graded exam contributing to a subject's assessment -
// the minimal shape ComputeSubjectAssessment needs. Assembled by the
// caller (which already queries Exam+Result together) rather than this
// package querying for them itself, keeping this package decoupled from
// repository-layer joins.
type ScoredExam struct {
	ExamType    string
	TotalMarks  int
	TotalScore  int
	// FullyGraded false means this exam still has essay/descriptive
	// answers awaiting manual marking (models.Result.FullyGraded) - its
	// score is provisional, not final. A subject whose total includes
	// any provisional exam is itself provisional (ComponentResult/
	// SubjectAssessmentResult.IsProvisional), never silently presented
	// as a finished result.
	FullyGraded bool
}

// ComponentSpec is one configured assessment component, resolved from
// either a school's AssessmentScheme or the built-in default.
type ComponentSpec struct {
	Code        string
	Name        string
	Weight      float64
	ExamTypes   []string
	Aggregation string // pooled | average | best | latest
}

// ResolvedScheme is what Service.ResolveScheme returns.
type ResolvedScheme struct {
	Source        string // DefaultAssessmentMode | "school" | "class_level" | "subject"
	SchemeID      *string
	MissingPolicy string
	Components    []ComponentSpec
}

// ResolveScheme looks up, in order: an AssessmentScheme scoped to this
// exact class-level+subject, then class-level-wide (any subject), then
// school-wide, then falls back to DefaultAssessmentMode (today's
// existing unweighted-pooled behavior - identical output for every
// school until one explicitly configures a scheme).
func (s *Service) ResolveScheme(ctx context.Context, schoolID, classLevelID, subjectID string) ResolvedScheme {
	fallback := ResolvedScheme{Source: DefaultAssessmentMode, MissingPolicy: "renormalize"}
	if s == nil || s.db == nil || schoolID == "" {
		return fallback
	}

	tryScope := func(source string, classLevel, subject *string) (ResolvedScheme, bool) {
		var scheme models.AssessmentScheme
		q := s.db.WithContext(ctx).Where("school_id = ? AND is_active = ?", schoolID, true)
		if classLevel == nil {
			q = q.Where("class_level_id IS NULL")
		} else {
			q = q.Where("class_level_id = ?", *classLevel)
		}
		if subject == nil {
			q = q.Where("subject_id IS NULL")
		} else {
			q = q.Where("subject_id = ?", *subject)
		}
		if err := q.Preload("Components").First(&scheme).Error; err != nil {
			return ResolvedScheme{}, false
		}
		if len(scheme.Components) == 0 {
			return ResolvedScheme{}, false
		}
		specs := make([]ComponentSpec, len(scheme.Components))
		for i, c := range scheme.Components {
			specs[i] = ComponentSpec{
				Code: c.Code, Name: c.Name, Weight: c.Weight,
				ExamTypes: []string(c.ExamTypes), Aggregation: c.Aggregation,
			}
		}
		id := scheme.ID
		return ResolvedScheme{Source: source, SchemeID: &id, MissingPolicy: scheme.MissingPolicy, Components: specs}, true
	}

	if classLevelID != "" && subjectID != "" {
		if r, ok := tryScope("subject", &classLevelID, &subjectID); ok {
			return r
		}
	}
	if classLevelID != "" {
		if r, ok := tryScope("class_level", &classLevelID, nil); ok {
			return r
		}
	}
	if r, ok := tryScope("school", nil, nil); ok {
		return r
	}
	return fallback
}

// ComponentResult is one assessment component's resolved outcome for a
// subject - what a report card's subject row needs to render a CA/Test/
// Exam breakdown.
type ComponentResult struct {
	Code       string
	Name       string
	Weight     float64
	Percentage float64
	// HasResults is false when no exam of this component's configured
	// ExamTypes exists yet for this subject/student/term - distinguishes
	// "not assessed yet" from "assessed and scored 0%". A report card
	// must show this component as missing/pending, never as a 0.
	HasResults bool
}

// SubjectAssessmentResult is the resolved, weighted outcome for one
// subject.
type SubjectAssessmentResult struct {
	Source       string
	SchemeID     *string
	Components   []ComponentResult
	TotalPercent float64
	// IsComplete is false if any CONFIGURED component (under a real
	// AssessmentScheme) has no results yet, or if there are no exams at
	// all under the default unweighted mode. A report card must mark an
	// incomplete subject's total as provisional, never present it as a
	// finished grade.
	IsComplete bool
	// IsProvisional is true if any contributing exam's Result isn't yet
	// FullyGraded (essay/descriptive marking still pending) - distinct
	// from IsComplete, which is about missing components entirely.
	IsProvisional bool
}

func examTypeInList(types []string, examType string) bool {
	for _, t := range types {
		if t == examType {
			return true
		}
	}
	return false
}

func percentOf(e ScoredExam) float64 {
	if e.TotalMarks == 0 {
		return 0
	}
	return float64(e.TotalScore) / float64(e.TotalMarks) * 100
}

// ComputeSubjectAssessment combines a subject's scored exams into one
// weighted total, per the resolved scheme. With scheme.Source ==
// DefaultAssessmentMode, this reproduces today's existing
// buildSubjectTermlyResult behavior exactly: pool every exam's raw
// score/marks regardless of ExamType, Σscore/Σmarks - so a school with
// no configured AssessmentScheme sees identical report-card totals to
// before this function existed.
func ComputeSubjectAssessment(scheme ResolvedScheme, exams []ScoredExam) SubjectAssessmentResult {
	if scheme.Source == DefaultAssessmentMode || scheme.Source == "" {
		if len(exams) == 0 {
			return SubjectAssessmentResult{Source: DefaultAssessmentMode, IsComplete: false}
		}
		var sumScore, sumMarks int
		provisional := false
		for _, e := range exams {
			sumScore += e.TotalScore
			sumMarks += e.TotalMarks
			if !e.FullyGraded {
				provisional = true
			}
		}
		pct := 0.0
		if sumMarks > 0 {
			pct = float64(sumScore) / float64(sumMarks) * 100
		}
		return SubjectAssessmentResult{
			Source: DefaultAssessmentMode, TotalPercent: pct,
			IsComplete: true, IsProvisional: provisional,
		}
	}

	results := make([]ComponentResult, 0, len(scheme.Components))
	allComplete := true
	anyProvisional := false
	var weightedSum, weightPresent float64

	for _, comp := range scheme.Components {
		var matched []ScoredExam
		for _, e := range exams {
			if examTypeInList(comp.ExamTypes, e.ExamType) {
				matched = append(matched, e)
			}
		}

		hasResults := len(matched) > 0
		pct := 0.0
		if hasResults {
			switch comp.Aggregation {
			case "best":
				for _, e := range matched {
					if p := percentOf(e); p > pct {
						pct = p
					}
				}
			case "average":
				var sum float64
				for _, e := range matched {
					sum += percentOf(e)
				}
				pct = sum / float64(len(matched))
			case "latest":
				// Caller is expected to pass exams for a component in
				// chronological order; the last match is the latest.
				pct = percentOf(matched[len(matched)-1])
			default: // "pooled"
				var sumScore, sumMarks int
				for _, e := range matched {
					sumScore += e.TotalScore
					sumMarks += e.TotalMarks
				}
				if sumMarks > 0 {
					pct = float64(sumScore) / float64(sumMarks) * 100
				}
			}
			for _, e := range matched {
				if !e.FullyGraded {
					anyProvisional = true
				}
			}
			weightedSum += pct * comp.Weight / 100
			weightPresent += comp.Weight
		} else {
			allComplete = false
		}

		results = append(results, ComponentResult{
			Code: comp.Code, Name: comp.Name, Weight: comp.Weight,
			Percentage: pct, HasResults: hasResults,
		})
	}

	total := weightedSum
	if scheme.MissingPolicy == "renormalize" && weightPresent > 0 && weightPresent < 100 {
		total = weightedSum / weightPresent * 100
	}
	// MissingPolicy "zero": weightedSum already excludes missing
	// components' contribution entirely (never added in the loop
	// above), which is exactly "treat the missing component as
	// contributing 0 toward the total weight", so no further scaling
	// is needed - total stays as weightedSum.

	return SubjectAssessmentResult{
		Source: scheme.Source, SchemeID: scheme.SchemeID,
		Components: results, TotalPercent: total,
		IsComplete: allComplete, IsProvisional: anyProvisional,
	}
}
