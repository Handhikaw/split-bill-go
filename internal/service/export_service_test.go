package service

import (
	"bytes"
	"strings"
	"testing"
)

func sampleExportDoc() exportDoc {
	return exportDoc{
		Title:    "Liburan Bali",
		Kind:     "Group Bill",
		Status:   "ACTIVE",
		Currency: "IDR",
		Summary: []exportSummary{
			{Label: "Total", Value: "IDR 150000"},
			{Label: "Plan", Value: "premium"},
		},
		Sections: []exportSection{
			{
				Title:   "Peserta",
				Headers: []string{"Nama", "Dibayar", "Utang", "Net"},
				Widths:  []float64{60, 40, 40, 40},
				Rows: [][]string{
					{"Budi", "IDR 100000", "IDR 50000", "IDR 50000"},
					{"Ani", "IDR 0", "IDR 50000", "-IDR 50000"},
				},
			},
		},
	}
}

func TestRenderCSV(t *testing.T) {
	data, err := renderCSV(sampleExportDoc())
	if err != nil {
		t.Fatalf("renderCSV error: %v", err)
	}
	text := string(data)
	for _, want := range []string{"Liburan Bali", "Peserta", "Budi", "IDR 100000"} {
		if !strings.Contains(text, want) {
			t.Errorf("CSV tidak mengandung %q", want)
		}
	}
	if !bytes.HasPrefix(data, []byte{0xEF, 0xBB, 0xBF}) {
		t.Error("CSV harus diawali UTF-8 BOM")
	}
}

func TestRenderPDF(t *testing.T) {
	data, err := renderPDF(sampleExportDoc())
	if err != nil {
		t.Fatalf("renderPDF error: %v", err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Error("output bukan PDF")
	}
	if len(data) < 1000 {
		t.Errorf("PDF terlalu kecil: %d bytes", len(data))
	}
}
