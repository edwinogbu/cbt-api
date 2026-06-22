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
    // Auth models
    Register(&User{})
    Register(&UserSession{})
    Register(&OTP{})  // ADD THIS LINE - OTP table was missing!

    
    // Academic models
    Register(&School{})
    Register(&AcademicSession{})
    Register(&Term{})
    Register(&ClassLevel{})
    Register(&ClassArm{})
    Register(&Class{})
      // ADD THESE TWO LINES
    Register(&Student{})
    Register(&ParentStudent{})


	// Subscription models
    Register(&Subscription{})
    Register(&Invoice{})
    Register(&PaymentIntent{})
    Register(&PaymentTransaction{})
    Register(&PaymentEventLog{})
    Register(&WebhookEvent{})
    Register(&SubscriptionHistory{})
    Register(&EmailNotification{})
    Register(&ReminderSchedule{})
   
    
    // CBT models
    Register(&QuestionBank{})
    Register(&QuestionTag{})
    Register(&QuestionBankAttachment{})
    Register(&BulkImportJob{})
    Register(&AIQuestionGenerationJob{})
    Register(&PracticeSession{})
    Register(&ProctoringSession{})
    Register(&ProctoringViolation{})
    Register(&OfflineAnswer{})
    Register(&ExamAssignment{})
    Register(&Subject{})
    Register(&Exam{})
    Register(&Question{})
    Register(&ExamAttempt{})
    Register(&StudentAnswer{})
    Register(&Result{})
}


 
