package service

import (
	"context"
	"testing"

	"cbt-api/internal/cbt/dto"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---- CreateExam/CreateExamWithContext: class must belong to caller's school ----
//
// Before this fix, neither method checked that req.ClassID belonged to
// the caller's own school at all - any caller able to reach these
// (previously: anyone, since the whole /exams group had no role gate
// either) could create an exam under ANY school's class platform-wide,
// injecting content into a tenant they have no relationship with.

func newExamDTO(classID string) *dto.CreateExamRequest {
	return &dto.CreateExamRequest{
		Title:           "Mid-Term",
		SubjectID:       uuid.NewString(),
		ClassID:         classID,
		SessionID:       uuid.NewString(),
		TermID:          uuid.NewString(),
		DurationMinutes: 30,
		TotalMarks:      10,
		PassMark:        5,
	}
}

func newExamWithContextDTO(classID string) *dto.CreateExamWithContextRequest {
	return &dto.CreateExamWithContextRequest{
		SchoolID:        uuid.NewString(),
		SessionID:       uuid.NewString(),
		TermID:          uuid.NewString(),
		SubjectID:       uuid.NewString(),
		ClassID:         classID,
		Title:           "Mid-Term",
		ExamType:        "main_exam",
		DurationMinutes: 30,
		TotalMarks:      10,
		PassMark:        5,
	}
}

func TestCreateExam_CrossTenantClass_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	classB := seedExamClass(t, db, "school-B")

	if _, err := svc.CreateExam(context.Background(), newExamDTO(classB), "school-A", false); err == nil {
		t.Fatal("expected School A caller to be rejected creating an exam under School B's class")
	}
}

func TestCreateExam_SameTenantClass_Succeeds(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	classA := seedExamClass(t, db, "school-A")

	if _, err := svc.CreateExam(context.Background(), newExamDTO(classA), "school-A", false); err != nil {
		t.Fatalf("expected School A caller to create an exam under its own class, got error: %v", err)
	}
}

func TestCreateExam_PlatformOwner_Bypasses(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	classB := seedExamClass(t, db, "school-B")

	if _, err := svc.CreateExam(context.Background(), newExamDTO(classB), "", true); err != nil {
		t.Fatalf("expected platform_owner to create an exam under any school's class, got error: %v", err)
	}
}

func TestCreateExamWithContext_CrossTenantClass_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	classB := seedExamClass(t, db, "school-B")

	if _, err := svc.CreateExamWithContext(context.Background(), newExamWithContextDTO(classB), "school-A", false); err == nil {
		t.Fatal("expected School A caller to be rejected creating an exam (with context) under School B's class")
	}
}

// ---- GetStudentPerformance: must be scoped to caller's own school ----
//
// Before this fix, studentID (taken from the URL path) had no check
// against the caller's school at all - any authenticated caller, from
// any school, could pull any student's full performance history
// platform-wide.

func patchStudentsTableForPerformance(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec("ALTER TABLE students ADD COLUMN school_id TEXT").Error; err != nil {
		t.Fatalf("failed to patch students table (school_id): %v", err)
	}
	if err := db.Exec("ALTER TABLE students ADD COLUMN deleted_at DATETIME").Error; err != nil {
		t.Fatalf("failed to patch students table (deleted_at): %v", err)
	}
}

func TestGetStudentPerformance_CrossTenant_Rejected(t *testing.T) {
	db := setupExamTestDB(t)
	patchStudentsTableForPerformance(t, db)
	svc := newTestExamService(db)

	studentID := uuid.NewString()
	if err := db.Exec("INSERT INTO students (id, user_id, school_id) VALUES (?, ?, ?)", studentID, uuid.NewString(), "school-B").Error; err != nil {
		t.Fatalf("failed to seed student: %v", err)
	}

	if _, err := svc.GetStudentPerformance(context.Background(), studentID, "school-A", false); err == nil {
		t.Fatal("expected School A caller to be rejected reading School B's student performance")
	}
}

func TestGetStudentPerformance_SameTenant_Succeeds(t *testing.T) {
	db := setupExamTestDB(t)
	patchStudentsTableForPerformance(t, db)
	svc := newTestExamService(db)

	studentID := uuid.NewString()
	if err := db.Exec("INSERT INTO students (id, user_id, school_id) VALUES (?, ?, ?)", studentID, uuid.NewString(), "school-A").Error; err != nil {
		t.Fatalf("failed to seed student: %v", err)
	}

	if _, err := svc.GetStudentPerformance(context.Background(), studentID, "school-A", false); err != nil {
		t.Fatalf("expected School A caller to read its own student's performance, got error: %v", err)
	}
}

func TestGetStudentPerformance_PlatformOwner_Bypasses(t *testing.T) {
	db := setupExamTestDB(t)
	patchStudentsTableForPerformance(t, db)
	svc := newTestExamService(db)

	studentID := uuid.NewString()
	if err := db.Exec("INSERT INTO students (id, user_id, school_id) VALUES (?, ?, ?)", studentID, uuid.NewString(), "school-B").Error; err != nil {
		t.Fatalf("failed to seed student: %v", err)
	}

	if _, err := svc.GetStudentPerformance(context.Background(), studentID, "", true); err != nil {
		t.Fatalf("expected platform_owner to read any school's student performance, got error: %v", err)
	}
}
