package service

import (
	"context"
	"testing"
)

func TestGetClassBroadSheet_ShowsPerSubjectCellsAndIncompleteCounts(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	schoolID, classID, sessionID, termID := "school-1", "class-1", "session-1", "term-1"
	seedSchool(t, db, schoolID, "Test School", "", "")
	seedReportCardClass(t, db, classID, schoolID, "")
	seedReportCardSession(t, db, sessionID, schoolID, "2025/2026")
	seedReportCardTerm(t, db, termID, sessionID, 1, "First Term")
	seedTermlySubject(t, db, "subj-math", "Mathematics")
	seedTermlySubject(t, db, "subj-eng", "English")
	seedReportCardStudent(t, db, "student-1", "user-1", schoolID, classID)
	seedReportCardStudent(t, db, "student-2", "user-2", schoolID, classID)

	seedReportCardExam(t, db, "exam-math-1", "subj-math", classID, termID, sessionID, "main_exam", 100)
	seedReportCardExam(t, db, "exam-eng-1", "subj-eng", classID, termID, sessionID, "main_exam", 100)

	// student-1: complete in both subjects (one pass at 80%, one clear
	// fail at 20% - well under BuiltinBands' lowest non-F band, which
	// starts at 40%).
	seedReportCardResult(t, db, "student-1", "exam-math-1", 80, true)
	seedReportCardResult(t, db, "student-1", "exam-eng-1", 20, true)
	// student-2: only Mathematics recorded - English is missing entirely.
	seedReportCardResult(t, db, "student-2", "exam-math-1", 60, true)

	resp, err := svc.GetClassBroadSheet(context.Background(), schoolID, false, classID, "termly", termID, sessionID)
	if err != nil {
		t.Fatalf("GetClassBroadSheet failed: %v", err)
	}
	if len(resp.SubjectCodes) != 2 {
		t.Fatalf("expected 2 subject columns, got %d: %v", len(resp.SubjectCodes), resp.SubjectCodes)
	}
	if resp.StudentCount != 2 {
		t.Fatalf("expected 2 students, got %d", resp.StudentCount)
	}

	for _, row := range resp.Rows {
		switch row.StudentID {
		case "student-1":
			if row.Pass != 1 || row.Fail != 1 || row.Incomplete != 0 {
				t.Errorf("student-1: pass/fail/incomplete = %d/%d/%d, want 1/1/0", row.Pass, row.Fail, row.Incomplete)
			}
		case "student-2":
			if row.Incomplete != 1 {
				t.Errorf("student-2: incomplete = %d, want 1 (English never recorded)", row.Incomplete)
			}
		}
	}
	if resp.StudentsWithIncompleteResults != 1 {
		t.Errorf("StudentsWithIncompleteResults = %d, want 1 (only student-2)", resp.StudentsWithIncompleteResults)
	}
}

