package service

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/harkandavez/split-bill-go/internal/model"
	"github.com/harkandavez/split-bill-go/internal/model/dto"
	"github.com/harkandavez/split-bill-go/internal/repository"
	"gorm.io/gorm"
)

type ExportFile struct {
	Filename    string
	ContentType string
	Data        []byte
}

type ExportService interface {
	Export(publicID string, viewerUserID *int64, format string) (ExportFile, error)
}

type exportServiceImpl struct {
	repoBill    repository.BillRepository
	billService BillService
	goalService GoalService
}

func NewExportService(
	repoBill repository.BillRepository,
	billService BillService,
	goalService GoalService,
) ExportService {
	return &exportServiceImpl{
		repoBill:    repoBill,
		billService: billService,
		goalService: goalService,
	}
}

type exportSummary struct {
	Label string
	Value string
}

type exportSection struct {
	Title   string
	Headers []string
	Widths  []float64
	Rows    [][]string
}

type exportDoc struct {
	Title    string
	Kind     string
	Status   string
	Currency string
	Summary  []exportSummary
	Sections []exportSection
}

func (s *exportServiceImpl) Export(publicID string, viewerUserID *int64, format string) (ExportFile, error) {
	bill, err := s.repoBill.GetBillByPublicID(publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ExportFile{}, model.NewNotFound("Bill tidak ditemukan")
		}
		return ExportFile{}, model.NewInternalError("Database error")
	}

	var doc exportDoc
	var premium bool

	switch bill.Kind {
	case "goal":
		data, err := s.goalService.GetGoalBill(publicID, viewerUserID)
		if err != nil {
			return ExportFile{}, err
		}
		premium = data.Plan == "premium" || data.ViewerIsPremium
		doc = buildGoalDoc(data)
	case "split":
		data, err := s.billService.GetSplitBill(publicID, viewerUserID)
		if err != nil {
			return ExportFile{}, err
		}
		premium = data.Plan == "premium" || data.ViewerIsPremium
		doc = buildSplitDoc(*data)
	default:
		data, err := s.billService.GetGroupBill(publicID, viewerUserID)
		if err != nil {
			return ExportFile{}, err
		}
		premium = data.Plan == "premium" || data.ViewerIsPremium
		doc = buildGroupDoc(data)
	}

	if !premium {
		return ExportFile{}, model.NewBadRequest("fitur export khusus Premium; upgrade ke Pro")
	}

	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		format = "csv"
	}

	var data []byte
	contentType := "text/csv; charset=utf-8"
	switch format {
	case "csv":
		data, err = renderCSV(doc)
	case "pdf":
		data, err = renderPDF(doc)
		contentType = "application/pdf"
	default:
		return ExportFile{}, model.NewBadRequest("format harus csv atau pdf")
	}
	if err != nil {
		return ExportFile{}, model.NewInternalError("Gagal membuat file export")
	}

	return ExportFile{
		Filename:    fmt.Sprintf("splitkita-%s.%s", publicID, format),
		ContentType: contentType,
		Data:        data,
	}, nil
}

func moneyLabel(currency, amount string) string {
	return strings.TrimSpace(currency + " " + amount)
}

func fmtTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("02 Jan 2006 15:04")
}

func dash(v string) string {
	if strings.TrimSpace(v) == "" {
		return "-"
	}
	return v
}

func noteOrDash(p *string) string {
	if p == nil || strings.TrimSpace(*p) == "" {
		return "-"
	}
	return *p
}

func participantNames(parts []dto.ExpenseItemParticipant) string {
	if len(parts) == 0 {
		return "-"
	}
	names := make([]string, 0, len(parts))
	for _, p := range parts {
		names = append(names, p.Name)
	}
	return strings.Join(names, ", ")
}

