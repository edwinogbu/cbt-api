package service

import (
	"context"
	"errors"
	"sort"

	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/models"
)

// GetClassBroadSheet is the Class Teacher Broad Sheet (template B):
// every enrolled student's row, one column per subject taught in this
// class/term, each cell's Score/Grade/IsComplete computed through
// exactly the same report-card builder (collectClassReportCards ->
// GetTermlyReportCardForStudent/GetMidTermProgressReport/
// GetPromotionReportCard, already shared with the report-card
// approvals/export endpoints) rather than a second calculation - the
// spec's core requirement that broad sheets and report cards can never
// disagree because they're reading the same authoritative records.
func (s *ExamService) GetClassBroadSheet(ctx context.Context, schoolID string, isPlatformOwner bool, classID, reportType, termID, sessionID string) (*dto.ClassBroadSheetResponse, error) {
	entries, className, err := s.collectClassReportCards(ctx, schoolID, isPlatformOwner, classID, reportType, termID, sessionID)
	if err != nil {
		return nil, err
	}

	type subjectHeader struct{ code, name string }
	var headers []subjectHeader
	seen := map[string]bool{}
	for _, e := range entries {
		for _, subj := range reportCardSubjects(e) {
			if !seen[subj.SubjectCode] {
				seen[subj.SubjectCode] = true
				headers = append(headers, subjectHeader{code: subj.SubjectCode, name: subj.SubjectName})
			}
		}
	}

	normalizedTermID := termID
	if reportType == "promotion" {
		normalizedTermID = ""
	}

	resp := &dto.ClassBroadSheetResponse{
		ClassID: classID, ClassName: className, TermID: normalizedTermID, SessionID: sessionID,
		ReportType: reportType, StudentCount: len(entries),
	}
	resp.SubjectCodes = make([]string, len(headers))
	resp.SubjectNames = make([]string, len(headers))
	for i, h := range headers {
		resp.SubjectCodes[i], resp.SubjectNames[i] = h.code, h.name
	}

	incompleteStudents := 0
	for _, e := range entries {
		row := dto.ClassBroadSheetRow{StudentID: e.student.ID, AdmissionNumber: e.student.AdmissionNo, StudentName: e.studentName()}
		bySubj := map[string]dto.ReportCardSubject{}
		for _, subj := range reportCardSubjects(e) {
			bySubj[subj.SubjectCode] = subj
		}
		row.Subjects = make([]dto.ClassBroadSheetSubjectCell, len(headers))
		anyIncomplete := e.err != nil
		for i, h := range headers {
			cell := dto.ClassBroadSheetSubjectCell{SubjectCode: h.code}
			if subj, ok := bySubj[h.code]; ok {
				score := subj.TotalScore
				cell.Score, cell.Grade, cell.IsComplete = &score, subj.Grade, subj.IsComplete
				if subj.Grade == "F" {
					row.Fail++
				} else if subj.IsComplete {
					row.Pass++
				}
				if !subj.IsComplete {
					row.Incomplete++
					anyIncomplete = true
				}
			} else {
				row.Incomplete++
				anyIncomplete = true
			}
			row.Subjects[i] = cell
		}
		if anyIncomplete {
			incompleteStudents++
		}
		if e.err == nil {
			pct, _, _ := e.overallFigures()
			row.OverallAverage = &pct
		}
		if reportType != "promotion" {
			present, _, open, attErr := s.attendanceSummary(ctx, e.student.ID, classID, normalizedTermID)
			if attErr == nil {
				row.DaysPresent, row.DaysOpen = present, open
			}
		}
		resp.Rows = append(resp.Rows, row)
	}
	resp.StudentsWithIncompleteResults = incompleteStudents

	if reportType == "termly" && termID != "" {
		subjects, err := s.examRepo.GetSubjectsByClassAndTerm(ctx, classID, termID)
		if err == nil && len(subjects) > 0 {
			allSubmitted := true
			for _, subj := range subjects {
				sub, subErr := s.findSubjectResultSubmission(ctx, classSchoolIDFallback(s, ctx, classID, schoolID, isPlatformOwner), classID, subj.ID, termID, sessionID)
				if subErr != nil || sub == nil || sub.Status == "draft" || sub.Status == "returned" {
					allSubmitted = false
					break
				}
			}
			resp.AllSubjectsSubmitted = allSubmitted
		}
	}

	return resp, nil
}

