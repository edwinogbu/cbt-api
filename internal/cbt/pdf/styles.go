package pdf

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"cbt-api/internal/cbt/dto"

	"github.com/go-pdf/fpdf"
)

// This file holds the shared, formal design system every report-card
// PDF (Mid-Term, Main Termly, Promotion) is built from - predominantly
// white/black/gray, with each school's OWN configured brand color
// (models.School.PrimaryColor) as the only accent, used sparingly
// (title bars and section headings), per the explicit instruction that
// these are real academic records schools print and sign, not
// colourful dashboards. A school with no configured color gets a
// neutral default navy, never another school's color and never a
// platform-wide "house" color.

const (
	pageWidthMM   = 210.0 // A4
	marginMM      = 14.0
	usableWidthMM = pageWidthMM - 2*marginMM
)

var (
	colorTextDark  = [3]int{17, 24, 39}    // near-black body text
	colorTextMuted = [3]int{100, 100, 100} // captions, muted labels
	colorBorder    = [3]int{200, 203, 209} // thin table/box borders
	colorPaleBg    = [3]int{240, 243, 247} // label cells, alternating rows, sub-bar
	colorWhite     = [3]int{255, 255, 255}
	defaultAccent  = [3]int{26, 58, 92} // neutral navy when a school has no configured PrimaryColor
)

// resolveAccent returns the school's own configured brand color
// (hex, e.g. "#1a3a5c") as RGB, or the neutral default navy if unset
// or malformed - never another school's color, never a hardcoded
// platform "house" color standing in for every tenant.
func resolveAccent(hex string) [3]int {
	r, g, b, ok := parseHexColor(hex)
	if !ok {
		return defaultAccent
	}
	return [3]int{r, g, b}
}

func parseHexColor(hex string) (r, g, b int, ok bool) {
	hex = stripLeadingHash(hex)
	if len(hex) != 6 {
		return 0, 0, 0, false
	}
	rv, err1 := strconv.ParseInt(hex[0:2], 16, 0)
	gv, err2 := strconv.ParseInt(hex[2:4], 16, 0)
	bv, err3 := strconv.ParseInt(hex[4:6], 16, 0)
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, 0, false
	}
	return int(rv), int(gv), int(bv), true
}

func stripLeadingHash(s string) string {
	if len(s) > 0 && s[0] == '#' {
		return s[1:]
	}
	return s
}

func setFill(pdfDoc *fpdf.Fpdf, c [3]int)  { pdfDoc.SetFillColor(c[0], c[1], c[2]) }
func setText(pdfDoc *fpdf.Fpdf, c [3]int)  { pdfDoc.SetTextColor(c[0], c[1], c[2]) }
func setDraw(pdfDoc *fpdf.Fpdf, c [3]int)  { pdfDoc.SetDrawColor(c[0], c[1], c[2]) }

// newReportPDF creates the shared A4 document every report type
// starts from.
func newReportPDF() *fpdf.Fpdf {
	pdfDoc := fpdf.New("P", "mm", "A4", "")
	pdfDoc.SetMargins(marginMM, marginMM, marginMM)
	pdfDoc.SetAutoPageBreak(true, marginMM)
	pdfDoc.AddPage()
	setDraw(pdfDoc, colorBorder)
	pdfDoc.SetLineWidth(0.2)
	return pdfDoc
}