func buildGroupDoc(d dto.GetGroupBillRs) exportDoc {
	doc := exportDoc{
		Title:    d.Name,
		Kind:     "Group Bill",
		Status:   d.Status,
		Currency: d.Currency,
		Summary: []exportSummary{
			{Label: "Total", Value: moneyLabel(d.Currency, d.TotalAmount)},
			{Label: "Plan", Value: d.Plan},
			{Label: "Peserta", Value: fmt.Sprintf("%d", len(d.Participants))},
			{Label: "Transaksi", Value: fmt.Sprintf("%d", len(d.Expense))},
		},
	}

	partRows := make([][]string, 0, len(d.Participants))
	for _, p := range d.Participants {
		partRows = append(partRows, []string{
			p.Name,
			moneyLabel(d.Currency, p.TotalPaid),
			moneyLabel(d.Currency, p.TotalOwed),
			moneyLabel(d.Currency, p.NetBalance),
		})
	}
	doc.Sections = append(doc.Sections, exportSection{
		Title: "Peserta", Headers: []string{"Nama", "Dibayar", "Utang", "Net"},
		Widths: []float64{60, 40, 40, 40}, Rows: partRows,
	})

	expRows := make([][]string, 0, len(d.Expense))
	itemRows := [][]string{}
	for _, e := range d.Expense {
		expRows = append(expRows, []string{
			fmtTime(e.Created_At), e.Title, dash(e.Category),
			moneyLabel(d.Currency, e.Amount), dash(e.Created_By),
		})
		if e.Items != nil {
			for _, it := range *e.Items {
				itemRows = append(itemRows, []string{
					e.Title, it.Name, fmt.Sprintf("%d", it.Quantity),
					moneyLabel(d.Currency, it.Price), moneyLabel(d.Currency, it.Subtotal),
					participantNames(it.Participants),
				})
			}
		}
	}
	doc.Sections = append(doc.Sections, exportSection{
		Title: "Pengeluaran", Headers: []string{"Tanggal", "Judul", "Kategori", "Nominal", "Pembayar"},
		Widths: []float64{30, 55, 30, 30, 35}, Rows: expRows,
	})
	if len(itemRows) > 0 {
		doc.Sections = append(doc.Sections, exportSection{
			Title: "Item", Headers: []string{"Expense", "Item", "Qty", "Harga", "Subtotal", "Untuk"},
			Widths: []float64{30, 45, 10, 25, 30, 40}, Rows: itemRows,
		})
	}

	settleRows := make([][]string, 0, len(d.Settlements))
	for _, s := range d.Settlements {
		settleRows = append(settleRows, []string{s.From.Name, s.To.Name, moneyLabel(d.Currency, s.Amount)})
	}
	doc.Sections = append(doc.Sections, exportSection{
		Title: "Sisa Utang", Headers: []string{"Dari", "Ke", "Nominal"},
		Widths: []float64{70, 70, 40}, Rows: settleRows,
	})

	payRows := make([][]string, 0, len(d.SettlementPayments))
	for _, p := range d.SettlementPayments {
		payRows = append(payRows, []string{
			fmtTime(p.PaidAt), p.From.Name, p.To.Name,
			moneyLabel(d.Currency, p.Amount), noteOrDash(p.Note),
		})
	}
	doc.Sections = append(doc.Sections, exportSection{
		Title: "Riwayat Pelunasan", Headers: []string{"Tanggal", "Dari", "Ke", "Nominal", "Catatan"},
		Widths: []float64{30, 35, 35, 35, 45}, Rows: payRows,
	})

	return doc
}