// reportCardSubjects extracts the one populated report type's Subjects
// slice from a classReportCardEntry - nil (not an error) when the
// entry itself errored, matching overallFigures' own "default" case.
func reportCardSubjects(e classReportCardEntry) []dto.ReportCardSubject {
	switch {
	case e.termly != nil:
		return e.termly.Subjects
	case e.midterm != nil:
		out := make([]dto.ReportCardSubject, len(e.midterm.Subjects))
		for i, s := range e.midterm.Subjects {
			out[i] = dto.ReportCardSubject{
				SubjectName: s.SubjectName, SubjectCode: s.SubjectCode,
				TotalScore: s.TotalScore, Percentage: s.Percentage, IsComplete: s.IsComplete,
			}
		}
		return out
	case e.promotion != nil:
		return e.promotion.Subjects
	default:
		return nil
	}
}

// classSchoolIDFallback resolves a class's real SchoolID for a
// SubjectResultSubmission lookup - GetClassBroadSheet only has the
// caller's own schoolID (fine for a non-platform-owner, since it must
// already match), but a platform_owner's caller-side schoolID can be
// empty, so this falls back to the class's own record in that case.
func classSchoolIDFallback(s *ExamService, ctx context.Context, classID, schoolID string, isPlatformOwner bool) string {
	if !isPlatformOwner && schoolID != "" {
		return schoolID
	}
	if class, err := s.getClassByID(ctx, classID); err == nil {
		return class.SchoolID
	}
	return schoolID
}

// GetSubjectBroadSheet is the HOD/Examination Officer's own view: one
// subject's standing across every class it's taught in for a term -
// the fourth broad sheet the spec describes in text but the attached
// templates don't render a page for, modeled consistently with the
// other three (same completeness/pass-rate/submission-status shape).
func (s *ExamService) GetSubjectBroadSheet(ctx context.Context, schoolID string, isPlatformOwner bool, subjectID, termID, sessionID string) (*dto.SubjectBroadSheetResponse, error) {
	subject, err := s.questionRepo.GetSubjectByID(ctx, subjectID)
	if err != nil {
		return nil, err
	}

	var assignments []models.TeacherSubjectAssignment
	q := s.db.WithContext(ctx).Where("subject_id = ? AND session_id = ? AND is_active = ?", subjectID, sessionID, true)
	if !isPlatformOwner {
		q = q.Where("school_id = ?", schoolID)
	}
	if err := q.Find(&assignments).Error; err != nil {
		return nil, err
	}

	resp := &dto.SubjectBroadSheetResponse{SubjectID: subjectID, SubjectName: subject.Name, TermID: termID, SessionID: sessionID}
	seenClasses := map[string]bool{}
	for _, a := range assignments {
		if seenClasses[a.ClassID] {
			continue
		}
		seenClasses[a.ClassID] = true

		class, err := s.getClassByID(ctx, a.ClassID)
		if err != nil {
			continue
		}
		row := dto.SubjectBroadSheetRow{ClassID: a.ClassID, ClassName: models.GetClassDisplayName(class), TeacherID: a.TeacherID}
		var teacher models.User
		if err := s.db.WithContext(ctx).Select("first_name", "last_name").First(&teacher, "id = ?", a.TeacherID).Error; err == nil {
			row.TeacherName = teacher.FirstName + " " + teacher.LastName
		}

		enrolled, marksEntered, incomplete, avgPct, passRate := s.subjectClassStats(ctx, a.ClassID, subjectID, class.SchoolID, class.ClassLevelID, termID)
		row.Enrolled, row.MarksEntered, row.Incomplete, row.ClassAverage, row.PassRate = enrolled, marksEntered, incomplete, avgPct, passRate

		if sub, err := s.findSubjectResultSubmission(ctx, class.SchoolID, a.ClassID, subjectID, termID, sessionID); err == nil && sub != nil {
			row.SubmissionStatus = sub.Status
		} else {
			row.SubmissionStatus = "draft"
		}
		resp.Rows = append(resp.Rows, row)
	}
	sort.Slice(resp.Rows, func(i, j int) bool { return resp.Rows[i].ClassName < resp.Rows[j].ClassName })
	return resp, nil
}

