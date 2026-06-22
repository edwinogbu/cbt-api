package handler

import (
    "cbt-api/internal/cbt/dto"
    "cbt-api/internal/cbt/service"
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

type ExamHandler struct {
    examService *service.ExamService
}

func NewExamHandler(s *service.ExamService) *ExamHandler {
    return &ExamHandler{examService: s}
}

// StartExam godoc
// @Summary      Start an exam attempt
// @Description  Create a new exam attempt for a student. Returns exam details and questions.
// @Tags         Exams
// @Accept       json
// @Produce      json
// @Param        request body dto.StartExamRequest true "Start exam request"
// @Success      200  {object}  map[string]interface{}  "data contains StartExamResponse"
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /exams/start [post]
func (h *ExamHandler) StartExam(c *gin.Context) {
    var req dto.StartExamRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    ctx := c.Request.Context()
    resp, err := h.examService.StartExam(ctx, &req)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": resp})
}

// SaveAnswer godoc
// @Summary      Save a single answer
// @Description  Save or update the student's answer for a question during an active exam attempt.
// @Tags         Exams
// @Accept       json
// @Produce      json
// @Param        request body dto.SaveAnswerRequest true "Save answer request"
// @Success      200  {object}  map[string]interface{}  "data contains SaveAnswerResponse"
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /exams/save-answer [post]
func (h *ExamHandler) SaveAnswer(c *gin.Context) {
    var req dto.SaveAnswerRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    resp, err := h.examService.SaveAnswer(c.Request.Context(), &req)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": resp})
}

// SubmitExam godoc
// @Summary      Submit an exam
// @Description  Finalise the exam attempt, calculate score and store the result.
// @Tags         Exams
// @Accept       json
// @Produce      json
// @Param        request body dto.SubmitExamRequest true "Submit exam request"
// @Success      200  {object}  map[string]interface{}  "data contains SubmitExamResponse"
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /exams/submit [post]
func (h *ExamHandler) SubmitExam(c *gin.Context) {
    var req dto.SubmitExamRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    resp, err := h.examService.SubmitExam(c.Request.Context(), &req)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": resp})
}

// StartPractice godoc
// @Summary      Start a practice session
// @Description  Generate random questions for a student to practice on a given subject.
// @Tags         Exams
// @Produce      json
// @Param        studentId path string true "Student UUID"
// @Param        subjectId path string true "Subject UUID"
// @Param        count query int false "Number of questions (default 10)"
// @Success      200  {object}  map[string]interface{}  "data contains PracticeSessionResponse"
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /exams/practice/{studentId}/{subjectId} [get]
func (h *ExamHandler) StartPractice(c *gin.Context) {
    studentID := c.Param("studentId")
    subjectID := c.Param("subjectId")
    count, _ := strconv.Atoi(c.DefaultQuery("count", "10"))

    sid, err := uuid.Parse(studentID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student ID"})
        return
    }
    subj, err := uuid.Parse(subjectID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subject ID"})
        return
    }
    resp, err := h.examService.StartPractice(c.Request.Context(), sid, subj, count)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": resp})
}



// package handler

// import (
//     "cbt-api/internal/cbt/dto"
//     "cbt-api/internal/cbt/service"
//     "net/http"
//     "strconv"

//     "github.com/gin-gonic/gin"
//     "github.com/google/uuid"
// )

// type ExamHandler struct {
//     examService *service.ExamService
// }

// func NewExamHandler(s *service.ExamService) *ExamHandler {
//     return &ExamHandler{examService: s}
// }

// func (h *ExamHandler) StartExam(c *gin.Context) {
//     var req dto.StartExamRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
//     ctx := c.Request.Context()
//     resp, err := h.examService.StartExam(ctx, &req)
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
//     c.JSON(http.StatusOK, gin.H{"data": resp})
// }

// func (h *ExamHandler) SaveAnswer(c *gin.Context) {
//     var req dto.SaveAnswerRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
//     resp, err := h.examService.SaveAnswer(c.Request.Context(), &req)
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
//     c.JSON(http.StatusOK, gin.H{"data": resp})
// }

// func (h *ExamHandler) SubmitExam(c *gin.Context) {
//     var req dto.SubmitExamRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
//     resp, err := h.examService.SubmitExam(c.Request.Context(), &req)
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
//     c.JSON(http.StatusOK, gin.H{"data": resp})
// }

// func (h *ExamHandler) StartPractice(c *gin.Context) {
//     studentID := c.Param("studentId")
//     subjectID := c.Param("subjectId")
//     count, _ := strconv.Atoi(c.DefaultQuery("count", "10"))

//     sid, err := uuid.Parse(studentID)
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student ID"})
//         return
//     }
//     subj, err := uuid.Parse(subjectID)
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subject ID"})
//         return
//     }
//     resp, err := h.examService.StartPractice(c.Request.Context(), sid, subj, count)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//         return
//     }
//     c.JSON(http.StatusOK, gin.H{"data": resp})
// }