func buildSplitDoc(d dto.GetSplitBillRs) exportDoc {
	doc := exportDoc{
		Title:    d.Name,
		Kind:     "Split Bill",
		Status:   d.Status,
		Currency: d.Currency,
		Summary: []exportSummary{
			{Label: "Total", Value: moneyLabel(d.Currency, d.TotalAmount)},
			{Label: "Pajak", Value: moneyLabel(d.Currency, d.Tax)},
			{Label: "Service", Value: moneyLabel(d.Currency, d.ServiceCharge)},
			{Label: "Diskon", Value: moneyLabel(d.Currency, d.Discount)},
			{Label: "Plan", Value: d.Plan},
		},
	}

	partRows := make([][]string, 0, len(d.Participants))
	for _, p := range d.Participants {
		partRows = append(partRows, []string{
			p.Name,
			moneyLabel(d.Currency, p.TotalPaid),
			moneyLabel(d.Currency, p.TotalOwed),
			moneyLabel(d.Currency, p.NetBalance),
		})
	}
	doc.Sections = append(doc.Sections, exportSection{
		Title: "Peserta", Headers: []string{"Nama", "Dibayar", "Utang", "Net"},
		Widths: []float64{60, 40, 40, 40}, Rows: partRows,
	})

	itemRows := make([][]string, 0, len(d.Items))
	for _, it := range d.Items {
		itemRows = append(itemRows, []string{
			it.Name, fmt.Sprintf("%d", it.Quantity), moneyLabel(d.Currency, it.Price),
			moneyLabel(d.Currency, it.Subtotal), participantNames(it.Participants),
		})
	}
	doc.Sections = append(doc.Sections, exportSection{
		Title: "Item", Headers: []string{"Item", "Qty", "Harga", "Subtotal", "Untuk"},
		Widths: []float64{45, 12, 35, 38, 50}, Rows: itemRows,
	})

	settleRows := make([][]string, 0, len(d.Settlements))
	for _, s := range d.Settlements {
		settleRows = append(settleRows, []string{s.From.Name, s.To.Name, moneyLabel(d.Currency, s.Amount)})
	}
	doc.Sections = append(doc.Sections, exportSection{
		Title: "Sisa Utang", Headers: []string{"Dari", "Ke", "Nominal"},
		Widths: []float64{70, 70, 40}, Rows: settleRows,
	})

	payRows := make([][]string, 0, len(d.SettlementPayments))
	for _, p := range d.SettlementPayments {
		payRows = append(payRows, []string{
			fmtTime(p.PaidAt), p.From.Name, p.To.Name,
			moneyLabel(d.Currency, p.Amount), noteOrDash(p.Note),
		})
	}
	doc.Sections = append(doc.Sections, exportSection{
		Title: "Riwayat Pelunasan", Headers: []string{"Tanggal", "Dari", "Ke", "Nominal", "Catatan"},
		Widths: []float64{30, 35, 35, 35, 45}, Rows: payRows,
	})

	return doc
}

func buildGoalDoc(d dto.GetGoalBillRs) exportDoc {
	doc := exportDoc{
		Title:    d.Name,
		Kind:     "Patungan",
		Status:   d.Status,
		Currency: d.Currency,
		Summary: []exportSummary{
			{Label: "Target", Value: moneyLabel(d.Currency, d.TargetAmount)},
			{Label: "Terkumpul", Value: moneyLabel(d.Currency, d.TotalCollected)},
			{Label: "Sisa", Value: moneyLabel(d.Currency, d.Remaining)},
			{Label: "Plan", Value: d.Plan},
			{Label: "Deskripsi", Value: noteOrDash(d.Description)},
		},
	}

	partRows := make([][]string, 0, len(d.Participants))
	for _, p := range d.Participants {
		partRows = append(partRows, []string{
			p.Name, moneyLabel(d.Currency, p.TotalContributed),
			fmt.Sprintf("%d", p.ContributionCount),
		})
	}
	doc.Sections = append(doc.Sections, exportSection{
		Title: "Peserta", Headers: []string{"Nama", "Kontribusi", "Jumlah"},
		Widths: []float64{80, 60, 40}, Rows: partRows,
	})

	contribRows := make([][]string, 0, len(d.Contributions))
	for _, c := range d.Contributions {
		contribRows = append(contribRows, []string{
			fmtTime(c.PaidAt), c.Name, moneyLabel(d.Currency, c.Amount), noteOrDash(c.Note),
		})
	}
	doc.Sections = append(doc.Sections, exportSection{
		Title: "Kontribusi", Headers: []string{"Tanggal", "Nama", "Nominal", "Catatan"},
		Widths: []float64{35, 45, 50, 50}, Rows: contribRows,
	})

	return doc
}

