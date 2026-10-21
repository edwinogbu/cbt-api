package models

import "time"

// GradingScale is a per-school (optionally per-class-level) configurable
// grading boundary set, replacing the hardcoded, mutually inconsistent
// grade-boundary logic found in three separate places before this
// (dto.GetGrade: A>=70, exam_repo.go's getGradeFromPercentage: A>=70,
// parent_service.go's calculateGrade: A>=80). A school with no
// GradingScale row uses the built-in default (grading.BuiltinBands,
// package internal/grading) - identical to today's behavior, so this
// table existing changes nothing until a school explicitly opts in.
//
// ClassLevelID is nullable: nil scopes the scale to the whole school
// (e.g. one scale for every class); set, it scopes to just that class
// level (e.g. a different WAEC-style scale for SSS than JSS). ScopeKey
// exists only so the two valid shapes - school-wide and
// school+class-level - can each be enforced unique via one ordinary
// Postgres unique index (a nullable column alone can't be, since
// Postgres treats every NULL as distinct).
type GradingScale struct {
	ID           string  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SchoolID     string  `gorm:"type:uuid;not null;uniqueIndex:idx_grading_scale_scope" json:"school_id"`
	ClassLevelID *string `gorm:"type:uuid;index" json:"class_level_id"`
	ScopeKey     string  `gorm:"type:varchar(80);not null;uniqueIndex:idx_grading_scale_scope" json:"-"`
	Name         string  `gorm:"type:varchar(100);not null" json:"name"`
	PassMark     float64 `gorm:"type:numeric(5,2);not null;default:50" json:"pass_mark"`
	// Deliberately no `gorm:"default:true"` tag here: GORM's Create()
	// omits zero-valued fields that carry a `default` tag (it can't
	// tell "intentionally false" from "never set" for a plain bool),
	// so an explicit IsActive: false would have been silently
	// overridden back to true by the database's own default - found
	// live by this package's own test for deactivating a scale.
	// Callers that create a new scale must set IsActive: true
	// themselves.
	IsActive  bool               `json:"is_active"`
	UpdatedBy *string            `gorm:"type:uuid" json:"updated_by"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
	Bands     []GradingScaleBand `gorm:"foreignKey:GradingScaleID;constraint:OnDelete:CASCADE" json:"bands"`
}

// GradingScaleBand is one grade band within a GradingScale - e.g.
// {Grade: "A", MinPercent: 70, Remark: "Excellent"}. Only a lower bound
// is stored (matching every existing hardcoded `if pct >= X` chain this
// replaces) - the highest band's upper bound is implicitly 100, and each
// other band's upper bound is implicitly the next band's MinPercent.
// This representation can't express a gap or overlap between bands by
// construction, so there is nothing to validate there; the grading
// service does still check that exactly one band has MinPercent = 0 (so
// every percentage from 0-100 resolves to some band) and that no two
// bands in the same scale share a MinPercent.
type GradingScaleBand struct {
	ID             string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	GradingScaleID string    `gorm:"type:uuid;not null;index" json:"grading_scale_id"`
	Grade          string    `gorm:"type:varchar(10);not null" json:"grade"`
	MinPercent     float64   `gorm:"type:numeric(5,2);not null" json:"min_percent"`
	GradePoint     float64   `gorm:"type:numeric(4,2);default:0" json:"grade_point"`
	Remark         string    `gorm:"type:varchar(50)" json:"remark"`
	SortOrder      int       `json:"sort_order"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// AssessmentScheme configures how a subject's assessment components
// (continuous assessment, exam, etc.) combine into one total, replacing
// today's single unweighted behavior (buildSubjectTermlyResult sums
// every exam's raw score and marks for the subject, regardless of
// ExamType, with no weighting at all). A school with no AssessmentScheme
// row keeps exactly that behavior (grading.DefaultAssessmentMode,
// "unweighted_pooled") - this table existing changes nothing until a
// school explicitly configures weighted components.
//
// Scope narrows from SubjectID+ClassLevelID (most specific) down to
// ClassLevelID alone, then school-wide (both nil) - see ScopeKey/
// grading.ResolveScheme for the exact lookup order.
type AssessmentScheme struct {
	ID            string                `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SchoolID      string                `gorm:"type:uuid;not null;uniqueIndex:idx_assessment_scheme_scope" json:"school_id"`
	ClassLevelID  *string               `gorm:"type:uuid;index" json:"class_level_id"`
	SubjectID     *string               `gorm:"type:uuid;index" json:"subject_id"`
	ScopeKey      string                `gorm:"type:varchar(120);not null;uniqueIndex:idx_assessment_scheme_scope" json:"-"`
	MissingPolicy string                `gorm:"type:varchar(20);not null;default:'renormalize'" json:"missing_policy"` // "renormalize" | "zero"
	IsActive      bool                  `json:"is_active"`
	UpdatedBy     *string               `gorm:"type:uuid" json:"updated_by"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
	Components    []AssessmentComponent `gorm:"foreignKey:SchemeID;constraint:OnDelete:CASCADE" json:"components"`
}

// AssessmentComponent is one weighted piece of an AssessmentScheme, e.g.
// {Code: "ca", Weight: 30, ExamTypes: ["continuous_assessment"]} and
// {Code: "exam", Weight: 70, ExamTypes: ["main_exam"]}. ExamTypes maps
// this component to the EXISTING models.ExamType values already on every
// Exam/Result - no new per-exam field, no existing data touched. A
// scheme's components' Weight values must sum to 100 and no ExamType
// value may appear in more than one component of the same scheme
// (enforced by the grading service on save, not by the database).
type AssessmentComponent struct {
	ID          string      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SchemeID    string      `gorm:"type:uuid;not null;index" json:"scheme_id"`
	Code        string      `gorm:"type:varchar(50);not null" json:"code"`
	Name        string      `gorm:"type:varchar(100);not null" json:"name"`
	Weight      float64     `gorm:"type:numeric(5,2);not null" json:"weight"`
	ExamTypes   StringArray `gorm:"type:jsonb;not null;default:'[]'" json:"exam_types"`
	Aggregation string      `gorm:"type:varchar(20);not null;default:'pooled'" json:"aggregation"` // pooled | average | best | latest
	SortOrder   int         `json:"sort_order"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// ClassLevelSubject maps a (global) Subject onto a school's own
// ClassLevel, with a per-school, per-level IsCore flag. Subject itself
// stays global and unchanged (it has no SchoolID and Name/Code are
// globally unique) - a core-subject flag can't live on Subject directly
// because whether a subject is core is a school's own policy decision,
// not an intrinsic property of the subject. Keying on ClassLevelID
// rather than the session-scoped Class means this mapping survives a
// session rollover with no copying.
//
// A school with no ClassLevelSubject rows for a given class level keeps
// today's behavior: the report card's subject list falls back to
// whichever subjects have an exam recorded for that class/term (the
// existing DISTINCT join in GetSubjectsByClassAndTerm).
type ClassLevelSubject struct {
	ID           string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SchoolID     string    `gorm:"type:uuid;not null;index" json:"school_id"`
	ClassLevelID string    `gorm:"type:uuid;not null;uniqueIndex:idx_class_level_subject" json:"class_level_id"`
	SubjectID    string    `gorm:"type:uuid;not null;uniqueIndex:idx_class_level_subject" json:"subject_id"`
	IsCore       bool      `gorm:"default:false" json:"is_core"`
	IsActive     bool      `json:"is_active"`
	SortOrder    int       `json:"sort_order"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
