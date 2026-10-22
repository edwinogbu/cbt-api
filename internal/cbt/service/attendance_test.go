package service

import (
	"context"
	"testing"
	"time"

	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/models"

	"github.com/google/uuid"
)

func TestMarkClassAttendance_HappyPathAndIdempotentReMark(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	ctx := context.Background()

	schoolID, classID, sessionID, termID := "school-1", "class-1", "session-1", "term-1"
	seedSchool(t, db, schoolID, "Test School", "", "")
	seedReportCardClass(t, db, classID, schoolID, "")
	seedReportCardSession(t, db, sessionID, schoolID, "2025/2026")
	seedReportCardTerm(t, db, termID, sessionID, 1, "First Term")
	seedReportCardStudent(t, db, "student-1", uuid.NewString(), schoolID, classID)
	seedReportCardStudent(t, db, "student-2", uuid.NewString(), schoolID, classID)

	date := time.Now().AddDate(0, 0, -1).Format(attendanceDateLayout)
	req := &dto.MarkClassAttendanceRequest{
		ClassID: classID, TermID: termID, SessionID: sessionID, Date: date,
		Records: []dto.AttendanceMarkEntry{
			{StudentID: "student-1", Status: "present"},
			{StudentID: "student-2", Status: "absent", Notes: "sick"},
		},
	}
	resp, err := svc.MarkClassAttendance(ctx, schoolID, "teacher-1", false, req)
	if err != nil {
		t.Fatalf("MarkClassAttendance failed: %v", err)
	}
	if len(resp.MarkedStudentIDs) != 2 || len(resp.Failed) != 0 {
		t.Fatalf("resp = %+v, want both students marked with no failures", resp)
	}

	// Re-mark student-1 as "late" for the same date - must update the
	// existing row, not create a second one.
	req2 := &dto.MarkClassAttendanceRequest{
		ClassID: classID, TermID: termID, SessionID: sessionID, Date: date,
		Records: []dto.AttendanceMarkEntry{{StudentID: "student-1", Status: "late"}},
	}
	if _, err := svc.MarkClassAttendance(ctx, schoolID, "teacher-1", false, req2); err != nil {
		t.Fatalf("re-mark failed: %v", err)
	}

	summary, err := svc.GetStudentAttendanceSummary(ctx, schoolID, false, "student-1", termID, sessionID)
	if err != nil {
		t.Fatalf("GetStudentAttendanceSummary failed: %v", err)
	}
	if summary.DaysSchoolOpen != 1 {
		t.Errorf("DaysSchoolOpen = %d, want 1 (one day marked, re-marking the same day must not double it)", summary.DaysSchoolOpen)
	}
	if summary.DaysPresent != 1 {
		t.Errorf("DaysPresent = %d, want 1 ('late' counts as present)", summary.DaysPresent)
	}
	if summary.DaysAbsent != 0 {
		t.Errorf("DaysAbsent = %d, want 0 (the re-mark to 'late' must have replaced the original mark, not added to it)", summary.DaysAbsent)
	}
}

func TestMarkClassAttendance_RejectsInvalidStatusAndStudentNotInClass(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	ctx := context.Background()

	schoolID, classID, sessionID, termID := "school-1", "class-1", "session-1", "term-1"
	seedSchool(t, db, schoolID, "Test School", "", "")
	seedReportCardClass(t, db, classID, schoolID, "")
	seedReportCardSession(t, db, sessionID, schoolID, "2025/2026")
	seedReportCardTerm(t, db, termID, sessionID, 1, "First Term")
	seedReportCardStudent(t, db, "student-1", uuid.NewString(), schoolID, classID)

	date := time.Now().AddDate(0, 0, -1).Format(attendanceDateLayout)
	req := &dto.MarkClassAttendanceRequest{
		ClassID: classID, TermID: termID, SessionID: sessionID, Date: date,
		Records: []dto.AttendanceMarkEntry{
			{StudentID: "student-1", Status: "on_fire"},         // invalid status
			{StudentID: "not-in-this-class", Status: "present"}, // not a class member
		},
	}
	resp, err := svc.MarkClassAttendance(ctx, schoolID, "teacher-1", false, req)
	if err != nil {
		t.Fatalf("MarkClassAttendance failed: %v", err)
	}
	if len(resp.MarkedStudentIDs) != 0 {
		t.Errorf("MarkedStudentIDs = %v, want none", resp.MarkedStudentIDs)
	}
	if len(resp.Failed) != 2 {
		t.Fatalf("Failed = %+v, want exactly 2 rejected entries", resp.Failed)
	}
}

