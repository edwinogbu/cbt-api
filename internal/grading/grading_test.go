package grading

import (
	"context"
	"testing"

	"cbt-api/internal/models"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// setupGradingTestDB mirrors internal/cbt/service/exam_service_test.go's
// pattern: hand-written SQLite-compatible DDL (production's
// `default:gen_random_uuid()` tags are Postgres-only), one uniquely
// named in-memory DB per test.
func setupGradingTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	ddl := []string{
		`CREATE TABLE grading_scales (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			school_id TEXT, class_level_id TEXT, scope_key TEXT,
			name TEXT, pass_mark REAL DEFAULT 50, is_active INTEGER DEFAULT 1,
			updated_by TEXT, created_at DATETIME, updated_at DATETIME
		)`,
		`CREATE TABLE grading_scale_bands (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			grading_scale_id TEXT, grade TEXT, min_percent REAL,
			grade_point REAL DEFAULT 0, remark TEXT, sort_order INTEGER,
			created_at DATETIME, updated_at DATETIME
		)`,
		`CREATE TABLE assessment_schemes (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			school_id TEXT, class_level_id TEXT, subject_id TEXT, scope_key TEXT,
			missing_policy TEXT DEFAULT 'renormalize', is_active INTEGER DEFAULT 1,
			updated_by TEXT, created_at DATETIME, updated_at DATETIME
		)`,
		`CREATE TABLE assessment_components (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			scheme_id TEXT, code TEXT, name TEXT, weight REAL,
			exam_types TEXT DEFAULT '[]', aggregation TEXT DEFAULT 'pooled',
			sort_order INTEGER, created_at DATETIME, updated_at DATETIME
		)`,
	}
	for _, stmt := range ddl {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("failed to create table: %v\nDDL: %s", err, stmt)
		}
	}
	return db
}

// ---- Builtin/FromBands: pure-function parity against the exact
// boundaries this package replaces ----

func TestBuiltin_MatchesPreExistingHardcodedBoundaries(t *testing.T) {
	// These cases mirror dto.GetGrade's own switch statement
	// (internal/cbt/dto/exam_dto.go) and exam_repo.go's
	// getGradeFromPercentage exactly - both used A>=70, B>=60, C>=50,
	// D>=45, E>=40, F otherwise, before this package existed.
	cases := []struct {
		pct        float64
		wantGrade  string
		wantGP     float64
		wantRemark string
	}{
		{100, "A", 5.0, "Excellent"},
		{70, "A", 5.0, "Excellent"},
		{69.99, "B", 4.0, "Very Good"},
		{60, "B", 4.0, "Very Good"},
		{59.99, "C", 3.0, "Good"},
		{50, "C", 3.0, "Good"},
		{49.99, "D", 2.0, "Fair"},
		{45, "D", 2.0, "Fair"},
		{44.99, "E", 1.0, "Pass"},
		{40, "E", 1.0, "Pass"},
		{39.99, "F", 0.0, "Fail"},
		{0, "F", 0.0, "Fail"},
	}
	for _, c := range cases {
		grade, gp, remark := Builtin(c.pct)
		if grade != c.wantGrade || gp != c.wantGP || remark != c.wantRemark {
			t.Errorf("Builtin(%.2f) = (%s, %.1f, %s), want (%s, %.1f, %s)",
				c.pct, grade, gp, remark, c.wantGrade, c.wantGP, c.wantRemark)
		}
	}
}

func TestFromBands_EmptyBandsReturnsZeroValue(t *testing.T) {
	band := FromBands(nil, 75)
	if band.Grade != "" {
		t.Errorf("FromBands(nil, 75) = %+v, want zero-value Band", band)
	}
}

func TestFromBands_CustomScaleDiffersFromBuiltin(t *testing.T) {
	// A school's own scale with a stricter boundary than the built-in
	// default (A requires 75%, not 70%) - proves FromBands actually
	// uses the given bands, not a hidden reference to BuiltinBands.
	custom := []Band{
		{Grade: "A", MinPercent: 75, GradePoint: 5.0, Remark: "Distinction"},
		{Grade: "B", MinPercent: 50, GradePoint: 3.0, Remark: "Credit"},
		{Grade: "F", MinPercent: 0, GradePoint: 0.0, Remark: "Fail"},
	}
	got := FromBands(custom, 72)
	if got.Grade != "B" {
		t.Errorf("FromBands(custom, 72) grade = %s, want B (72%% is below this school's custom 75%% A threshold)", got.Grade)
	}
	if builtinGrade, _, _ := Builtin(72); builtinGrade != "A" {
		t.Fatalf("test assumption broken: Builtin(72) = %s, want A (this test's whole point is that custom differs from builtin)", builtinGrade)
	}
}

