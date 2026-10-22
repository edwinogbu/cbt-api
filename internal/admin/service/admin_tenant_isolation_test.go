package service

import (
	"testing"

	"cbt-api/internal/academic/repository"
	"cbt-api/internal/admin/dto"
	"cbt-api/internal/models"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---- AdminService: every single method in this file had no school
// check of any kind before this fix. The route group backing it
// (middleware.AdminOnly(), an exact match on the school-scoped "admin"
// role - not platform_owner) is reachable by any single school's plain
// admin, yet ListAllStudents/ListAllClasses/ListAllTeachers/
// ListUsersByRole returned every school's full roster platform-wide
// (names, admission numbers, emails, usernames), AssignTeacher/
// UnassignTeacher let any school's admin reassign another school's
// class or teacher, and HardDeleteStudent let any school's admin
// permanently (Unscoped) destroy any other school's student record.

func setupAdminTenantTestDB(t *testing.T) *gorm.DB {
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
			id TEXT PRIMARY KEY, school_id TEXT, session_id TEXT, teacher_id TEXT, class_level_id TEXT, class_arm_id TEXT,
			class_code TEXT, room_number TEXT, is_active INTEGER DEFAULT 1,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE class_levels (id TEXT PRIMARY KEY, name TEXT, deleted_at DATETIME)`,
		`CREATE TABLE class_arms (id TEXT PRIMARY KEY, name TEXT, deleted_at DATETIME)`,
		`CREATE TABLE schools (id TEXT PRIMARY KEY, name TEXT, deleted_at DATETIME)`,
		`CREATE TABLE students (
			id TEXT PRIMARY KEY, user_id TEXT, school_id TEXT, class_id TEXT, admission_no TEXT,
			is_active INTEGER DEFAULT 1, status TEXT DEFAULT 'active',
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
	}
	for _, stmt := range ddl {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("failed to create test schema (%s): %v", stmt, err)
		}
	}
	return db
}

func newTenantTestAdminService(db *gorm.DB) *AdminService {
	return NewAdminService(
		repository.NewUserRepository(db),
		repository.NewClassRepository(db),
		repository.NewStudentRepository(db),
		db,
	)
}

func seedAdminUser(t *testing.T, db *gorm.DB, id, role, schoolID string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO users (id, username, role, status, school_id) VALUES (?, ?, ?, 'active', ?)",
		id, id, role, schoolID,
	).Error; err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
}

func seedAdminClass(t *testing.T, db *gorm.DB, id, schoolID string) {
	t.Helper()
	if err := db.Exec("INSERT INTO classes (id, school_id) VALUES (?, ?)", id, schoolID).Error; err != nil {
		t.Fatalf("failed to seed class: %v", err)
	}
}

func seedAdminStudent(t *testing.T, db *gorm.DB, id, schoolID string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO students (id, user_id, school_id, admission_no) VALUES (?, ?, ?, ?)",
		id, "user-"+id, schoolID, "ADM-"+id,
	).Error; err != nil {
		t.Fatalf("failed to seed student: %v", err)
	}
}

func TestAssignTeacher_CrossTenantClass_Rejected(t *testing.T) {
	db := setupAdminTenantTestDB(t)
	s := newTenantTestAdminService(db)
	seedAdminClass(t, db, "class-b", "school-B")
	seedAdminUser(t, db, "teacher-a", string(models.RoleTeacher), "school-A")

	err := s.AssignTeacher(&dto.AssignTeacherRequest{ClassID: "class-b", TeacherID: "teacher-a"}, "school-A", false)
	if err == nil {
		t.Fatal("expected School A's admin to be rejected assigning a teacher to School B's class, got no error")
	}

	var teacherID *string
	db.Table("classes").Select("teacher_id").Where("id = ?", "class-b").Scan(&teacherID)
	if teacherID != nil {
		t.Fatal("School B's class was mutated by a cross-tenant AssignTeacher call that should have been rejected")
	}
}

func TestAssignTeacher_CrossTenantTeacher_Rejected(t *testing.T) {
	db := setupAdminTenantTestDB(t)
	s := newTenantTestAdminService(db)
	seedAdminClass(t, db, "class-a", "school-A")
	seedAdminUser(t, db, "teacher-b", string(models.RoleTeacher), "school-B")

	err := s.AssignTeacher(&dto.AssignTeacherRequest{ClassID: "class-a", TeacherID: "teacher-b"}, "school-A", false)
	if err == nil {
		t.Fatal("expected School A's admin to be rejected assigning School B's teacher to School A's own class, got no error")
	}
}

func TestAssignTeacher_SameTenant_Allowed(t *testing.T) {
	db := setupAdminTenantTestDB(t)
	s := newTenantTestAdminService(db)
	seedAdminClass(t, db, "class-a", "school-A")
	seedAdminUser(t, db, "teacher-a", string(models.RoleTeacher), "school-A")

	if err := s.AssignTeacher(&dto.AssignTeacherRequest{ClassID: "class-a", TeacherID: "teacher-a"}, "school-A", false); err != nil {
		t.Fatalf("expected School A's admin to assign its own teacher to its own class, got error: %v", err)
	}
}

func TestAssignTeacher_PlatformOwner_Bypasses(t *testing.T) {
	db := setupAdminTenantTestDB(t)
	s := newTenantTestAdminService(db)
	seedAdminClass(t, db, "class-b", "school-B")
	seedAdminUser(t, db, "teacher-b", string(models.RoleTeacher), "school-B")

	if err := s.AssignTeacher(&dto.AssignTeacherRequest{ClassID: "class-b", TeacherID: "teacher-b"}, "", true); err != nil {
		t.Fatalf("expected platform_owner to assign any school's teacher to any school's class, got error: %v", err)
	}
}