func renderCSV(doc exportDoc) ([]byte, error) {
	var buf bytes.Buffer
	// BOM agar Excel membaca UTF-8 dengan benar.
	buf.WriteString("\xEF\xBB\xBF")

	w := csv.NewWriter(&buf)
	write := func(row ...string) {
		_ = w.Write(row)
	}

	write("Bill", doc.Title)
	write("Tipe", doc.Kind)
	write("Status", doc.Status)
	write("Currency", doc.Currency)
	for _, s := range doc.Summary {
		write(s.Label, s.Value)
	}

	for _, sec := range doc.Sections {
		write("")
		write(sec.Title)
		if len(sec.Headers) > 0 {
			write(sec.Headers...)
		}
		if len(sec.Rows) == 0 {
			write("-")
			continue
		}
		for _, row := range sec.Rows {
			write(row...)
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func asciiText(s string) string {
	replacer := strings.NewReplacer(
		"→", "->",
		"–", "-",
		"—", "-",
		"·", "-",
		"’", "'",
		"“", `"`,
		"”", `"`,
		"\n", " ",
		"\r", " ",
	)
	s = replacer.Replace(s)
	var b strings.Builder
	for _, r := range s {
		if r <= 255 {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-3]) + "..."
}

func pdfEnsureSpace(pdf *fpdf.Fpdf, height float64) {
	if pdf.GetY()+height > 280 {
		pdf.AddPage()
	}
}

func pdfSection(pdf *fpdf.Fpdf, title string) {
	pdfEnsureSpace(pdf, 16)
	pdf.Ln(3)
	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(0, 7, asciiText(title), "", 1, "L", false, 0, "")
}

func pdfHeaders(pdf *fpdf.Fpdf, headers []string, widths []float64) {
	pdfEnsureSpace(pdf, 8)
	pdf.SetFont("Helvetica", "B", 8.5)
	for i, h := range headers {
		w := 36.0
		if i < len(widths) {
			w = widths[i]
		}
		pdf.CellFormat(w, 6, asciiText(truncate(h, int(w*1.7))), "B", 0, "L", false, 0, "")
	}
	pdf.Ln(6)
}

func pdfRow(pdf *fpdf.Fpdf, row []string, widths []float64) {
	pdfEnsureSpace(pdf, 7)
	pdf.SetFont("Helvetica", "", 8.5)
	for i, cell := range row {
		w := 36.0
		if i < len(widths) {
			w = widths[i]
		}
		pdf.CellFormat(w, 6, asciiText(truncate(cell, int(w*1.7))), "", 0, "L", false, 0, "")
	}
	pdf.Ln(6)
}

func renderPDF(doc exportDoc) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 15)
	pdf.SetTitle(asciiText(doc.Title), false)
	pdf.SetAuthor("SplitKita", false)
	pdf.AddPage()

	pdf.SetFont("Helvetica", "B", 17)
	pdf.CellFormat(0, 9, asciiText(doc.Title), "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(90, 90, 100)
	pdf.CellFormat(0, 6, asciiText(fmt.Sprintf("%s · %s · %s", doc.Kind, doc.Status, doc.Currency)), "", 1, "L", false, 0, "")
	pdf.SetTextColor(0, 0, 0)

	for _, s := range doc.Summary {
		pdfEnsureSpace(pdf, 7)
		pdf.SetFont("Helvetica", "B", 9)
		pdf.CellFormat(32, 6, asciiText(s.Label), "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 9)
		pdf.CellFormat(0, 6, asciiText(s.Value), "", 1, "L", false, 0, "")
	}

	for _, sec := range doc.Sections {
		pdfSection(pdf, sec.Title)
		if len(sec.Headers) > 0 {
			pdfHeaders(pdf, sec.Headers, sec.Widths)
		}
		if len(sec.Rows) == 0 {
			pdf.SetFont("Helvetica", "", 9)
			pdf.CellFormat(0, 6, "-", "", 1, "L", false, 0, "")
			continue
		}
		for _, row := range sec.Rows {
			pdfRow(pdf, row, sec.Widths)
		}
	}

	pdf.SetFooterFunc(func() {
		pdf.SetY(-12)
		pdf.SetFont("Helvetica", "", 8)
		pdf.SetTextColor(120, 120, 130)
		pdf.CellFormat(0, 6, asciiText(fmt.Sprintf("SplitKita - halaman %d", pdf.PageNo())), "", 0, "C", false, 0, "")
	})

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