// ---- ValidateBands ----

func TestValidateBands_RejectsMissingZeroBand(t *testing.T) {
	bands := []Band{{Grade: "A", MinPercent: 50}}
	if err := ValidateBands(bands); err == nil {
		t.Error("ValidateBands with no band starting at 0% should fail - a 0-49% score would have no matching band")
	}
}

func TestValidateBands_RejectsDuplicateBoundary(t *testing.T) {
	bands := []Band{
		{Grade: "A", MinPercent: 50},
		{Grade: "B", MinPercent: 50},
		{Grade: "F", MinPercent: 0},
	}
	if err := ValidateBands(bands); err == nil {
		t.Error("ValidateBands with two bands sharing MinPercent=50 should fail - ambiguous which grade a 50% score gets")
	}
}

func TestValidateBands_AcceptsWellFormedScale(t *testing.T) {
	bands := []Band{
		{Grade: "A", MinPercent: 75, GradePoint: 5.0, Remark: "Distinction"},
		{Grade: "B", MinPercent: 50, GradePoint: 3.0, Remark: "Credit"},
		{Grade: "F", MinPercent: 0, GradePoint: 0.0, Remark: "Fail"},
	}
	if err := ValidateBands(bands); err != nil {
		t.Errorf("ValidateBands rejected a well-formed scale: %v", err)
	}
}

// ---- Service.ResolveScale / ComputeGrade: DB-backed resolution ----

func TestResolveScale_NoConfigFallsBackToBuiltin(t *testing.T) {
	db := setupGradingTestDB(t)
	svc := NewService(db)
	resolved := svc.ResolveScale(context.Background(), "school-1", "level-1")
	if resolved.Source != "builtin_default" {
		t.Errorf("Source = %s, want builtin_default for a school with no configured scale", resolved.Source)
	}
	if resolved.PassMark != BuiltinPassMark {
		t.Errorf("PassMark = %.1f, want the builtin %.1f", resolved.PassMark, BuiltinPassMark)
	}
}

func TestResolveScale_NilServiceFallsBackToBuiltin(t *testing.T) {
	var svc *Service
	resolved := svc.ResolveScale(context.Background(), "school-1", "")
	if resolved.Source != "builtin_default" {
		t.Errorf("a nil Service should always fall back to builtin_default, got Source=%s", resolved.Source)
	}
}

func seedSchoolWideScale(t *testing.T, db *gorm.DB, schoolID string) {
	t.Helper()
	scale := models.GradingScale{
		SchoolID: schoolID,
		ScopeKey: "school",
		Name:     "Custom School Scale",
		PassMark: 45,
		IsActive: true,
		Bands: []models.GradingScaleBand{
			{Grade: "A", MinPercent: 80, GradePoint: 5, Remark: "Distinction"},
			{Grade: "B", MinPercent: 45, GradePoint: 3, Remark: "Credit"},
			{Grade: "F", MinPercent: 0, GradePoint: 0, Remark: "Fail"},
		},
	}
	if err := db.Create(&scale).Error; err != nil {
		t.Fatalf("failed to seed school-wide grading scale: %v", err)
	}
}

func seedClassLevelScale(t *testing.T, db *gorm.DB, schoolID, classLevelID string) {
	t.Helper()
	scale := models.GradingScale{
		SchoolID:     schoolID,
		ClassLevelID: &classLevelID,
		ScopeKey:     "level:" + classLevelID,
		Name:         "SSS WAEC-style Scale",
		PassMark:     40,
		IsActive:     true,
		Bands: []models.GradingScaleBand{
			{Grade: "A1", MinPercent: 75, GradePoint: 5, Remark: "Excellent"},
			{Grade: "F9", MinPercent: 0, GradePoint: 0, Remark: "Fail"},
		},
	}
	if err := db.Create(&scale).Error; err != nil {
		t.Fatalf("failed to seed class-level grading scale: %v", err)
	}
}

func TestResolveScale_UsesSchoolWideScaleWhenConfigured(t *testing.T) {
	db := setupGradingTestDB(t)
	seedSchoolWideScale(t, db, "school-1")
	svc := NewService(db)

	resolved := svc.ResolveScale(context.Background(), "school-1", "")
	if resolved.Source != "school" {
		t.Fatalf("Source = %s, want school", resolved.Source)
	}
	if resolved.PassMark != 45 {
		t.Errorf("PassMark = %.1f, want this school's configured 45", resolved.PassMark)
	}
	band := FromBands(resolved.Bands, 80)
	if band.Grade != "A" || band.Remark != "Distinction" {
		t.Errorf("80%% graded as %+v, want this school's own A/Distinction band (80%% is below the builtin's 70%% A but matches this school's custom 80%% boundary too)", band)
	}
}

