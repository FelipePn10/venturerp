package gantt

import (
	"strings"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/aps/entity"
)

func sampleMonth() *entity.GanttMonth {
	loc := time.Local
	from := time.Date(2026, 6, 1, 0, 0, 0, 0, loc)
	to := from.AddDate(0, 1, 0)
	days := make([]entity.GanttDay, 0, 30)
	for d := 1; d <= 30; d++ {
		date := time.Date(2026, 6, d, 0, 0, 0, 0, loc)
		days = append(days, entity.GanttDay{
			Date:      date,
			Day:       d,
			Weekday:   date.Weekday(),
			IsWorkday: date.Weekday() != time.Saturday && date.Weekday() != time.Sunday,
			IsToday:   d == 15,
		})
	}
	bar := &entity.GanttBar{
		ProductionOrderID: 101, OrderNumber: 101, ItemCode: 1010,
		WorkCenterID: 7, WorkCenterName: "Corte", OperationName: "Serra",
		Start:  time.Date(2026, 6, 3, 8, 0, 0, 0, loc),
		End:    time.Date(2026, 6, 9, 17, 0, 0, 0, loc),
		Status: "SCHEDULED", Priority: "1", PercentComplete: 40,
		ColorHex: "#e67e22",
	}
	return &entity.GanttMonth{
		Year: 2026, Month: 6, RangeFrom: from, RangeTo: to,
		GroupBy: entity.GroupByWorkCenter,
		Days:    days,
		Rows: []*entity.GanttRow{
			{Key: "wc:7", ID: 7, Label: "Corte", Bars: []*entity.GanttBar{bar}},
		},
		Load: []*entity.GanttResourceLoad{
			{WorkCenterID: 7, Date: time.Date(2026, 6, 3, 0, 0, 0, 0, loc), LoadPct: 150, IsOverloaded: true},
		},
		Summary:     entity.GanttSummary{TotalRows: 1, TotalBars: 1, SequencedBars: 1},
		GeneratedAt: time.Date(2026, 6, 15, 10, 30, 0, 0, loc),
	}
}

func TestRenderSVG(t *testing.T) {
	data, ct, err := Render(sampleMonth(), "svg", Branding{CompanyName: "ACME S/A", GeneratedAt: time.Now()})
	if err != nil {
		t.Fatalf("render svg: %v", err)
	}
	if ct != "image/svg+xml" {
		t.Errorf("content type = %q", ct)
	}
	s := string(data)
	if !strings.HasPrefix(s, "<svg") || !strings.HasSuffix(s, "</svg>") {
		t.Errorf("output is not a well-formed svg envelope")
	}
	for _, want := range []string{"Corte", "ACME S/A", "Programação de Produção", "hoje", "#e67e22"} {
		if !strings.Contains(s, want) {
			t.Errorf("svg missing %q", want)
		}
	}
}

func TestRenderPDF(t *testing.T) {
	data, ct, err := Render(sampleMonth(), "pdf", Branding{CompanyName: "ACME S/A", GeneratedAt: time.Now()})
	if err != nil {
		t.Fatalf("render pdf: %v", err)
	}
	if ct != "application/pdf" {
		t.Errorf("content type = %q", ct)
	}
	if !strings.HasPrefix(string(data), "%PDF-") {
		t.Errorf("output is not a PDF (missing %%PDF- header)")
	}
	if len(data) < 500 {
		t.Errorf("pdf suspiciously small: %d bytes", len(data))
	}
}

func TestRenderEmptyMonthPDF(t *testing.T) {
	m := sampleMonth()
	m.Rows = nil
	data, _, err := Render(m, "pdf", Branding{})
	if err != nil {
		t.Fatalf("render empty pdf: %v", err)
	}
	if !strings.HasPrefix(string(data), "%PDF-") {
		t.Errorf("empty board should still produce a valid PDF")
	}
}

func TestRenderUnsupportedFormat(t *testing.T) {
	if _, _, err := Render(sampleMonth(), "xlsx", Branding{}); err == nil {
		t.Error("xlsx must be rejected")
	}
}