// drawSchoolHeader renders the shared masthead: a logo (the school's
// own uploaded image, fetched best-effort, or a plain placeholder box
// when none is configured or it can't be reached - this must never
// fail or hang PDF generation, since a LAN school-node server may have
// no path to a cloud-hosted logo URL at all), the school's name,
// address and motto, all centered, in plain black/gray text - no
// colored background block, matching the reference template's
// restrained masthead rather than a colorful banner.
func drawSchoolHeader(pdfDoc *fpdf.Fpdf, school dto.SchoolInfo, motto string) {
	const logoSize = 20.0
	logoX, logoY := marginMM, marginMM

	drewLogo := false
	if school.Logo != "" {
		if imgBytes, imgType, err := fetchLogoBytes(school.Logo); err == nil {
			imgName := "school-logo-" + school.ID
			pdfDoc.RegisterImageOptionsReader(imgName, fpdf.ImageOptions{ImageType: imgType, ReadDpi: true}, bytes.NewReader(imgBytes))
			if pdfDoc.Ok() {
				pdfDoc.ImageOptions(imgName, logoX, logoY, logoSize, logoSize, false, fpdf.ImageOptions{ImageType: imgType}, 0, "")
				drewLogo = pdfDoc.Ok()
			}
		}
	}
	if !drewLogo {
		setDraw(pdfDoc, colorBorder)
		setFill(pdfDoc, colorPaleBg)
		pdfDoc.Rect(logoX, logoY, logoSize, logoSize, "DF")
		pdfDoc.SetFont("helvetica", "", 6.5)
		setText(pdfDoc, colorTextMuted)
		pdfDoc.SetXY(logoX, logoY+logoSize/2-3)
		pdfDoc.CellFormat(logoSize, 3, "LOGO", "", 2, "C", false, 0, "")
		pdfDoc.CellFormat(logoSize, 3, "PLACEHOLDER", "", 0, "C", false, 0, "")
	}

	pdfDoc.SetXY(0, marginMM+1)
	pdfDoc.SetFont("helvetica", "B", 17)
	setText(pdfDoc, colorTextDark)
	pdfDoc.CellFormat(pageWidthMM, 8, school.Name, "", 1, "C", false, 0, "")

	detailLine := school.Address
	pdfDoc.SetFont("helvetica", "", 9)
	setText(pdfDoc, colorTextMuted)
	pdfDoc.SetXY(0, pdfDoc.GetY())
	pdfDoc.CellFormat(pageWidthMM, 5, detailLine, "", 1, "C", false, 0, "")

	if motto != "" {
		pdfDoc.SetFont("helvetica", "I", 8.5)
		pdfDoc.CellFormat(pageWidthMM, 5, "\""+motto+"\"", "", 1, "C", false, 0, "")
	}

	y := pdfDoc.GetY()
	if y < marginMM+logoSize+3 {
		y = marginMM + logoSize + 3
	}
	pdfDoc.SetXY(marginMM, y+2)
}