func TestGetSubjectBroadSheet_AggregatesAcrossClasses(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedRegisterClass(t, db, "class-1", "school-1")
	seedRegisterClass(t, db, "class-2", "school-1")
	seedTermlySubject(t, db, "subj-math", "Mathematics")
	if err := db.Exec("INSERT INTO users (id, first_name, last_name, role) VALUES ('teacher-1','Ada','Lovelace','teacher')").Error; err != nil {
		t.Fatalf("seed teacher failed: %v", err)
	}
	if err := db.Exec("INSERT INTO teacher_subject_assignments (id, school_id, teacher_id, subject_id, class_id, session_id, is_active) VALUES ('a1','school-1','teacher-1','subj-math','class-1','session-1',1)").Error; err != nil {
		t.Fatalf("seed assignment failed: %v", err)
	}
	if err := db.Exec("INSERT INTO teacher_subject_assignments (id, school_id, teacher_id, subject_id, class_id, session_id, is_active) VALUES ('a2','school-1','teacher-1','subj-math','class-2','session-1',1)").Error; err != nil {
		t.Fatalf("seed assignment failed: %v", err)
	}
	seedRegisterStudent(t, db, "student-1", "class-1", "school-1", "A001", "Amara")
	seedRegisterStudent(t, db, "student-2", "class-2", "school-1", "B001", "Bayo")
	seedRegisterExam(t, db, "exam-math-c1", "subj-math", "class-1", "term-1", "session-1", "main_exam", 100, nil)
	seedRegisterExam(t, db, "exam-math-c2", "subj-math", "class-2", "term-1", "session-1", "main_exam", 100, nil)
	if err := db.Exec("INSERT INTO results (id, student_id, exam_id, total_score, percentage, grade, fully_graded) VALUES ('r1','student-1','exam-math-c1',90,90,'A',1)").Error; err != nil {
		t.Fatalf("seed result failed: %v", err)
	}
	// class-2's student has no result yet - that class should show incomplete.

	resp, err := svc.GetSubjectBroadSheet(context.Background(), "school-1", false, "subj-math", "term-1", "session-1")
	if err != nil {
		t.Fatalf("GetSubjectBroadSheet failed: %v", err)
	}
	if len(resp.Rows) != 2 {
		t.Fatalf("expected 2 class rows, got %d", len(resp.Rows))
	}
	byClass := map[string]int{}
	for i, r := range resp.Rows {
		byClass[r.ClassID] = i
	}
	c1 := resp.Rows[byClass["class-1"]]
	if c1.Enrolled != 1 || c1.MarksEntered != 1 || c1.Incomplete != 0 {
		t.Errorf("class-1: enrolled/entered/incomplete = %d/%d/%d, want 1/1/0", c1.Enrolled, c1.MarksEntered, c1.Incomplete)
	}
	c2 := resp.Rows[byClass["class-2"]]
	if c2.Enrolled != 1 || c2.MarksEntered != 0 || c2.Incomplete != 1 {
		t.Errorf("class-2: enrolled/entered/incomplete = %d/%d/%d, want 1/0/1", c2.Enrolled, c2.MarksEntered, c2.Incomplete)
	}
}

func TestGetSchoolBroadSheet_CountsSubmissionStates(t *testing.T) {
	db := setupExamTestDB(t)
	svc := newTestExamService(db)
	seedRegisterClass(t, db, "class-1", "school-1")
	seedTermlySubject(t, db, "subj-math", "Mathematics")
	seedTermlySubject(t, db, "subj-eng", "English")
	if err := db.Exec("INSERT INTO users (id, first_name, last_name, role) VALUES ('teacher-1','Ada','Lovelace','teacher')").Error; err != nil {
		t.Fatalf("seed teacher failed: %v", err)
	}
	if err := db.Exec("INSERT INTO teacher_subject_assignments (id, school_id, teacher_id, subject_id, class_id, session_id, is_active) VALUES ('a1','school-1','teacher-1','subj-math','class-1','session-1',1)").Error; err != nil {
		t.Fatalf("seed assignment failed: %v", err)
	}
	if err := db.Exec("INSERT INTO teacher_subject_assignments (id, school_id, teacher_id, subject_id, class_id, session_id, is_active) VALUES ('a2','school-1','teacher-1','subj-eng','class-1','session-1',1)").Error; err != nil {
		t.Fatalf("seed assignment failed: %v", err)
	}
	// Mathematics approved, English never submitted.
	if err := db.Exec(
		"INSERT INTO subject_result_submissions (id, school_id, class_id, subject_id, term_id, session_id, status, version, teacher_id) VALUES ('s1','school-1','class-1','subj-math','term-1','session-1','approved',1,'teacher-1')",
	).Error; err != nil {
		t.Fatalf("seed submission failed: %v", err)
	}

	resp, err := svc.GetSchoolBroadSheet(context.Background(), "school-1", false, "", "term-1", "session-1", "termly")
	if err != nil {
		t.Fatalf("GetSchoolBroadSheet failed: %v", err)
	}
	if resp.SubjectResultSets != 2 {
		t.Fatalf("expected 2 subject result sets (rows), got %d", resp.SubjectResultSets)
	}
	if resp.Approved != 1 || resp.Published != 1 {
		t.Errorf("Approved/Published = %d/%d, want 1/1", resp.Approved, resp.Published)
	}
	if resp.SubmissionsComplete != 1 {
		t.Errorf("SubmissionsComplete = %d, want 1 (only Mathematics)", resp.SubmissionsComplete)
	}
	if resp.PendingIncomplete != 1 {
		t.Errorf("PendingIncomplete = %d, want 1 (English never submitted)", resp.PendingIncomplete)
	}
}