func TestRenderSVG_WithDependencies(t *testing.T) {
	loc := time.Local
	from := time.Date(2026, 6, 1, 0, 0, 0, 0, loc)
	to := from.AddDate(0, 1, 0)
	days := make([]entity.GanttDay, 0, 30)
	for d := 1; d <= 30; d++ {
		date := time.Date(2026, 6, d, 0, 0, 0, 0, loc)
		days = append(days, entity.GanttDay{Date: date, End: date.AddDate(0, 0, 1), Day: d, Weekday: date.Weekday(), IsWorkday: true})
	}
	b1 := &entity.GanttBar{SequenceID: 1, OrderNumber: 101, WorkCenterID: 7, WorkCenterName: "Corte",
		Start: time.Date(2026, 6, 3, 8, 0, 0, 0, loc), End: time.Date(2026, 6, 4, 8, 0, 0, 0, loc), ColorHex: "#2f6fb0"}
	b2 := &entity.GanttBar{SequenceID: 2, OrderNumber: 101, WorkCenterID: 8, WorkCenterName: "Solda",
		Start: time.Date(2026, 6, 5, 8, 0, 0, 0, loc), End: time.Date(2026, 6, 6, 8, 0, 0, 0, loc), ColorHex: "#2f6fb0"}
	m := &entity.GanttMonth{
		Year: 2026, Month: 6, RangeFrom: from, RangeTo: to, Scale: entity.ScaleDay,
		GroupBy: entity.GroupByWorkCenter, Days: days,
		Rows: []*entity.GanttRow{
			{Key: "wc:7", ID: 7, Label: "Corte", Bars: []*entity.GanttBar{b1}},
			{Key: "wc:8", ID: 8, Label: "Solda", Bars: []*entity.GanttBar{b2}},
		},
		Dependencies: []entity.GanttDependency{{FromSequenceID: 1, ToSequenceID: 2, Implicit: true}},
	}
	data, _, err := Render(m, "svg", Branding{})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(string(data), "<path") {
		t.Error("dependency connector (a <path>) should be drawn")
	}
	if _, _, err := Render(m, "pdf", Branding{}); err != nil {
		t.Fatalf("render pdf with deps: %v", err)
	}
}

func TestRenderWeekScale(t *testing.T) {
	loc := time.Local
	from := time.Date(2026, 6, 1, 0, 0, 0, 0, loc)
	to := from.AddDate(0, 0, 28)
	cols := make([]entity.GanttDay, 0, 4)
	for i := 0; i < 4; i++ {
		ws := from.AddDate(0, 0, i*7)
		_, wk := ws.ISOWeek()
		cols = append(cols, entity.GanttDay{Date: ws, End: ws.AddDate(0, 0, 7), Day: wk, Weekday: ws.Weekday(), IsWorkday: true, Label: ws.Format("02/01")})
	}
	m := &entity.GanttMonth{
		RangeFrom: from, RangeTo: to, Scale: entity.ScaleWeek, GroupBy: entity.GroupByWorkCenter,
		Days: cols,
		Rows: []*entity.GanttRow{{Key: "wc:7", ID: 7, Label: "Corte"}},
	}
	data, _, err := Render(m, "svg", Branding{})
	if err != nil {
		t.Fatalf("render week svg: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "semanal") {
		t.Error("week-scale title should read 'semanal'")
	}
	if !strings.Contains(s, cols[0].Label) {
		t.Errorf("week column label %q should appear", cols[0].Label)
	}
}

func TestParseHex(t *testing.T) {
	c := parseHex("#e67e22")
	if c.R != 0xe6 || c.G != 0x7e || c.B != 0x22 {
		t.Errorf("parseHex(#e67e22) = %+v", c)
	}
	def := parseHex("garbage")
	if def.R != 0x1f || def.G != 0x3a || def.B != 0x5f {
		t.Errorf("parseHex fallback = %+v", def)
	}
}
