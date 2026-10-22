package models

import "time"

// PromotionPolicy configures a school's (optionally a single class
// level's) promotion-eligibility rules: an overall percentage
// threshold and, optionally, a set of "core" subjects a student must
// also individually pass. A school with no PromotionPolicy row uses
// the built-in default (grading.BuiltinPromotionThreshold, the same
// 50% pass mark every other unconfigured grading check in this
// codebase already falls back to) - this table existing changes
// nothing until a school explicitly configures its own policy.
//
// ClassLevelID is nullable: nil scopes the policy to the whole school;
// set, it scopes to just that class level (e.g. a stricter policy for
// an exam class than a junior one). Mirrors models.GradingScale's own
// ScopeKey pattern for the same reason: two valid shapes (school-wide,
// school+class-level), each enforced unique via one ordinary Postgres
// unique index (a nullable column alone can't be, since Postgres
// treats every NULL as distinct).
type PromotionPolicy struct {
	ID                  string      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SchoolID            string      `gorm:"type:uuid;not null;uniqueIndex:idx_promotion_policy_scope" json:"school_id"`
	ClassLevelID        *string     `gorm:"type:uuid;index" json:"class_level_id"`
	ScopeKey            string      `gorm:"type:varchar(80);not null;uniqueIndex:idx_promotion_policy_scope" json:"-"`
	OverallThreshold    float64     `gorm:"type:numeric(5,2);not null;default:50" json:"overall_threshold"`
	CoreSubjectIDs      StringArray `gorm:"type:jsonb;not null;default:'[]'" json:"core_subject_ids"`
	CoreSubjectPassMark float64     `gorm:"type:numeric(5,2);not null;default:40" json:"core_subject_pass_mark"`
	// Deliberately no `gorm:"default:true"` tag - see GradingScale's
	// own IsActive field for why (GORM's Create() silently drops an
	// explicit false on a zero-valued bool carrying a `default` tag).
	IsActive  bool      `json:"is_active"`
	UpdatedBy *string   `gorm:"type:uuid" json:"updated_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PromotionDecision records the authorized, FINAL promotion decision
// for one student at the end of one academic session - distinct from,
// and never auto-filled from, any system-computed recommendation
// (internal/grading's EvaluatePromotion only ever returns a
// recommendation; nothing in this codebase may copy that
// recommendation into a PromotionDecision automatically). A student
// with no PromotionDecision row for a session has not yet had a
// decision made - a report card must show "Pending authorized
// approval", never infer one from the recommendation.
type PromotionDecision struct {
	ID                 string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SchoolID           string     `gorm:"type:uuid;not null;index" json:"school_id"`
	StudentID          string     `gorm:"type:uuid;not null;uniqueIndex:idx_promotion_decision_scope" json:"student_id"`
	SessionID          string     `gorm:"type:uuid;not null;uniqueIndex:idx_promotion_decision_scope" json:"session_id"`
	ProposedClassLevelID *string  `gorm:"type:uuid" json:"proposed_class_level_id"`
	// Decision is a short, school-facing label the recorder typed or
	// picked (e.g. "Promoted", "Repeated", "Promoted on Trial") - kept
	// as free text rather than a fixed enum, since schools' own
	// promotion terminology varies and this record is a note of what
	// was decided, not a state machine this codebase acts on.
	Decision  string    `gorm:"type:varchar(100);not null" json:"decision"`
	Notes     string    `gorm:"type:text" json:"notes"`
	DecidedBy string    `gorm:"type:uuid;not null" json:"decided_by"`
	DecidedAt time.Time `json:"decided_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
