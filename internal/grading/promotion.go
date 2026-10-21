package grading

import (
	"context"

	"cbt-api/internal/models"
)

// BuiltinPromotionThreshold is the overall percentage a student needs
// to be recommended eligible when no PromotionPolicy is configured -
// the same 50% pass mark used everywhere else in this package.
const BuiltinPromotionThreshold = 50.0

// BuiltinCoreSubjectPassMark is the per-subject pass mark applied to a
// school's configured core subjects when no PromotionPolicy specifies
// its own.
const BuiltinCoreSubjectPassMark = 40.0

// ResolvedPromotionPolicy is what Service.ResolvePromotionPolicy
// returns.
type ResolvedPromotionPolicy struct {
	Source              string // "builtin_default" | "school" | "class_level"
	PolicyID            *string
	OverallThreshold    float64
	CoreSubjectIDs      []string
	CoreSubjectPassMark float64
}

// ResolvePromotionPolicy looks up, in order: a PromotionPolicy scoped
// to the given class level, then one scoped to the whole school, then
// falls back to the built-in default (no core-subject requirement,
// BuiltinPromotionThreshold overall). Mirrors ResolveScale's own
// lookup order and nil-safety for the identical reason: a school/
// Service that never configured anything must see identical,
// deterministic behavior.
func (s *Service) ResolvePromotionPolicy(ctx context.Context, schoolID, classLevelID string) ResolvedPromotionPolicy {
	fallback := ResolvedPromotionPolicy{
		Source: "builtin_default", OverallThreshold: BuiltinPromotionThreshold,
		CoreSubjectPassMark: BuiltinCoreSubjectPassMark,
	}
	if s == nil || s.db == nil || schoolID == "" {
		return fallback
	}

	tryScope := func(source string, classLevel *string) (ResolvedPromotionPolicy, bool) {
		var policy models.PromotionPolicy
		q := s.db.WithContext(ctx).Where("school_id = ? AND is_active = ?", schoolID, true)
		if classLevel == nil {
			q = q.Where("class_level_id IS NULL")
		} else {
			q = q.Where("class_level_id = ?", *classLevel)
		}
		if err := q.First(&policy).Error; err != nil {
			return ResolvedPromotionPolicy{}, false
		}
		id := policy.ID
		return ResolvedPromotionPolicy{
			Source: source, PolicyID: &id,
			OverallThreshold: policy.OverallThreshold, CoreSubjectIDs: []string(policy.CoreSubjectIDs),
			CoreSubjectPassMark: policy.CoreSubjectPassMark,
		}, true
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

// SubjectStanding is the minimal per-subject input EvaluatePromotion
// needs - a caller (which already has each subject's full computed
// result) assembles this rather than this package querying for it.
type SubjectStanding struct {
	SubjectID  string
	Percentage float64
	IsComplete bool
}

// PromotionEvaluation is what EvaluatePromotion returns - a
// RECOMMENDATION only. Nothing in this struct, and no caller of this
// function, may treat Recommendation as an authorized decision: that
// is a models.PromotionDecision row a human deliberately creates
// through its own, separate write path.
type PromotionEvaluation struct {
	// Recommendation is one of "ELIGIBLE FOR REVIEW", "NOT ELIGIBLE -
	// BELOW THRESHOLD", "NOT ELIGIBLE - CORE SUBJECT REQUIREMENT NOT
	// MET", or "INCOMPLETE RESULTS - CANNOT EVALUATE".
	Recommendation string
	Reasons        []string
	ResultsComplete bool
}

// EvaluatePromotion applies a resolved policy to one student's
// subject standings - pure computation, no I/O, so it is trivially
// testable and has no way to reach outside the data it was given.
func EvaluatePromotion(policy ResolvedPromotionPolicy, overallPercentage float64, subjects []SubjectStanding) PromotionEvaluation {
	allComplete := true
	for _, s := range subjects {
		if !s.IsComplete {
			allComplete = false
			break
		}
	}
	if !allComplete || len(subjects) == 0 {
		return PromotionEvaluation{
			Recommendation:  "INCOMPLETE RESULTS - CANNOT EVALUATE",
			Reasons:         []string{"One or more subjects do not yet have a complete result."},
			ResultsComplete: false,
		}
	}

	var reasons []string
	eligible := true

	if overallPercentage < policy.OverallThreshold {
		eligible = false
		reasons = append(reasons, "Overall average is below the configured threshold.")
	}

	if len(policy.CoreSubjectIDs) > 0 {
		standingByID := make(map[string]SubjectStanding, len(subjects))
		for _, s := range subjects {
			standingByID[s.SubjectID] = s
		}
		for _, coreID := range policy.CoreSubjectIDs {
			standing, found := standingByID[coreID]
			if !found || standing.Percentage < policy.CoreSubjectPassMark {
				eligible = false
				reasons = append(reasons, "A required core subject did not meet the configured pass mark.")
				break
			}
		}
	}

	if eligible {
		return PromotionEvaluation{Recommendation: "ELIGIBLE FOR REVIEW", ResultsComplete: true}
	}
	return PromotionEvaluation{Recommendation: "NOT ELIGIBLE FOR REVIEW", Reasons: reasons, ResultsComplete: true}
}