func TestResolveScale_ClassLevelScaleTakesPrecedenceOverSchoolWide(t *testing.T) {
	db := setupGradingTestDB(t)
	seedSchoolWideScale(t, db, "school-1")
	seedClassLevelScale(t, db, "school-1", "level-sss")
	svc := NewService(db)

	// A class level WITH its own scale configured should use it, not
	// the school-wide one.
	resolved := svc.ResolveScale(context.Background(), "school-1", "level-sss")
	if resolved.Source != "class_level" {
		t.Fatalf("Source = %s, want class_level", resolved.Source)
	}
	band := FromBands(resolved.Bands, 75)
	if band.Grade != "A1" {
		t.Errorf("75%% graded as %s, want this class level's own A1 band", band.Grade)
	}

	// A DIFFERENT class level with no scale of its own should fall back
	// to the school-wide scale, not the other level's scale and not the
	// builtin default.
	resolvedOtherLevel := svc.ResolveScale(context.Background(), "school-1", "level-jss")
	if resolvedOtherLevel.Source != "school" {
		t.Errorf("a class level with no scale of its own should fall back to the school-wide scale, got Source=%s", resolvedOtherLevel.Source)
	}
}

func TestResolveScale_InactiveScaleIsIgnored(t *testing.T) {
	db := setupGradingTestDB(t)
	scale := models.GradingScale{
		SchoolID: "school-1",
		ScopeKey: "school",
		Name:     "Disabled Scale",
		PassMark: 45,
		IsActive: false, // deliberately inactive
		Bands: []models.GradingScaleBand{
			{Grade: "A", MinPercent: 80, GradePoint: 5, Remark: "Distinction"},
			{Grade: "F", MinPercent: 0, GradePoint: 0, Remark: "Fail"},
		},
	}
	if err := db.Create(&scale).Error; err != nil {
		t.Fatalf("seed failed: %v", err)
	}
	svc := NewService(db)
	resolved := svc.ResolveScale(context.Background(), "school-1", "")
	if resolved.Source != "builtin_default" {
		t.Errorf("an inactive scale must be ignored, got Source=%s", resolved.Source)
	}
}

func TestResolveScale_DifferentSchoolsAreIsolated(t *testing.T) {
	db := setupGradingTestDB(t)
	seedSchoolWideScale(t, db, "school-1")
	svc := NewService(db)

	resolved := svc.ResolveScale(context.Background(), "school-2", "")
	if resolved.Source != "builtin_default" {
		t.Errorf("school-2 must never see school-1's configured scale, got Source=%s", resolved.Source)
	}
}

func TestComputeGrade_EndToEndWithCustomScaleAndPassMark(t *testing.T) {
	db := setupGradingTestDB(t)
	seedSchoolWideScale(t, db, "school-1") // pass mark 45, A>=80, B>=45, F<45
	svc := NewService(db)

	// 50% fails the builtin's own grading (C, which IS a pass under the
	// builtin's 50 pass mark) differently from this school's real
	// policy: under this school's configured scale, 50% is a B
	// (Credit) and passes (pass mark 45) - proving ComputeGrade used
	// the resolved scale end-to-end, not a mix of builtin grade logic
	// with custom pass mark or vice versa.
	result := svc.ComputeGrade(context.Background(), "school-1", "", 50)
	if result.Grade != "B" || result.Remark != "Credit" {
		t.Errorf("Grade/Remark = %s/%s, want B/Credit per this school's configured scale", result.Grade, result.Remark)
	}
	if !result.Passed {
		t.Errorf("Passed = false, want true (50%% >= this school's configured 45%% pass mark)")
	}
	if result.Source != "school" || result.ScaleID == nil {
		t.Errorf("Source/ScaleID = %s/%v, want school/<real id>", result.Source, result.ScaleID)
	}
}

func TestComputeGrade_NoConfigMatchesBuiltinExactly(t *testing.T) {
	db := setupGradingTestDB(t)
	svc := NewService(db)
	result := svc.ComputeGrade(context.Background(), "school-with-no-config", "", 72)
	grade, gp, remark := Builtin(72)
	if result.Grade != grade || result.GradePoint != gp || result.Remark != remark {
		t.Errorf("ComputeGrade with no config = %+v, want it to exactly match Builtin(72) = (%s,%.1f,%s)", result, grade, gp, remark)
	}
	if result.Source != "builtin_default" {
		t.Errorf("Source = %s, want builtin_default", result.Source)
	}
}
