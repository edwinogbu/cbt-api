package dto

// ============================================
// ATTENDANCE TRACKING
// ============================================
//
// Feeds the Attendance section of all three report-card types
// (TermlyReportCardResponse.Attendance / MidTermReportCardResponse.
// Attendance) - see models.AttendanceRecord's own doc comment for how
// "days school open" is derived (no separate calendar table: a date
// counts as a school day purely by having a register taken for it).

// AttendanceMarkEntry is one student's mark within a MarkClassAttendanceRequest.
type AttendanceMarkEntry struct {
	StudentID string `json:"student_id" binding:"required"`
	// Status is one of "present", "absent", "late", "excused".
	Status string `json:"status" binding:"required"`
	Notes  string `json:"notes,omitempty"`
}

// MarkClassAttendanceRequest records one class's attendance register for
// one calendar date in a single call - the realistic daily workflow (a
// teacher marks their whole class at once, not one student at a time).
// Re-submitting the same date updates each student's existing mark
// rather than creating a duplicate.
type MarkClassAttendanceRequest struct {
	ClassID   string `json:"class_id" binding:"required"`
	TermID    string `json:"term_id" binding:"required"`
	SessionID string `json:"session_id" binding:"required"`
	// Date is "YYYY-MM-DD" - a calendar date, not a timestamp; attendance
	// has no meaningful time-of-day component.
	Date    string                `json:"date" binding:"required"`
	Records []AttendanceMarkEntry `json:"records" binding:"required"`
}

// AttendanceMarkFailure is one student's reason for not being recorded
// in a MarkClassAttendanceRequest - never all-or-nothing, the same
// partial-success convention as BatchApproveReportCards.
type AttendanceMarkFailure struct {
	StudentID string `json:"student_id"`
	Error     string `json:"error"`
}

type MarkClassAttendanceResponse struct {
	MarkedStudentIDs []string                `json:"marked_student_ids"`
	Failed           []AttendanceMarkFailure `json:"failed"`
}

// ClassAttendanceRow is one student's current mark (if any) for the
// requested date - Status is empty for a student with no mark yet
// today, never inferred as "present" or "absent" by default.
type ClassAttendanceRow struct {
	StudentID       string `json:"student_id"`
	StudentName     string `json:"student_name"`
	AdmissionNumber string `json:"admission_number"`
	Status          string `json:"status,omitempty"`
	Notes           string `json:"notes,omitempty"`
}

// ClassAttendanceResponse is a class's attendance register for one
// date - what a teacher sees when opening today's (or any past) roll
// call, pre-filled with whatever has already been marked.
type ClassAttendanceResponse struct {
	ClassID   string               `json:"class_id"`
	ClassName string               `json:"class_name"`
	Date      string               `json:"date"`
	Rows      []ClassAttendanceRow `json:"rows"`
}

// StudentAttendanceSummaryResponse is one student's attendance figures
// for one term - the same (days_present, days_absent, days_school_open)
// shape report cards already render via AttendanceSummary/
// MidTermAttendance, exposed directly for admin/teacher lookup and the
// student's own "My Attendance" view.
type StudentAttendanceSummaryResponse struct {
	StudentID      string `json:"student_id"`
	ClassID        string `json:"class_id"`
	TermID         string `json:"term_id"`
	SessionID      string `json:"session_id"`
	DaysPresent    int    `json:"days_present"`
	DaysAbsent     int    `json:"days_absent"`
	DaysSchoolOpen int    `json:"days_school_open"`
}
