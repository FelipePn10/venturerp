package notification_uc

import (
	"testing"
	"time"
)

func TestNextDigestAfterHandlesDST(t *testing.T) {
	before := time.Date(2026, 11, 1, 5, 30, 0, 0, time.UTC)
	next, err := NextDigestAfter(before, "08:00", "America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	loc, _ := time.LoadLocation("America/New_York")
	if got := next.In(loc).Format("2006-01-02 15:04 MST"); got != "2026-11-01 08:00 EST" {
		t.Fatalf("horário DST incorreto: %s", got)
	}
}