func TestUnassignTeacher_CrossTenant_Rejected(t *testing.T) {
	db := setupAdminTenantTestDB(t)
	s := newTenantTestAdminService(db)
	seedAdminClass(t, db, "class-b", "school-B")
	db.Exec("UPDATE classes SET teacher_id = ? WHERE id = ?", "teacher-b", "class-b")

	if err := s.UnassignTeacher("class-b", "school-A", false); err == nil {
		t.Fatal("expected School A's admin to be rejected unassigning School B's class's teacher, got no error")
	}

	var teacherID string
	db.Table("classes").Select("teacher_id").Where("id = ?", "class-b").Scan(&teacherID)
	if teacherID != "teacher-b" {
		t.Fatal("School B's class was mutated by a cross-tenant UnassignTeacher call that should have been rejected")
	}
}

func TestListUsersByRole_CrossTenant_Excluded(t *testing.T) {
	db := setupAdminTenantTestDB(t)
	s := newTenantTestAdminService(db)
	seedAdminUser(t, db, "admin-a", string(models.RoleAdmin), "school-A")
	seedAdminUser(t, db, "admin-b", string(models.RoleAdmin), "school-B")

	users, total, err := s.ListUsersByRole(dto.ListUsersQuery{Page: 1, Limit: 20}, "school-A", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected School A's admin to see exactly 1 user (their own school's), got %d", total)
	}
	for _, u := range users {
		if u.ID == "admin-b" {
			t.Fatal("School B's user leaked into School A's unfiltered user list")
		}
	}
}

func TestListUsersByRole_PlatformOwner_SeesAllSchools(t *testing.T) {
	db := setupAdminTenantTestDB(t)
	s := newTenantTestAdminService(db)
	seedAdminUser(t, db, "admin-a", string(models.RoleAdmin), "school-A")
	seedAdminUser(t, db, "admin-b", string(models.RoleAdmin), "school-B")

	_, total, err := s.ListUsersByRole(dto.ListUsersQuery{Page: 1, Limit: 20}, "", true)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected platform_owner to see both schools' users, got %d", total)
	}
}

func TestListAllStudents_CrossTenant_Excluded(t *testing.T) {
	db := setupAdminTenantTestDB(t)
	s := newTenantTestAdminService(db)
	seedAdminStudent(t, db, "student-a", "school-A")
	seedAdminStudent(t, db, "student-b", "school-B")

	students, total, err := s.ListAllStudents(1, 20, "", "school-A", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected School A's admin to see exactly 1 student (their own school's), got %d", total)
	}
	for _, st := range students {
		if st.ID == "student-b" {
			t.Fatal("School B's student leaked into School A's unfiltered student list")
		}
	}
}

func TestListAllStudents_PlatformOwner_SeesAllSchools(t *testing.T) {
	db := setupAdminTenantTestDB(t)
	s := newTenantTestAdminService(db)
	seedAdminStudent(t, db, "student-a", "school-A")
	seedAdminStudent(t, db, "student-b", "school-B")

	_, total, err := s.ListAllStudents(1, 20, "", "", true)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected platform_owner to see both schools' students, got %d", total)
	}
}

func TestHardDeleteStudent_CrossTenant_Rejected(t *testing.T) {
	db := setupAdminTenantTestDB(t)
	s := newTenantTestAdminService(db)
	seedAdminStudent(t, db, "student-b", "school-B")

	if err := s.HardDeleteStudent("student-b", "school-A", false); err == nil {
		t.Fatal("expected School A's admin to be rejected hard-deleting School B's student, got no error")
	}

	var count int64
	db.Table("students").Where("id = ?", "student-b").Count(&count)
	if count != 1 {
		t.Fatal("School B's student was permanently deleted by a cross-tenant HardDeleteStudent call that should have been rejected")
	}
}

func TestHardDeleteStudent_SameTenant_Allowed(t *testing.T) {
	db := setupAdminTenantTestDB(t)
	s := newTenantTestAdminService(db)
	seedAdminStudent(t, db, "student-a", "school-A")

	if err := s.HardDeleteStudent("student-a", "school-A", false); err != nil {
		t.Fatalf("expected School A's admin to hard-delete its own school's student, got error: %v", err)
	}

	var count int64
	db.Table("students").Where("id = ?", "student-a").Count(&count)
	if count != 0 {
		t.Fatal("expected the student to actually be deleted")
	}
}

func TestListAllClasses_CrossTenant_Excluded(t *testing.T) {
	db := setupAdminTenantTestDB(t)
	s := newTenantTestAdminService(db)
	seedAdminClass(t, db, "class-a", "school-A")
	seedAdminClass(t, db, "class-b", "school-B")

	classes, total, err := s.ListAllClasses(1, 20, "", "school-A", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected School A's admin to see exactly 1 class (their own school's), got %d", total)
	}
	for _, cl := range classes {
		if cl.ID == "class-b" {
			t.Fatal("School B's class leaked into School A's unfiltered class list")
		}
	}
}

func TestListAllTeachers_CrossTenant_Excluded(t *testing.T) {
	db := setupAdminTenantTestDB(t)
	s := newTenantTestAdminService(db)
	seedAdminUser(t, db, "teacher-a", string(models.RoleTeacher), "school-A")
	seedAdminUser(t, db, "teacher-b", string(models.RoleTeacher), "school-B")

	teachers, total, err := s.ListAllTeachers(1, 20, "", "school-A", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected School A's admin to see exactly 1 teacher (their own school's), got %d", total)
	}
	for _, te := range teachers {
		if te.ID == "teacher-b" {
			t.Fatal("School B's teacher leaked into School A's unfiltered teacher list")
		}
	}
}
