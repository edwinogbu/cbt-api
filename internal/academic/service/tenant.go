package service

import "errors"

// ErrSchoolScopeRequired is returned when a non-platform-owner caller has
// no school_id in their verified JWT claim (e.g. a legacy account seeded
// with none) and therefore cannot be scoped to any school at all - never
// treated as "unscoped"/full access.
var ErrSchoolScopeRequired = errors.New("no school scope for this account")

// authorizeSchoolOwnership is the shared tenant boundary for every
// academic-module record that carries its own SchoolID column (Class,
// AcademicSession, ClassLevel, ClassArm, Student - Term is the one
// exception, scoped only transitively via its Session, so it has its own
// variant in term_service.go). It mirrors ExamService.authorizeExamTenant
// in internal/cbt/service/exam_service.go:
//   - platform_owner is exempt entirely.
//   - every other caller must have recordSchoolID == ctxSchoolID.
//   - an empty ctxSchoolID on a non-platform-owner caller is never treated
//     as "unscoped" - it simply never matches a real record's school.
//
// ctxSchoolID/isPlatformOwner must come only from
// middleware.GetSchoolID(c)/middleware.IsPlatformOwner(c) - the verified
// JWT claims - never a client-supplied query/body/header value.
func authorizeSchoolOwnership(recordSchoolID, ctxSchoolID string, isPlatformOwner bool) error {
	if isPlatformOwner {
		return nil
	}
	if ctxSchoolID == "" || recordSchoolID != ctxSchoolID {
		return errors.New("not found")
	}
	return nil
}

// effectiveSchoolID resolves the school_id a list/search/stats/create
// operation should actually use: a platform_owner's own (possibly empty,
// meaning "all schools" for a list/filter) client-supplied value is
// honored as-is, since they are deliberately not confined to one tenant;
// every other caller's value is ALWAYS forced to ctxSchoolID, silently
// discarding whatever the client supplied - a client-supplied school_id
// must never be able to independently establish access.
func effectiveSchoolID(clientSuppliedSchoolID, ctxSchoolID string, isPlatformOwner bool) string {
	if isPlatformOwner {
		return clientSuppliedSchoolID
	}
	return ctxSchoolID
}
