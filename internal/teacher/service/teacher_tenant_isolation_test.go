package service

import (
	"testing"

	"cbt-api/internal/academic/repository"
	"cbt-api/internal/teacher/dto"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ---- createStudentInternal: req.SchoolID was client-supplied and used
// directly for the new student's stored SchoolID, with zero check that
// it matched the already-authorized class's own school. A teacher
// (legitimately assigned to their own class, via the class.TeacherID
// check right above) could submit any school_id in the request body
// and get a student record mis-tagged as belonging to a DIFFERENT
// school than the class it's actually enrolled in - exactly the kind
// of mismatch that would let that other school's admin later see,
// manage, or hard-delete the student via this codebase's other
// school_id-scoped checks (ListAllStudents, HardDeleteStudent), or
// make the student vanish from the real school's own admin view. Now
// class.SchoolID (resolved server-side) is used instead, everywhere.

func setupTeacherTenantTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	ddl := []string{
		`CREATE TABLE users (
			id TEXT PRIMARY KEY, username TEXT, email TEXT, password TEXT,
			first_name TEXT, last_name TEXT, phone_number TEXT, role TEXT,
			status TEXT, email_verified INTEGER DEFAULT 0, is_active INTEGER DEFAULT 1,
			two_factor_secret TEXT, two_factor_enabled INTEGER DEFAULT 0, last_login_at DATETIME,
			school_id TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME,
			plain_password_encrypted TEXT
		)`,
		`CREATE TABLE classes (
			id TEXT PRIMARY KEY, school_id TEXT, session_id TEXT, teacher_id TEXT,
			class_level_id TEXT, class_arm_id TEXT, class_code TEXT, room_number TEXT,
			is_active INTEGER DEFAULT 1, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE students (
			id TEXT PRIMARY KEY, user_id TEXT, school_id TEXT, class_id TEXT, admission_no TEXT,
			date_of_birth DATETIME, gender TEXT, address TEXT,
			guardian_name TEXT, guardian_phone TEXT, guardian_email TEXT,
			is_active INTEGER DEFAULT 1, status TEXT DEFAULT 'active',
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE schools (id TEXT PRIMARY KEY, name TEXT, deleted_at DATETIME)`,
	}
	for _, stmt := range ddl {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("failed to create test schema (%s): %v", stmt, err)
		}
	}
	return db
}

func newTenantTestTeacherService(db *gorm.DB) *TeacherService {
	logger := zap.NewNop()
	return NewTeacherService(
		repository.NewUserRepository(db),
		repository.NewStudentRepository(db),
		repository.NewClassRepository(db),
		repository.NewSchoolRepository(db),
		db,
		logger,
	)
}

func seedTeacherClass(t *testing.T, db *gorm.DB, id, schoolID, teacherID string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO classes (id, school_id, teacher_id) VALUES (?, ?, ?)", id, schoolID, teacherID,
	).Error; err != nil {
		t.Fatalf("failed to seed class: %v", err)
	}
}

func seedTeacherSchool(t *testing.T, db *gorm.DB, id, name string) {
	t.Helper()
	if err := db.Exec("INSERT INTO schools (id, name) VALUES (?, ?)", id, name).Error; err != nil {
		t.Fatalf("failed to seed school: %v", err)
	}
}

func TestCreateStudent_ClientSuppliedSchoolID_Ignored(t *testing.T) {
	db := setupTeacherTenantTestDB(t)
	s := newTenantTestTeacherService(db)
	seedTeacherSchool(t, db, "school-A", "School A")
	seedTeacherSchool(t, db, "school-B", "School B")
	seedTeacherClass(t, db, "class-a", "school-A", "teacher-a")

	// teacher-a is legitimately assigned to class-a (in school-A), but
	// submits school_id=school-B in the request body.
	resp, err := s.CreateStudent(&dto.CreateStudentByTeacherRequest{
		SchoolID:  "school-B",
		ClassID:   "class-a",
		FirstName: "New",
		LastName:  "Student",
	}, "teacher-a")
	if err != nil {
		t.Fatalf("expected the teacher to create a student in their own authorized class, got error: %v", err)
	}

	var storedSchoolID string
	db.Table("students").Select("school_id").Where("id = ?", resp.StudentID).Scan(&storedSchoolID)
	if storedSchoolID != "school-A" {
		t.Fatalf("expected the student to be tagged with the class's real school (school-A), got %q - the client-supplied school_id leaked through", storedSchoolID)
	}
}

func TestCreateStudent_CrossTenantClass_Rejected(t *testing.T) {
	db := setupTeacherTenantTestDB(t)
	s := newTenantTestTeacherService(db)
	seedTeacherSchool(t, db, "school-B", "School B")
	seedTeacherClass(t, db, "class-b", "school-B", "teacher-b")

	// teacher-a (not assigned to class-b at all) tries to add a student
	// to School B's class.
	if _, err := s.CreateStudent(&dto.CreateStudentByTeacherRequest{
		SchoolID:  "school-B",
		ClassID:   "class-b",
		FirstName: "New",
		LastName:  "Student",
	}, "teacher-a"); err == nil {
		t.Fatal("expected a teacher not assigned to class-b to be rejected, got no error")
	}

	var count int64
	db.Table("students").Where("class_id = ?", "class-b").Count(&count)
	if count != 0 {
		t.Fatal("a student was created by a cross-tenant CreateStudent call that should have been rejected")
	}
}