// subjectClassStats computes one class's completeness/performance
// figures for one subject/term, through the same buildSubjectTermlyResult
// every other view uses - shared by GetSubjectBroadSheet and
// GetSchoolBroadSheet so the two can never disagree with each other.
func (s *ExamService) subjectClassStats(ctx context.Context, classID, subjectID, schoolID, classLevelID, termID string) (enrolled, marksEntered, incomplete int, avgPercent, passRate float64) {
	var students []models.Student
	s.db.WithContext(ctx).Where("class_id = ? AND is_active = ? AND deleted_at IS NULL", classID, true).Find(&students)
	enrolled = len(students)
	if enrolled == 0 {
		return
	}

	allResults, err := s.examRepo.FindResultsByClassAndTerm(ctx, classID, termID)
	if err != nil {
		return
	}
	bySubjectAndStudent := map[string][]models.Result{}
	for _, r := range allResults {
		bySubjectAndStudent[r.StudentID] = append(bySubjectAndStudent[r.StudentID], r)
	}

	var pctSum float64
	var completeCount, passCount int
	for _, st := range students {
		assessment, err := s.buildSubjectTermlyResult(ctx, st.ID, subjectID, schoolID, classLevelID, bySubjectAndStudent[st.ID])
		if err != nil {
			incomplete++
			continue
		}
		if len(assessment.Exams) > 0 {
			marksEntered++
		}
		if !assessment.IsComplete {
			incomplete++
			continue
		}
		completeCount++
		pctSum += assessment.Percentage
		if assessment.Percentage >= 50 {
			passCount++
		}
	}
	if completeCount > 0 {
		avgPercent = pctSum / float64(completeCount)
		passRate = float64(passCount) / float64(completeCount) * 100
	}
	return
}

// GetSchoolBroadSheet is the VP Academics/School General Broad Sheet
// (template C): a school-wide class x subject submission/completeness
// monitor, built from exactly the same TeacherSubjectAssignment and
// subjectClassStats data the other two broad sheets use.
func (s *ExamService) GetSchoolBroadSheet(ctx context.Context, schoolID string, isPlatformOwner bool, targetSchoolID string, termID, sessionID, reportType string) (*dto.SchoolBroadSheetResponse, error) {
	effSchool := examEffectiveSchoolID(targetSchoolID, schoolID, isPlatformOwner)
	if effSchool == "" {
		return nil, errors.New("school context is required")
	}

	var assignments []models.TeacherSubjectAssignment
	if err := s.db.WithContext(ctx).Where("school_id = ? AND session_id = ? AND is_active = ?", effSchool, sessionID, true).Find(&assignments).Error; err != nil {
		return nil, err
	}

	resp := &dto.SchoolBroadSheetResponse{TermID: termID, SessionID: sessionID, ReportType: reportType}
	classesSeen := map[string]bool{}
	totalStudents := 0
	for _, a := range assignments {
		class, err := s.getClassByID(ctx, a.ClassID)
		if err != nil {
			continue
		}
		subject, err := s.questionRepo.GetSubjectByID(ctx, a.SubjectID)
		if err != nil {
			continue
		}
		row := dto.SchoolBroadSheetRow{ClassID: a.ClassID, ClassName: models.GetClassDisplayName(class), SubjectID: a.SubjectID, SubjectName: subject.Name}
		var teacher models.User
		if err := s.db.WithContext(ctx).Select("first_name", "last_name").First(&teacher, "id = ?", a.TeacherID).Error; err == nil {
			row.TeacherName = teacher.FirstName + " " + teacher.LastName
		}

		enrolled, marksEntered, incomplete, avgPct, passRate := s.subjectClassStats(ctx, a.ClassID, a.SubjectID, class.SchoolID, class.ClassLevelID, termID)
		row.Enrolled, row.MarksEntered, row.Incomplete, row.ClassAveragePct, row.PassRatePct = enrolled, marksEntered, incomplete, avgPct, passRate

		if sub, err := s.findSubjectResultSubmission(ctx, class.SchoolID, a.ClassID, a.SubjectID, termID, sessionID); err == nil && sub != nil {
			row.Submitted = sub.Status == "submitted" || sub.Status == "reviewed" || sub.Status == "approved"
			row.Reviewed = sub.Status == "reviewed" || sub.Status == "approved"
			row.ApprovedPublished = sub.Status == "approved"
			if row.Submitted {
				resp.SubmissionsComplete++
			} else {
				resp.PendingIncomplete++
			}
			if sub.Status == "submitted" {
				resp.AwaitingReview++
			}
			if sub.Status == "approved" {
				resp.Approved++
				resp.Published++
			}
		} else {
			resp.PendingIncomplete++
		}

		if !classesSeen[a.ClassID] {
			classesSeen[a.ClassID] = true
			totalStudents += enrolled
		}
		resp.Rows = append(resp.Rows, row)
	}
	resp.ClassesIncluded = len(classesSeen)
	resp.SubjectResultSets = len(resp.Rows)
	resp.TotalStudents = totalStudents
	sort.Slice(resp.Rows, func(i, j int) bool {
		if resp.Rows[i].ClassName != resp.Rows[j].ClassName {
			return resp.Rows[i].ClassName < resp.Rows[j].ClassName
		}
		return resp.Rows[i].SubjectName < resp.Rows[j].SubjectName
	})
	return resp, nil
}
