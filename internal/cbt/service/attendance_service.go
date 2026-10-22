package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/models"

	"gorm.io/gorm/clause"
)

// validAttendanceStatuses are the only marks MarkClassAttendance
// accepts - a short, service-validated set rather than a DB enum,
// matching this codebase's established preference (PromotionDecision.
// Decision, ReportCardApproval.Status) elsewhere.
var validAttendanceStatuses = map[string]bool{
	"present": true, "absent": true, "late": true, "excused": true,
}

const attendanceDateLayout = "2006-01-02"

func parseAttendanceDate(s string) (time.Time, error) {
	d, err := time.Parse(attendanceDateLayout, s)
	if err != nil {
		return time.Time{}, errors.New("date must be in YYYY-MM-DD format")
	}
	return d, nil
}

// MarkClassAttendance records one class's attendance register for one
// date - never all-or-nothing (see dto.MarkClassAttendanceResponse):
// each student's mark is validated and upserted independently, so one
// bad entry (an unrecognized status, a student who has since left the
// class) never blocks the rest of a teacher's roll call from being
// saved. Re-submitting the same date for the same student updates their
// existing mark (ON CONFLICT on the (student_id, term_id, date) unique
// index) rather than creating a duplicate - a teacher correcting a
// mistake just marks the day again.
func (s *ExamService) MarkClassAttendance(ctx context.Context, schoolID, actorID string, isPlatformOwner bool, req *dto.MarkClassAttendanceRequest) (*dto.MarkClassAttendanceResponse, error) {
	if err := s.authorizeClassTenant(ctx, req.ClassID, schoolID, isPlatformOwner); err != nil {
		return nil, err
	}
	date, err := parseAttendanceDate(req.Date)
	if err != nil {
		return nil, err
	}
	// Attendance records a day that already happened - a future date
	// would let "days school open" (see attendanceSummary) count a day
	// nobody has actually held classes on yet, inflating every student's
	// denominator before the fact.
	if date.After(time.Now()) {
		return nil, errors.New("cannot mark attendance for a future date")
	}

	students, err := s.examRepo.GetStudentsByClass(ctx, req.ClassID)
	if err != nil {
		return nil, err
	}
	inClass := make(map[string]bool, len(students))
	for _, st := range students {
		inClass[st.ID] = true
	}

	resp := &dto.MarkClassAttendanceResponse{MarkedStudentIDs: []string{}, Failed: []dto.AttendanceMarkFailure{}}
	for _, r := range req.Records {
		if !validAttendanceStatuses[r.Status] {
			resp.Failed = append(resp.Failed, dto.AttendanceMarkFailure{StudentID: r.StudentID, Error: "status must be one of present, absent, late, excused"})
			continue
		}
		if !inClass[r.StudentID] {
			resp.Failed = append(resp.Failed, dto.AttendanceMarkFailure{StudentID: r.StudentID, Error: "student is not an active member of this class"})
			continue
		}

		record := models.AttendanceRecord{
			ID: models.GenerateID(), SchoolID: schoolID, StudentID: r.StudentID, ClassID: req.ClassID,
			TermID: req.TermID, SessionID: req.SessionID, Date: date,
			Status: r.Status, MarkedBy: actorID, Notes: r.Notes,
		}
		if err := s.db.WithContext(ctx).
			Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "student_id"}, {Name: "term_id"}, {Name: "date"}},
				// synced_to_cloud_at is reset to NULL (record's own zero
				// value) on every update, not just set on first insert -
				// a correction to an already-synced mark must re-enter
				// the node's outbox (WHERE synced_to_cloud_at IS NULL),
				// never stay "synced" under its stale pre-correction
				// content.
				DoUpdates: clause.AssignmentColumns([]string{"status", "notes", "marked_by", "class_id", "session_id", "synced_to_cloud_at", "updated_at"}),
			}).
			Create(&record).Error; err != nil {
			resp.Failed = append(resp.Failed, dto.AttendanceMarkFailure{StudentID: r.StudentID, Error: "write failed"})
			continue
		}
		resp.MarkedStudentIDs = append(resp.MarkedStudentIDs, r.StudentID)
	}
	return resp, nil
}