func TestMarkClassAttendance_RejectsFutureDate(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	ctx := context.Background()

	seedSchool(t, db, "school-1", "Test School", "", "")
	seedReportCardClass(t, db, "class-1", "school-1", "")

	future := time.Now().AddDate(0, 0, 7).Format(attendanceDateLayout)
	req := &dto.MarkClassAttendanceRequest{
		ClassID: "class-1", TermID: "term-1", SessionID: "session-1", Date: future,
		Records: []dto.AttendanceMarkEntry{{StudentID: "student-1", Status: "present"}},
	}
	if _, err := svc.MarkClassAttendance(ctx, "school-1", "teacher-1", false, req); err == nil {
		t.Fatal("expected an error marking attendance for a future date, got nil")
	}
}

func TestGetClassAttendanceForDate_PrefillsExistingMarksLeavesOthersEmpty(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	ctx := context.Background()

	schoolID, classID, sessionID, termID := "school-1", "class-1", "session-1", "term-1"
	seedSchool(t, db, schoolID, "Test School", "", "")
	seedReportCardClass(t, db, classID, schoolID, "")
	seedReportCardSession(t, db, sessionID, schoolID, "2025/2026")
	seedReportCardTerm(t, db, termID, sessionID, 1, "First Term")
	seedReportCardStudent(t, db, "student-1", uuid.NewString(), schoolID, classID)
	seedReportCardStudent(t, db, "student-2", uuid.NewString(), schoolID, classID)

	date := time.Now().AddDate(0, 0, -1).Format(attendanceDateLayout)
	if _, err := svc.MarkClassAttendance(ctx, schoolID, "teacher-1", false, &dto.MarkClassAttendanceRequest{
		ClassID: classID, TermID: termID, SessionID: sessionID, Date: date,
		Records: []dto.AttendanceMarkEntry{{StudentID: "student-1", Status: "present"}},
	}); err != nil {
		t.Fatalf("MarkClassAttendance failed: %v", err)
	}

	register, err := svc.GetClassAttendanceForDate(ctx, schoolID, false, classID, date)
	if err != nil {
		t.Fatalf("GetClassAttendanceForDate failed: %v", err)
	}
	if len(register.Rows) != 2 {
		t.Fatalf("Rows length = %d, want 2", len(register.Rows))
	}
	byID := map[string]dto.ClassAttendanceRow{}
	for _, r := range register.Rows {
		byID[r.StudentID] = r
	}
	if byID["student-1"].Status != "present" {
		t.Errorf("student-1 status = %q, want present", byID["student-1"].Status)
	}
	if byID["student-2"].Status != "" {
		t.Errorf("student-2 status = %q, want empty (never marked yet, never inferred)", byID["student-2"].Status)
	}
}

func TestAttendanceSummary_NilUntilRecordsExistThenPopulatesTermlyAndMidTermReports(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	ctx := context.Background()

	schoolID, classID, sessionID, termID := "school-1", "class-1", "session-1", "term-1"
	seedSchool(t, db, schoolID, "Test School", "", "")
	seedReportCardClass(t, db, classID, schoolID, "")
	seedReportCardSession(t, db, sessionID, schoolID, "2025/2026")
	seedReportCardTerm(t, db, termID, sessionID, 1, "First Term")
	seedTermlySubject(t, db, "subj-1", "Mathematics")
	examID := "exam-main"
	seedReportCardExam(t, db, examID, "subj-1", classID, termID, sessionID, "main_exam", 100)
	userID := uuid.NewString()
	seedReportCardStudent(t, db, "student-1", userID, schoolID, classID)
	seedReportCardResult(t, db, "student-1", examID, 70, true)

	// Before any attendance is marked, both report types must show no
	// attendance section at all (nil), not an all-zero one.
	termlyBefore, err := svc.GetReportCardForUser(ctx, termID, sessionID, userID)
	if err != nil {
		t.Fatalf("GetReportCardForUser failed: %v", err)
	}
	if termlyBefore.Attendance != nil {
		t.Errorf("Attendance = %+v, want nil before any attendance is recorded", termlyBefore.Attendance)
	}

	// Mark 3 days: 2 present, 1 absent.
	for i, status := range []string{"present", "present", "absent"} {
		date := time.Now().AddDate(0, 0, -(i + 1)).Format(attendanceDateLayout)
		if _, err := svc.MarkClassAttendance(ctx, schoolID, "teacher-1", false, &dto.MarkClassAttendanceRequest{
			ClassID: classID, TermID: termID, SessionID: sessionID, Date: date,
			Records: []dto.AttendanceMarkEntry{{StudentID: "student-1", Status: status}},
		}); err != nil {
			t.Fatalf("MarkClassAttendance failed on day %d: %v", i, err)
		}
	}

	termlyAfter, err := svc.GetReportCardForUser(ctx, termID, sessionID, userID)
	if err != nil {
		t.Fatalf("GetReportCardForUser failed: %v", err)
	}
	if termlyAfter.Attendance == nil {
		t.Fatal("Attendance is nil, want populated now that records exist")
	}
	if termlyAfter.Attendance.DaysPresent != 2 || termlyAfter.Attendance.DaysAbsent != 1 || termlyAfter.Attendance.DaysSchoolOpen != 3 {
		t.Errorf("Attendance = %+v, want {present:2 absent:1 open:3}", termlyAfter.Attendance)
	}

	midTerm, err := svc.GetMidTermProgressReportForUser(ctx, termID, sessionID, userID)
	if err != nil {
		t.Fatalf("GetMidTermProgressReportForUser failed: %v", err)
	}
	if midTerm.Attendance == nil {
		t.Fatal("Mid-term Attendance is nil, want populated")
	}
	if midTerm.Attendance.DaysPossible != 3 || midTerm.Attendance.DaysPresent != 2 || midTerm.Attendance.DaysAbsent != 1 {
		t.Errorf("Mid-term Attendance = %+v, want {possible:3 present:2 absent:1}", midTerm.Attendance)
	}
}