// fetchLogoBytes retrieves a school's configured logo over HTTP with a
// short timeout - deliberately best-effort: a report card must never
// fail or hang because an external/cloud logo URL is unreachable from
// an offline LAN school-node server.
func fetchLogoBytes(url string) ([]byte, string, error) {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("logo fetch: unexpected status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20)) // 5MB cap
	if err != nil {
		return nil, "", err
	}
	contentType := http.DetectContentType(data)
	switch contentType {
	case "image/png":
		return data, "PNG", nil
	case "image/jpeg":
		return data, "JPG", nil
	case "image/gif":
		return data, "GIF", nil
	default:
		return nil, "", fmt.Errorf("logo fetch: unsupported image type %q", contentType)
	}
}

// drawTitleBar renders the full-width accent-colored report-title bar
// (the ONE deliberately bold use of color on the page, matching the
// reference) and the pale sub-bar beneath it carrying the
// term/session line.
func drawTitleBar(pdfDoc *fpdf.Fpdf, title, subtitle string, accent [3]int) {
	y := pdfDoc.GetY()
	setFill(pdfDoc, accent)
	pdfDoc.Rect(marginMM, y, usableWidthMM, 9, "F")
	pdfDoc.SetFont("helvetica", "B", 12)
	setText(pdfDoc, colorWhite)
	pdfDoc.SetXY(marginMM, y+1.5)
	pdfDoc.CellFormat(usableWidthMM, 6, title, "", 1, "C", false, 0, "")

	y2 := pdfDoc.GetY()
	setFill(pdfDoc, colorPaleBg)
	pdfDoc.Rect(marginMM, y2, usableWidthMM, 6, "F")
	pdfDoc.SetFont("helvetica", "", 8)
	setText(pdfDoc, colorTextMuted)
	pdfDoc.SetXY(marginMM+2, y2+1)
	pdfDoc.CellFormat(usableWidthMM-4, 4, subtitle, "", 1, "L", false, 0, "")
	pdfDoc.SetXY(marginMM, pdfDoc.GetY()+3)
}

// drawKeyValueGrid renders the shared student-identification block: a
// bordered grid of label/value pairs, `cols` per row, bold dark values
// under small muted uppercase-ish labels, with a pale background only
// behind the label line (matching the reference's identification
// table) rather than a solid colored fill across the whole cell.
func drawKeyValueGrid(pdfDoc *fpdf.Fpdf, rows [][2]string, cols int) {
	colW := usableWidthMM / float64(cols)
	const rowH = 12.0
	startY := pdfDoc.GetY()
	rowY := startY

	for i, field := range rows {
		col := i % cols
		if col == 0 && i > 0 {
			rowY += rowH
		}
		x := marginMM + float64(col)*colW
		pdfDoc.SetXY(x, rowY)
		pdfDoc.SetFont("helvetica", "", 7.5)
		setText(pdfDoc, colorTextMuted)
		pdfDoc.CellFormat(colW-2, 4, field[0], "", 0, "L", false, 0, "")
		pdfDoc.SetXY(x, rowY+4.2)
		pdfDoc.SetFont("helvetica", "B", 9.5)
		setText(pdfDoc, colorTextDark)
		pdfDoc.CellFormat(colW-2, 5, field[1], "", 0, "L", false, 0, "")
	}
	rowCount := (len(rows) + cols - 1) / cols
	totalH := float64(rowCount) * rowH
	setDraw(pdfDoc, colorBorder)
	pdfDoc.Rect(marginMM, startY, usableWidthMM, totalH, "D")
	for c := 1; c < cols; c++ {
		x := marginMM + float64(c)*colW
		pdfDoc.Line(x, startY, x, startY+totalH)
	}
	for r := 1; r < rowCount; r++ {
		y := startY + float64(r)*rowH
		pdfDoc.Line(marginMM, y, marginMM+usableWidthMM, y)
	}
	pdfDoc.SetXY(marginMM, startY+totalH+4)
}

// drawSectionHeading renders a bold, accent-colored section title with
// an optional small italic muted subtitle beneath it.
func drawSectionHeading(pdfDoc *fpdf.Fpdf, title, subtitle string, accent [3]int) {
	pdfDoc.SetFont("helvetica", "B", 10.5)
	setText(pdfDoc, accent)
	pdfDoc.CellFormat(usableWidthMM, 5.5, title, "", 1, "L", false, 0, "")
	if subtitle != "" {
		pdfDoc.SetFont("helvetica", "I", 7.5)
		setText(pdfDoc, colorTextMuted)
		pdfDoc.CellFormat(usableWidthMM, 4, subtitle, "", 1, "L", false, 0, "")
	}
	pdfDoc.Ln(1)
}

// statBox is one figure in the shared four-up summary strip (TOTAL
// OBTAINED / OVERALL AVERAGE / SUBJECTS PASSED / CLASS POSITION in the
// reference) - AccentValue marks the single figure allowed to use the
// school's accent color (the overall average, in the reference),
// every other value stays plain black.
type statBox struct {
	Label, Value, Caption string
	AccentValue           bool
}

func drawStatBoxes(pdfDoc *fpdf.Fpdf, boxes []statBox, accent [3]int) {
	colW := usableWidthMM / float64(len(boxes))
	startY := pdfDoc.GetY()
	const boxH = 16.0
	setDraw(pdfDoc, colorBorder)
	setFill(pdfDoc, colorPaleBg)
	pdfDoc.Rect(marginMM, startY, usableWidthMM, boxH, "DF")
	for i := 1; i < len(boxes); i++ {
		x := marginMM + float64(i)*colW
		pdfDoc.Line(x, startY, x, startY+boxH)
	}
	for i, b := range boxes {
		x := marginMM + float64(i)*colW
		pdfDoc.SetXY(x+2, startY+1.5)
		pdfDoc.SetFont("helvetica", "B", 7)
		setText(pdfDoc, colorTextMuted)
		pdfDoc.CellFormat(colW-4, 4, b.Label, "", 2, "L", false, 0, "")
		pdfDoc.SetXY(x+2, startY+6)
		valueSize := fitFontSize(pdfDoc, b.Value, colW-4, 13)
		pdfDoc.SetFont("helvetica", "B", valueSize)
		if b.AccentValue {
			setText(pdfDoc, accent)
		} else {
			setText(pdfDoc, colorTextDark)
		}
		pdfDoc.CellFormat(colW-4, 6, b.Value, "", 2, "L", false, 0, "")
		pdfDoc.SetXY(x+2, startY+12.5)
		pdfDoc.SetFont("helvetica", "I", 6.5)
		setText(pdfDoc, colorTextMuted)
		pdfDoc.CellFormat(colW-4, 3, b.Caption, "", 0, "L", false, 0, "")
	}
	pdfDoc.SetXY(marginMM, startY+boxH+4)
}

// fitFontSize shrinks a bold-helvetica font size (down to a floor of
// 7pt) until s fits within maxWidth - stat-box values are usually a
// short number or percentage sized generously at maxSize, but a few
// (e.g. a homework-habit note) are a short phrase that would otherwise
// overflow its box at that size.
func fitFontSize(pdfDoc *fpdf.Fpdf, s string, maxWidth, maxSize float64) float64 {
	const floor = 7.0
	for size := maxSize; size >= floor; size -= 0.5 {
		pdfDoc.SetFont("helvetica", "B", size)
		if pdfDoc.GetStringWidth(s) <= maxWidth {
			return size
		}
	}
	return floor
}

// drawLabelledRows renders the shared "label | text" block used for
// attendance figures, teacher/head comments, promotion-review fields
// and academic recommendations across all three report types - a bold
// label in a pale left column next to plain text, bordered, one row
// per entry, with MultiCell wrapping for longer text.
func drawLabelledRows(pdfDoc *fpdf.Fpdf, rows [][2]string, labelW float64) {
	textW := usableWidthMM - labelW
	for _, row := range rows {
		startY := pdfDoc.GetY()
		pdfDoc.SetFont("helvetica", "", 8.5)
		setText(pdfDoc, colorTextDark)
		pdfDoc.SetXY(marginMM+labelW+2, startY+1.5)
		pdfDoc.MultiCell(textW-4, 4.2, row[1], "", "L", false)
		endY := pdfDoc.GetY()
		rowH := endY - startY
		if rowH < 9 {
			rowH = 9
		}

		setDraw(pdfDoc, colorBorder)
		setFill(pdfDoc, colorPaleBg)
		pdfDoc.Rect(marginMM, startY, labelW, rowH, "DF")
		pdfDoc.Rect(marginMM+labelW, startY, textW, rowH, "D")
		pdfDoc.SetFont("helvetica", "B", 8)
		setText(pdfDoc, colorTextDark)
		pdfDoc.SetXY(marginMM+2, startY+2)
		pdfDoc.MultiCell(labelW-4, 4, row[0], "", "L", false)

		pdfDoc.SetXY(marginMM, startY+rowH)
	}
	pdfDoc.Ln(2)
}

// drawSignatureLines renders N evenly-spaced blank signature lines
// with their role labels beneath - 2 for the Mid-Term card, 3 for Main
// Termly and Promotion, per the reference.
func drawSignatureLines(pdfDoc *fpdf.Fpdf, labels []string) {
	y := pdfDoc.GetY() + 12
	if y > 270 { // keep signatures on the page rather than spilling onto a near-empty next one
		y = 270
	}
	colW := usableWidthMM / float64(len(labels))
	setDraw(pdfDoc, colorTextMuted)
	for i, label := range labels {
		x0 := marginMM + float64(i)*colW + 6
		x1 := marginMM + float64(i+1)*colW - 6
		pdfDoc.Line(x0, y, x1, y)
		pdfDoc.SetXY(marginMM+float64(i)*colW, y+1.5)
		pdfDoc.SetFont("helvetica", "B", 7.5)
		setText(pdfDoc, colorTextDark)
		pdfDoc.CellFormat(colW, 4, label, "", 0, "C", false, 0, "")
	}
	pdfDoc.SetY(y + 8)
}

// drawFooterDisclaimer renders the shared bottom-of-page disclaimer
// text, small, italic and muted - never a decorative footer.
func drawFooterDisclaimer(pdfDoc *fpdf.Fpdf, lines ...string) {
	pdfDoc.SetFont("helvetica", "I", 7)
	setText(pdfDoc, colorTextMuted)
	for _, line := range lines {
		pdfDoc.CellFormat(usableWidthMM, 4, line, "", 1, "C", false, 0, "")
	}
}