// GetClassAttendanceForDate is the register a teacher sees when opening
// roll call for one date - every active student in the class, with
// whatever has already been marked (an empty Status for a student with
// no mark yet, never inferred as present or absent).
func (s *ExamService) GetClassAttendanceForDate(ctx context.Context, schoolID string, isPlatformOwner bool, classID, dateStr string) (*dto.ClassAttendanceResponse, error) {
	if err := s.authorizeClassTenant(ctx, classID, schoolID, isPlatformOwner); err != nil {
		return nil, err
	}
	date, err := parseAttendanceDate(dateStr)
	if err != nil {
		return nil, err
	}

	className, _, _ := s.resolveClassWithTeacher(ctx, classID)

	var students []models.Student
	if err := s.db.WithContext(ctx).Preload("User").
		Where("class_id = ? AND is_active = ? AND deleted_at IS NULL", classID, true).
		Find(&students).Error; err != nil {
		return nil, err
	}
	sort.Slice(students, func(i, j int) bool { return students[i].AdmissionNo < students[j].AdmissionNo })

	var records []models.AttendanceRecord
	if err := s.db.WithContext(ctx).Where("class_id = ? AND date = ?", classID, date).Find(&records).Error; err != nil {
		return nil, err
	}
	byStudent := make(map[string]models.AttendanceRecord, len(records))
	for _, r := range records {
		byStudent[r.StudentID] = r
	}

	rows := make([]dto.ClassAttendanceRow, 0, len(students))
	for _, st := range students {
		row := dto.ClassAttendanceRow{
			StudentID: st.ID, StudentName: strings.TrimSpace(st.User.FirstName + " " + st.User.LastName),
			AdmissionNumber: st.AdmissionNo,
		}
		if r, ok := byStudent[st.ID]; ok {
			row.Status = r.Status
			row.Notes = r.Notes
		}
		rows = append(rows, row)
	}

	return &dto.ClassAttendanceResponse{ClassID: classID, ClassName: className, Date: dateStr, Rows: rows}, nil
}

// attendanceSummary computes one student's (daysPresent, daysAbsent,
// daysSchoolOpen) for one class+term from models.AttendanceRecord.
// "Days school open" is the count of distinct dates ANY attendance was
// recorded for this class+term (see models.AttendanceRecord's own doc
// comment for why there is no separate configured calendar) - so for a
// Mid-Term report, built partway through a term, this naturally reflects
// only the days that have actually happened so far, with no separate
// "as of today" filtering needed. "Present" counts present+late (the
// student did attend, whether on time or not); "absent" counts
// absent+excused (the student did not attend, whatever the reason) -
// the two-bucket split report cards already render.
func (s *ExamService) attendanceSummary(ctx context.Context, studentID, classID, termID string) (daysPresent, daysAbsent, daysSchoolOpen int, err error) {
	var openDays int64
	if err = s.db.WithContext(ctx).Model(&models.AttendanceRecord{}).
		Where("class_id = ? AND term_id = ?", classID, termID).
		Distinct("date").Count(&openDays).Error; err != nil {
		return 0, 0, 0, err
	}

	var present, absent int64
	if err = s.db.WithContext(ctx).Model(&models.AttendanceRecord{}).
		Where("student_id = ? AND term_id = ? AND status IN ?", studentID, termID, []string{"present", "late"}).
		Count(&present).Error; err != nil {
		return 0, 0, 0, err
	}
	if err = s.db.WithContext(ctx).Model(&models.AttendanceRecord{}).
		Where("student_id = ? AND term_id = ? AND status IN ?", studentID, termID, []string{"absent", "excused"}).
		Count(&absent).Error; err != nil {
		return 0, 0, 0, err
	}
	return int(present), int(absent), int(openDays), nil
}

// GetStudentAttendanceSummary is the standalone admin/teacher lookup of
// one student's attendance figures - the same computation report cards
// embed, exposed directly (e.g. for a parent-facing summary or an admin
// spot-check without pulling a whole report card).
func (s *ExamService) GetStudentAttendanceSummary(ctx context.Context, schoolID string, isPlatformOwner bool, studentID, termID, sessionID string) (*dto.StudentAttendanceSummaryResponse, error) {
	student, err := s.examRepo.GetStudentByID(ctx, studentID)
	if err != nil {
		return nil, errors.New("student not found")
	}
	if !isPlatformOwner && student.SchoolID != schoolID {
		return nil, errors.New("student not found")
	}
	present, absent, open, err := s.attendanceSummary(ctx, studentID, student.ClassID, termID)
	if err != nil {
		return nil, err
	}
	return &dto.StudentAttendanceSummaryResponse{
		StudentID: studentID, ClassID: student.ClassID, TermID: termID, SessionID: sessionID,
		DaysPresent: present, DaysAbsent: absent, DaysSchoolOpen: open,
	}, nil
}

// GetMyAttendanceSummary is the authenticated student's own view of
// GetStudentAttendanceSummary - resolves the student from the JWT
// subject rather than trusting a client-supplied studentID.
func (s *ExamService) GetMyAttendanceSummary(ctx context.Context, userID, termID, sessionID string) (*dto.StudentAttendanceSummaryResponse, error) {
	student, err := s.examRepo.GetStudentByUserID(ctx, userID)
	if err != nil {
		return nil, errors.New("student not found")
	}
	present, absent, open, err := s.attendanceSummary(ctx, student.ID, student.ClassID, termID)
	if err != nil {
		return nil, err
	}
	return &dto.StudentAttendanceSummaryResponse{
		StudentID: student.ID, ClassID: student.ClassID, TermID: termID, SessionID: sessionID,
		DaysPresent: present, DaysAbsent: absent, DaysSchoolOpen: open,
	}, nil
}
