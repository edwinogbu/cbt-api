package models

import (
	"reflect"
)

var modelRegistry []interface{}

func Register(model interface{}) {
	modelRegistry = append(modelRegistry, model)
}

func GetAllModels() []interface{} {
	return modelRegistry
}

func GetModelNames() []string {
	var names []string
	for _, model := range modelRegistry {
		t := reflect.TypeOf(model)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		names = append(names, t.Name())
	}
	return names
}

func init() {
	// ============================================
	// AUTH MODELS
	// ============================================
	Register(&User{})
	Register(&UserSession{})
	Register(&OTP{})

	// ============================================
	// ACADEMIC MODELS
	// ============================================
	Register(&School{})
	Register(&AcademicSession{})
	Register(&Term{})
	Register(&ClassLevel{})
	Register(&ClassArm{})
	Register(&Class{})
	Register(&Student{})
	Register(&ParentStudent{})

	// ============================================
	// SUBSCRIPTION MODELS
	// ============================================
	Register(&Subscription{})
	Register(&Invoice{})
	Register(&PaymentIntent{})
	Register(&PaymentTransaction{})
	Register(&PaymentEventLog{})
	Register(&WebhookEvent{})
	Register(&SubscriptionHistory{})
	Register(&EmailNotification{})
	Register(&ReminderSchedule{})

	// ============================================
	// CBT MODELS - ALL REGISTERED
	// ============================================
	Register(&QuestionBank{})      // ✅ Questions stored here
	Register(&Tag{})               // ✅ Tags for questions (was QuestionTag)
	Register(&QuestionBankAttachment{})
	Register(&BulkImportJob{})
	Register(&AIQuestionGenerationJob{})
	Register(&PracticeSession{})
	Register(&ProctoringSession{})
	Register(&ProctoringViolation{})
	Register(&OfflineAnswer{})     // ✅ Offline answers storage
	Register(&ExamAssignment{})
	Register(&Subject{})           // ✅ Subjects
	Register(&Exam{})              // ✅ Exams
	Register(&ExamQuestion{})      // ✅ Exam<->question pivot - was never registered, so AUTO_MIGRATE never created this table on a fresh database
	// REMOVED: Register(&Question{}) - This model doesn't exist
	Register(&ExamAttempt{})       // ✅ Exam attempts
	Register(&StudentAnswer{})     // ✅ Student answers
	Register(&Result{})            // ✅ Results

	// ============================================
	// OFFLINE-FIRST SYNC MODELS
	// ============================================
	Register(&SyncOperation{})
	Register(&ExamEventLog{})

	// ============================================
	// SCHOOL NODE SYNC MODELS
	// ============================================
	Register(&SchoolNodeCredential{})
	Register(&NodeSyncState{})
}


// package models

// import (
//     "reflect"
// )

// var modelRegistry []interface{}

// func Register(model interface{}) {
//     modelRegistry = append(modelRegistry, model)
// }

// func GetAllModels() []interface{} {
//     return modelRegistry
// }

// func GetModelNames() []string {
//     var names []string
//     for _, model := range modelRegistry {
//         t := reflect.TypeOf(model)
//         if t.Kind() == reflect.Ptr {
//             t = t.Elem()
//         }
//         names = append(names, t.Name())
//     }
//     return names
// }

// func init() {
//     // Auth models
//     Register(&User{})
//     Register(&UserSession{})
//     Register(&OTP{})  // ADD THIS LINE - OTP table was missing!

    
//     // Academic models
//     Register(&School{})
//     Register(&AcademicSession{})
//     Register(&Term{})
//     Register(&ClassLevel{})
//     Register(&ClassArm{})
//     Register(&Class{})
//       // ADD THESE TWO LINES
//     Register(&Student{})
//     Register(&ParentStudent{})


// 	// Subscription models
//     Register(&Subscription{})
//     Register(&Invoice{})
//     Register(&PaymentIntent{})
//     Register(&PaymentTransaction{})
//     Register(&PaymentEventLog{})
//     Register(&WebhookEvent{})
//     Register(&SubscriptionHistory{})
//     Register(&EmailNotification{})
//     Register(&ReminderSchedule{})
   
    
//     // CBT models
//     // Register(&QuestionBank{})
//     // Register(&QuestionTag{})
//      // CBT models
//     Register(&QuestionBank{})          // ✅ FIXED: was Register(&Question{})
//     Register(&Tag{})  
//     Register(&QuestionBankAttachment{})
//     Register(&BulkImportJob{})
//     Register(&AIQuestionGenerationJob{})
//     Register(&PracticeSession{})
//     Register(&ProctoringSession{})
//     Register(&ProctoringViolation{})
//     Register(&OfflineAnswer{})
//     Register(&ExamAssignment{})
//     Register(&Subject{})
//     Register(&Exam{})
//     Register(&Question{})
//     Register(&ExamAttempt{})
//     Register(&StudentAnswer{})
//     Register(&Result{})
// }


 