func TestGetMyAttendanceSummary_ResolvesFromUserID(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	ctx := context.Background()

	schoolID, classID, sessionID, termID := "school-1", "class-1", "session-1", "term-1"
	seedSchool(t, db, schoolID, "Test School", "", "")
	seedReportCardClass(t, db, classID, schoolID, "")
	userID := uuid.NewString()
	seedReportCardStudent(t, db, "student-1", userID, schoolID, classID)

	date := time.Now().AddDate(0, 0, -1).Format(attendanceDateLayout)
	if _, err := svc.MarkClassAttendance(ctx, schoolID, "teacher-1", false, &dto.MarkClassAttendanceRequest{
		ClassID: classID, TermID: termID, SessionID: sessionID, Date: date,
		Records: []dto.AttendanceMarkEntry{{StudentID: "student-1", Status: "present"}},
	}); err != nil {
		t.Fatalf("MarkClassAttendance failed: %v", err)
	}

	summary, err := svc.GetMyAttendanceSummary(ctx, userID, termID, sessionID)
	if err != nil {
		t.Fatalf("GetMyAttendanceSummary failed: %v", err)
	}
	if summary.StudentID != "student-1" || summary.DaysPresent != 1 {
		t.Errorf("summary = %+v, want student-1 with 1 day present", summary)
	}
}

// TestMarkClassAttendance_CorrectionResetsSyncFlag is the regression
// test for the nodesync outbox bug this feature would otherwise have:
// a mark already pushed to the cloud (synced_to_cloud_at set) that is
// later corrected locally must re-enter the node's outbox, never stay
// "synced" under its stale pre-correction content.
func TestMarkClassAttendance_CorrectionResetsSyncFlag(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	ctx := context.Background()

	schoolID, classID, sessionID, termID := "school-1", "class-1", "session-1", "term-1"
	seedSchool(t, db, schoolID, "Test School", "", "")
	seedReportCardClass(t, db, classID, schoolID, "")
	seedReportCardStudent(t, db, "student-1", uuid.NewString(), schoolID, classID)

	date := time.Now().AddDate(0, 0, -1).Format(attendanceDateLayout)
	req := &dto.MarkClassAttendanceRequest{
		ClassID: classID, TermID: termID, SessionID: sessionID, Date: date,
		Records: []dto.AttendanceMarkEntry{{StudentID: "student-1", Status: "absent"}},
	}
	if _, err := svc.MarkClassAttendance(ctx, schoolID, "teacher-1", false, req); err != nil {
		t.Fatalf("MarkClassAttendance failed: %v", err)
	}

	// Simulate the record having already been pushed to the cloud.
	// Queried by student_id alone (only one record exists for this
	// student in this test) rather than re-deriving the stored date
	// representation - MarkClassAttendance itself parses "YYYY-MM-DD"
	// into a time.Time before storing, so a raw string comparison here
	// would not reliably match the driver's own stored format.
	if err := db.Model(&models.AttendanceRecord{}).Where("student_id = ?", "student-1").
		Update("synced_to_cloud_at", time.Now()).Error; err != nil {
		t.Fatalf("failed to simulate a prior sync: %v", err)
	}

	// Correct the mark to "present".
	correction := &dto.MarkClassAttendanceRequest{
		ClassID: classID, TermID: termID, SessionID: sessionID, Date: date,
		Records: []dto.AttendanceMarkEntry{{StudentID: "student-1", Status: "present"}},
	}
	if _, err := svc.MarkClassAttendance(ctx, schoolID, "teacher-1", false, correction); err != nil {
		t.Fatalf("correction MarkClassAttendance failed: %v", err)
	}

	var record models.AttendanceRecord
	if err := db.Where("student_id = ?", "student-1").First(&record).Error; err != nil {
		t.Fatalf("failed to load record: %v", err)
	}
	if record.Status != "present" {
		t.Errorf("Status = %q, want present after correction", record.Status)
	}
	if record.SyncedToCloudAt != nil {
		t.Error("SyncedToCloudAt is still set after a correction - the corrected mark will never be re-pushed to the cloud")
	}
}
