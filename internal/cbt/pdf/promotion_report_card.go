package pdf

import (
	"bytes"
	"fmt"
	"time"

	"cbt-api/internal/cbt/dto"
)

// defaultFinalDecision is what a Promotion Examination Report Card
// shows until an authorized human records a real decision through an
// approval workflow - GeneratePromotionExaminationReport falls back to
// this whenever FinalDecision is empty, so no caller can leave the
// field blank and have it silently read as a decision either way.
const defaultFinalDecision = "Pending authorized approval"

// GeneratePromotionExaminationReport renders a Promotion Examination
// Report Card. SystemRecommendation is rendered clearly labeled as a
// recommendation, directly beside an explicit disclaimer, and
// FinalDecision is rendered as its own, visually distinct field that
// defaults to "Pending authorized approval" - this function never
// derives one from the other. A report card is a record of a
// recommendation and (once made) a decision, not the mechanism that
// makes the decision.
func GeneratePromotionExaminationReport(data *dto.PromotionReportCardResponse) ([]byte, error) {
	accent := resolveAccent(data.School.PrimaryColor)
	pdfDoc := newReportPDF()

	drawSchoolHeader(pdfDoc, data.School, "")
	drawTitleBar(pdfDoc, "PROMOTION EXAMINATION REPORT CARD", data.Session.Name, accent)

	drawKeyValueGrid(pdfDoc, [][2]string{
		{"STUDENT NAME", data.Student.Name}, {"ADMISSION NO.", data.Student.AdmissionNumber},
		{"CLASS / ARM", data.Class.Name}, {"ACADEMIC SESSION", data.Session.Name},
		{"TERM / ASSESSMENT", "Promotion Examination"}, {"CLASS TEACHER", orDash(data.Class.ClassTeacher)},
	}, 2)

	componentColumns := collectComponentColumns(data.Subjects)
	drawSectionHeading(pdfDoc, "PROMOTION EXAMINATION RESULTS", "", accent)
	drawSubjectTable(pdfDoc, data.Subjects, componentColumns, accent)

	drawStatBoxes(pdfDoc, []statBox{
		{Label: "TOTAL OBTAINED", Value: fmt.Sprintf("%d", data.Summary.TotalScore), Caption: fmt.Sprintf("out of %d", data.Summary.TotalMarks)},
		{Label: "OVERALL AVERAGE", Value: fmt.Sprintf("%.1f%%", data.Summary.OverallPercentage), Caption: fmt.Sprintf("Grade %s", data.Summary.OverallGrade), AccentValue: true},
		{Label: "SUBJECTS PASSED", Value: fmt.Sprintf("%d / %d", data.Summary.NumberPassed, data.Summary.TotalSubjects), Caption: passFailCaption(data.Summary)},
		{Label: "CLASS POSITION", Value: classPositionValue(data.Summary), Caption: "If ranking is enabled"},
	}, accent)

	drawSectionHeading(pdfDoc, "PROMOTION REVIEW", "", accent)
	finalDecision := data.FinalDecision
	if finalDecision == "" {
		finalDecision = defaultFinalDecision
	}
	recommendation := data.SystemRecommendation
	if recommendation == "" {
		recommendation = "Not yet evaluated"
	}
	drawLabelledRows(pdfDoc, [][2]string{
		{"Current Class", data.Class.Name},
		{"Proposed Next Class", orDash(data.ProposedNextClass)},
		{"Configured Policy Checks", orDash(data.PolicyChecksSummary)},
		{"Result Completeness", orDash(data.ResultCompleteness)},
		{"System Recommendation", recommendation + "  -  recommendation only; apply the school's approved policy."},
		{"Final Decision", finalDecision},
	}, 50)

	if data.Strengths != "" || data.FollowUp != "" || data.RemedialPlan != "" {
		drawSectionHeading(pdfDoc, "ACADEMIC REVIEW & RECOMMENDATIONS", "", accent)
		rows := [][2]string{}
		if data.Strengths != "" {
			rows = append(rows, [2]string{"Strengths", data.Strengths})
		}
		if data.FollowUp != "" {
			rows = append(rows, [2]string{"Follow-up", data.FollowUp})
		}
		if data.RemedialPlan != "" {
			rows = append(rows, [2]string{"Remedial Plan", data.RemedialPlan})
		}
		drawLabelledRows(pdfDoc, rows, 40)
	}

	drawSignatureLines(pdfDoc, []string{"Class Teacher", "Academic Approver", "School Head"})

	drawFooterDisclaimer(pdfDoc,
		"Promotion is not automatic from report generation. The authorized school decision and applicable policy determine progression.",
		fmt.Sprintf("Generated on %s", time.Now().Format("January 2, 2006")))

	var buf bytes.Buffer
	if err := pdfDoc.Output(&buf); err != nil {
		return nil, fmt.Errorf("render promotion report card pdf: %w", err)
	}
	return buf.Bytes(), nil
}
